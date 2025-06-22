package main

import (
	"fmt"
	"os"

	"qbuploader/internal/config"
	"qbuploader/internal/logger"
)

func main() {
	if err := config.Init(); err != nil {
		// 在日志系统初始化前，只能用 fmt 打印
		fmt.Fprintf(os.Stderr, "[CRITICAL] 配置初始化失败！程序无法启动。\n           原因: %v\n", err)
		os.Exit(1)
	}

	logger.Init()

	log := logger.Log

	// --- 验证阶段 ---
	log.Info("qBittorrent Uploader v1.0.0 (alpha) 启动...")
	log.Debug("这是一条 DEBUG 信息，只有在 debug 模式下可见。")
	log.Info("配置文件加载成功！")
	log.Infof("将上传到网盘目录: %s", config.Cfg.Uploader.RemoteDir)
	log.Warn("这是一个警告信息，表示可能有问题，但程序可以继续。")
	log.Error("这是一个错误信息，表示发生了严重问题。\n         这第二行是错误的详细解释，\n         第三行也是。")

	// TODO: 初始化数据库模块，并根据命令行参数选择运行模式
}