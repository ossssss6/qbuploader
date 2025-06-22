package scheduler

import (
	"fmt"
	"os"
	"path/filepath"

	"qbuploader/internal/baidupcs"
	"qbuploader/internal/config"
	"qbuploader/internal/database"
	"qbuploader/internal/logger"

	"github.com/autobrr/go-qbittorrent"
)

var log = logger.Log

// (RunUploadMode 函数保持不变，这里省略)
func RunUploadMode(infoHash, torrentName, contentPath string) error {
	log.Info("===== 进入上传模式 =====")
	log.Infof("任务: %s (%s)", torrentName, infoHash)
	log.Infof("路径: %s", contentPath)

	log.Info("-> 正在数据库中登记任务...")
	if err := database.AddTask(infoHash, torrentName); err != nil {
		log.Errorf("数据库登记任务失败: %v", err)
		return err
	}
	database.UpdateTaskStatus(infoHash, "uploading", "开始上传")
	log.Info("-> 数据库登记完成。")

	uploader := baidupcs.NewUploader()
	err := uploader.Upload(contentPath, config.Cfg.Uploader.RemoteDir, torrentName)
	if err != nil {
		database.UpdateTaskStatus(infoHash, "failed", err.Error())
		log.Errorf("上传失败: %v", err)
		return err
	}

	database.UpdateTaskStatus(infoHash, "success", "上传成功")
	log.Info("[OK] 上传成功！") // 这里用 [OK] 很好，因为它是一个模式的最终成功标志
	log.Info("===== 上传模式执行完毕 =====")
	return nil
}

// RunCleanupMode 执行清理模式的逻辑
func RunCleanupMode() error {
	log.Info("===== 进入清理模式 =====")

	log.Info("-> 正在执行数据库维护...")
	rowsAffected, err := database.PruneOldTasks()
	if err != nil {
		log.Warnf("-> 数据库维护失败: %v", err)
	} else if rowsAffected > 0 {
		log.Infof("-> [OK] 成功清理了 %d 条过期的数据库记录。", rowsAffected)
	} else {
		log.Debug("-> 数据库无需清理。")
	}

	log.Info("-> 正在连接 qBittorrent...")
	qbConfig := qbittorrent.Config{
		Host:     config.Cfg.QBittorrent.Host,
		Username: config.Cfg.QBittorrent.Username,
		Password: config.Cfg.QBittorrent.Password,
	}
	qbClient := qbittorrent.NewClient(qbConfig)

	if err := qbClient.Login(); err != nil {
		return fmt.Errorf("登录 qBittorrent 失败: %w", err)
	}
	log.Info("-> [OK] 登录成功。")

	log.Info("-> 正在获取 qBittorrent 版本...")
	appVersion, err := qbClient.GetAppVersion()
	if err != nil {
		return fmt.Errorf("获取 qBittorrent 版本信息失败: %w", err)
	}
	log.Infof("-> [OK] 获取版本成功: %s", appVersion)

	log.Info("-> 正在获取任务列表与上传记录...")
	allTorrents, err := qbClient.GetTorrents(qbittorrent.TorrentFilterOptions{})
	if err != nil {
		return fmt.Errorf("获取 qBittorrent 任务列表失败: %w", err)
	}

	uploadedHashes, err := database.GetSuccessfulUploads()
	if err != nil {
		return fmt.Errorf("从数据库获取已上传列表失败: %w", err)
	}
	log.Infof("-> [OK] 数据获取完毕: %d 个 qB 任务, %d 条已上传记录。", len(allTorrents), len(uploadedHashes))

	log.Info("-> 正在筛选满足条件的任务...")
	var tasksToProcess []qbittorrent.Torrent
	for _, t := range allTorrents {
		policyMet := isPolicyMet(t)
		isUploaded := uploadedHashes[t.Hash]

		if policyMet && isUploaded {
			tasksToProcess = append(tasksToProcess, t)
			log.Infof("  -> 任务 '%s' 符合所有条件，已加入处理队列。", t.Name)
		}
	}

	if len(tasksToProcess) == 0 {
		log.Info("-> 没有需要处理的任务。")
		log.Info("===== 清理模式执行完毕 =====")
		return nil
	}

	log.Infof("-> 筛选完毕，共 %d 个任务待处理。", len(tasksToProcess))

	uploader := baidupcs.NewUploader()
	var hashesToDelete []string
	for i, t := range tasksToProcess {
		log.Infof("--> [ %d / %d ] 正在处理任务: %s", i+1, len(tasksToProcess), t.Name)

		log.Info("    -> 正在校验网盘文件...")
		exists, err := uploader.CheckFileExists(config.Cfg.Uploader.RemoteDir, t.Name)
		if err != nil {
			log.Warnf("    -> 网盘文件校验时发生错误: %v。为安全起见，跳过此任务。", err)
			continue
		}
		if !exists {
			log.Errorf("    -> [严重] 最终校验失败！网盘上未找到文件 '%s'。为安全起见，跳过此任务。", t.Name)
			continue
		}
		log.Info("    -> [OK] 校验成功！")

		contentPath := filepath.Join(t.SavePath, t.Name)
		log.Infof("    -> 正在删除本地文件: %s", contentPath)
		if err := os.RemoveAll(contentPath); err != nil {
			log.Errorf("    -> 删除本地文件失败: %v。跳过此任务。", err)
			continue
		}
		log.Infof("    -> [OK] 本地文件已删除。")

		hashesToDelete = append(hashesToDelete, t.Hash)
		database.ArchiveTask(t.Hash)
	}

	if len(hashesToDelete) > 0 {
		deleteFiles := false
		log.Infof("-> 准备从 qBittorrent 中批量删除 %d 个任务记录...", len(hashesToDelete))
		if err := qbClient.DeleteTorrents(hashesToDelete, deleteFiles); err != nil {
			log.Errorf("从 qBittorrent 删除任务失败: %v", err)
		} else {
			log.Info("-> [OK] 成功从 qBittorrent 移除任务。")
		}
	}

	log.Info("===== 清理模式执行完毕 =====")
	return nil
}

// isPolicyMet 判断单个任务是否满足保种策略
func isPolicyMet(t qbittorrent.Torrent) bool {
	cfg := config.Cfg.SeedingPolicy

	if cfg.TargetRatio > 0 && t.Ratio >= cfg.TargetRatio {
		return true
	}

	if cfg.TargetSeedingHours > 0 && t.SeedingTime >= int64(cfg.TargetSeedingHours*3600) {
		return true
	}

	return false
}
