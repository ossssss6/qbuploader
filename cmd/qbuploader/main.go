package main

import (
	"os"

	"qbuploader/internal/config"
	"qbuploader/internal/database"
	"qbuploader/internal/logger"
	"qbuploader/internal/qbittorrent" // 导入 qbittorrent 包
)

func main() {
	// 初始化所有基础模块...
	if err := config.Init(); err != nil {
		// ...
		os.Exit(1)
	}
	logger.Init()
	if err := database.Init(); err != nil {
		// ...
		os.Exit(1)
	}
	defer database.DB.Close()

	log := logger.Log
	log.Info("qBittorrent Uploader v1.0.0 (alpha) 启动...")

	// --- 新的测试代码 ---
	log.Info("正在尝试连接到 qBittorrent...")
	qbClient, err := qbittorrent.NewClient()
	if err != nil {
		log.Errorf("无法连接到 qBittorrent: %v", err)
		log.Errorf("请确保 qBittorrent 正在运行，Web UI 已开启，并且 config.ini 中的地址、用户名、密码正确。")
		os.Exit(1)
	}

	// 如果连接成功，尝试获取种子列表
	torrents, err := qbClient.GetTorrents()
	if err != nil {
		log.Errorf("获取种子列表失败: %v", err)
		os.Exit(1)
	}
	log.Infof("成功获取到种子列表！当前共有 %d 个任务。", len(torrents))

	log.Info("外部交互模块测试成功！")
}