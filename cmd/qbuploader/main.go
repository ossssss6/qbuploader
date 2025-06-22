package main

import (
	"fmt"
	"os"

	"qbuploader/internal/config"
)

func main() {
	// 1. 初始化配置
	if err := config.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "严重错误: 配置初始化失败！\n原因: %v\n", err)
		os.Exit(1)
	}

	// --- 验证阶段 ---
	// 打印一些从配置中读取到的值，来检查是否成功
	fmt.Println("配置加载成功！")
	fmt.Printf("日志模式将被设置为: %s\n", config.Cfg.General.LogLevel)
	fmt.Printf("将上传到网盘目录: %s\n", config.Cfg.Uploader.RemoteDir)
	fmt.Printf("保种分享率目标: %.2f\n", config.Cfg.SeedingPolicy.TargetRatio)

	// TODO: 初始化日志模块，并根据命令行参数选择运行模式
}
