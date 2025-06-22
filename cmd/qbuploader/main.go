package main

import (
	"fmt"
	"os"

	"qbuploader/internal/config"
	"qbuploader/internal/database"
	"qbuploader/internal/logger"
	"qbuploader/internal/scheduler"

	"github.com/urfave/cli/v2"
)

func main() {
	if err := config.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "[CRITICAL] 配置初始化失败！程序无法启动。\n           原因: %v\n", err)
		os.Exit(1)
	}
	logger.Init()
	if err := database.Init(); err != nil {
		logger.Log.Errorf("数据库初始化失败！程序无法启动。")
		logger.Log.Errorf("原因: %v", err)
		os.Exit(1)
	}
	defer database.DB.Close()
	log := logger.Log
	app := &cli.App{
		Name:    "qbuploader",
		Usage:   "qBittorrent 自动化保种与备份工具",
		Version: "1.0.0",
		Commands: []*cli.Command{
			{
				Name:      "upload",
				Usage:     "上传单个任务 (由 qB '任务完成时' 调用)",
				ArgsUsage: "<content_path> <torrent_name> <info_hash>",
				Action: func(c *cli.Context) error {
					if c.NArg() < 3 {
						return fmt.Errorf("upload 命令需要 3 个参数: content_path, torrent_name, info_hash")
					}
					contentPath := c.Args().Get(0)
					torrentName := c.Args().Get(1)
					infoHash := c.Args().Get(2)
					return scheduler.RunUploadMode(infoHash, torrentName, contentPath)
				},
			},
			{
				Name:  "cleanup",
				Usage: "执行定期巡检和清理 (由任务计划程序调用)",
				Action: func(c *cli.Context) error {
					return scheduler.RunCleanupMode()
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalf("程序执行出错: %v", err)
	}
}
