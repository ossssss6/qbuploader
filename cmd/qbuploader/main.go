package main

import (
	"fmt"
	"os"

	"qbuploader/internal/config"
	"qbuploader/internal/database"
	"qbuploader/internal/logger"
)

func main() {
	// 1. 初始化配置
	if err := config.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "[CRITICAL] 配置初始化失败！程序无法启动。\n           原因: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志
	logger.Init()
	log := logger.Log

	// 3. 初始化数据库
	if err := database.Init(); err != nil {
		log.Errorf("数据库初始化失败！程序无法启动。")
		log.Errorf("原因: %v", err)
		os.Exit(1)
	}
	// 程序退出前，安全地关闭数据库连接
	defer database.DB.Close()

	log.Info("qBittorrent Uploader v1.0.0 (alpha) 启动...")
	log.Info("配置、日志、数据库模块均已成功初始化！")
	log.Warn("程序当前没有实现任何功能，运行后会直接退出。")

	// TODO: 根据命令行参数选择运行模式
}