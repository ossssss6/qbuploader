package scheduler

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"qbuploader/internal/baidupcs"
	"qbuploader/internal/config"
	"qbuploader/internal/database"
	"qbuploader/internal/logger"

	"github.com/autobrr/go-qbittorrent" // <<<--- 【最终修正】修正了这里的拼写错误
)

var log = logger.Log

// RunUploadMode 函数...
func RunUploadMode(infoHash, torrentName, contentPath string) error {
	log.Infof("===== [Upload Mode] 任务: %s =====", torrentName)
	log.Debugf("InfoHash: %s, 本地路径: %s", infoHash, contentPath)
	task, err := getTaskByHash(infoHash)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("查询数据库失败: %w", err)
	}
	if task != nil && task.UploadStatus == "success" {
		log.Infof("-> 任务已在数据库中标记为上传成功，跳过本次上传。")
		return nil
	}
	log.Info("-> 正在登记任务并准备上传...")
	if err := addTask(infoHash, torrentName); err != nil {
		return fmt.Errorf("数据库登记任务失败: %w", err)
	}
	updateTaskStatus(infoHash, "uploading", "开始上传")
	uploader := baidupcs.NewUploader()
	err = uploader.Upload(contentPath, config.Cfg.Uploader.RemoteDir, torrentName)
	if err != nil {
		updateTaskStatus(infoHash, "failed", err.Error())
		return fmt.Errorf("上传失败: %w", err)
	}
	updateTaskStatus(infoHash, "success", "上传成功")
	log.Info("-> [OK] 上传成功！")
	log.Info("===== [Upload Mode] 执行完毕 =====")
	return nil
}

// RunCleanupMode 函数...
func RunCleanupMode() error {
	log.Info("===== [Cleanup Mode] 开始执行巡检 =====")

	log.Info("-> 正在执行数据库维护...")
	rowsAffected, err := pruneOldTasks()
	if err != nil {
		log.Warnf("-> 数据库维护失败: %v", err)
	} else if rowsAffected > 0 {
		log.Infof("-> [OK] 成功清理了 %d 条过期的数据库记录。", rowsAffected)
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

	log.Info("-> 正在获取任务列表与上传记录...")
	allTorrents, err := qbClient.GetTorrents(qbittorrent.TorrentFilterOptions{})
	if err != nil {
		return fmt.Errorf("获取 qB 任务列表失败: %w", err)
	}
	uploadedHashes, err := getTasksByStatus("success")
	if err != nil {
		return fmt.Errorf("从数据库获取已上传列表失败: %w", err)
	}
	log.Infof("-> [OK] 数据获取完毕: %d 个 qB 任务, %d 条已上传记录。", len(allTorrents), len(uploadedHashes))

	log.Info("-> 正在筛选满足清理条件的任务...")
	targetStates := strings.Split(config.Cfg.SeedingPolicy.Cleanup_Target_States, ",")
	stateMap := make(map[string]bool)
	for _, s := range targetStates {
		stateMap[strings.TrimSpace(s)] = true
	}
	var tasksToProcess []qbittorrent.Torrent
	for _, t := range allTorrents {
		if stateMap[string(t.State)] && uploadedHashes[t.Hash] {
			tasksToProcess = append(tasksToProcess, t)
			log.Infof("  -> 任务 '%s' 符合所有条件，已加入处理队列。", t.Name)
		}
	}

	if len(tasksToProcess) == 0 {
		log.Info("-> 没有需要处理的任务。")
		log.Info("===== [Cleanup Mode] 巡检完毕 =====")
		return nil
	}
	log.Infof("-> 筛选完毕，共 %d 个任务待处理。", len(tasksToProcess))
	uploader := baidupcs.NewUploader()
	var hashesToDeleteFromQB []string
	for i, t := range tasksToProcess {
		log.Infof("--> [ %d / %d ] 正在处理任务: %s", i+1, len(tasksToProcess), t.Name)
		log.Info("    -> 正在校验网盘文件...")
		exists, err := uploader.CheckFileExists(config.Cfg.Uploader.RemoteDir, t.Name)
		if err != nil {
			log.Warnf("    -> 网盘文件校验时发生错误，跳过此任务: %v", err)
			continue
		}
		if !exists {
			log.Errorf("    -> [严重] 最终校验失败！网盘上未找到文件 '%s'。为安全起见，将不会删除任何文件！", t.Name)
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
		hashesToDeleteFromQB = append(hashesToDeleteFromQB, t.Hash)
		archiveTask(t.Hash)
	}
	if len(hashesToDeleteFromQB) > 0 {
		deleteFiles := false
		log.Infof("-> 准备从 qBittorrent 中批量删除 %d 个任务记录...", len(hashesToDeleteFromQB))
		if err := qbClient.DeleteTorrents(hashesToDeleteFromQB, deleteFiles); err != nil {
			log.Errorf("-> 从 qBittorrent 删除任务失败: %v", err)
		} else {
			log.Info("-> [OK] 成功从 qBittorrent 移除任务。")
		}
	}
	log.Info("===== [Cleanup Mode] 巡检完毕 =====")
	return nil
}

// --- 数据库操作封装 ---
func addTask(infoHash, torrentName string) error {
	query := `INSERT OR IGNORE INTO tasks (info_hash, torrent_name) VALUES (?, ?)`
	_, err := database.DB.Exec(query, infoHash, torrentName)
	return err
}

func updateTaskStatus(infoHash, status, message string) error {
	query := `UPDATE tasks SET upload_status = ?, message = ? WHERE info_hash = ?`
	_, err := database.DB.Exec(query, status, message, infoHash)
	return err
}

func getTaskByHash(infoHash string) (*database.Task, error) {
	query := `SELECT info_hash, torrent_name, upload_status, message, created_at, updated_at FROM tasks WHERE info_hash = ?`
	row := database.DB.QueryRow(query, infoHash)
	var t database.Task
	err := row.Scan(&t.InfoHash, &t.TorrentName, &t.UploadStatus, &t.Message, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func getTasksByStatus(status string) (map[string]bool, error) {
	query := `SELECT info_hash FROM tasks WHERE upload_status = ?`
	rows, err := database.DB.Query(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hashes := make(map[string]bool)
	for rows.Next() {
		var infoHash string
		if err := rows.Scan(&infoHash); err != nil {
			return nil, err
		}
		hashes[infoHash] = true
	}
	return hashes, nil
}

func pruneOldTasks() (int64, error) {
	days := config.Cfg.Maintenance.DBKeepArchivedDays
	if days <= 0 {
		return 0, nil
	}
	query := `DELETE FROM tasks WHERE upload_status = 'archived' AND updated_at < date('now', '-' || ? || ' day')`
	result, err := database.DB.Exec(query, days)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func archiveTask(infoHash string) error {
	return updateTaskStatus(infoHash, "archived", "任务已完成并归档")
}
