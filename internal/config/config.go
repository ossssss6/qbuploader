package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

// Cfg 是一个全局变量，所有其他模块都可以通过它来访问配置。
var Cfg *Config

// Config 结构体是我们程序内部使用的“标准参数表”。
type Config struct {
	Uploader struct {
		Path      string
		RemoteDir string
		ExtraArgs []string
	}
	QBittorrent struct {
		Host     string
		Username string
		Password string
	}
	SeedingPolicy struct {
		TargetRatio        float64
		TargetSeedingHours int
		ActionAfterProcess string
	}
	General struct {
		LogLevel string
	}
	Maintenance struct {
		LogMaxSizeMB       int
		LogMaxBackups      int
		DBKeepArchivedDays int
	}
}

// rawConfig 结构体精确对应 config.ini 文件中的“问卷式”键名。
type rawConfig struct {
	Uploader struct {
		Path          string `ini:"Path"`
		MyCloudFolder string `ini:"MyCloudFolder"`
		ExtraArgs     string `ini:"ExtraArgs"`
	} `ini:"Uploader"`
	QBittorrent struct {
		Host     string `ini:"Host"`
		Username string `ini:"Username"`
		Password string `ini:"Password"`
	} `ini:"qBittorrent"`
	SeedingPolicy struct {
		TargetRatio        float64 `ini:"Target_Ratio"`
		TargetSeedingHours int     `ini:"Target_Seeding_Hours"`
		ActionAfterProcess string  `ini:"Action_After_Process"`
	} `ini:"Seeding_Policy"`
	General struct {
		LogMode string `ini:"Log_Mode"`
	} `ini:"General"`
	Maintenance struct {
		LogMaxSizeMB       int `ini:"Log_Max_Size_MB"`
		LogMaxBackups      int `ini:"Log_Max_Backups"`
		DBKeepArchivedDays int `ini:"DB_Keep_Archived_Days"`
	} `ini:"Maintenance"`
}

// Init 函数是这个模块的唯一入口。
func Init() error {
	// --- 新的、正确的策略：基于当前工作目录寻找配置文件 ---
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("无法获取当前工作目录: %w", err)
	}
	// 使用 filepath.Join 来安全地拼接路径
	configPath := filepath.Join(wd, "config.ini")

	// 为了调试，我们暂时保留这条打印信息
	fmt.Printf("[DEBUG] 正在基于当前工作目录寻找配置文件: %s\n", configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("配置文件 'config.ini' 不存在于当前目录！请将 'config.ini.example' 复制一份并重命名为 'config.ini'")
	}

	iniCfg, err := ini.Load(configPath)
	if err != nil {
		return fmt.Errorf("无法加载 config.ini: %w", err)
	}

	rawCfg := new(rawConfig)
	if err = iniCfg.MapTo(rawCfg); err != nil {
		return fmt.Errorf("解析 config.ini 失败: %w", err)
	}

	// --- 将“问卷答案”翻译成“标准参数” ---
	Cfg = new(Config)
	Cfg.Uploader.Path = rawCfg.Uploader.Path
	Cfg.Uploader.RemoteDir = rawCfg.Uploader.MyCloudFolder
	Cfg.Uploader.ExtraArgs = strings.Fields(rawCfg.Uploader.ExtraArgs)
	Cfg.QBittorrent.Host = rawCfg.QBittorrent.Host
	Cfg.QBittorrent.Username = rawCfg.QBittorrent.Username
	Cfg.QBittorrent.Password = rawCfg.QBittorrent.Password
	Cfg.SeedingPolicy.TargetRatio = rawCfg.SeedingPolicy.TargetRatio
	Cfg.SeedingPolicy.TargetSeedingHours = rawCfg.SeedingPolicy.TargetSeedingHours
	Cfg.SeedingPolicy.ActionAfterProcess = rawCfg.SeedingPolicy.ActionAfterProcess

	switch strings.ToLower(rawCfg.General.LogMode) {
	case "debug":
		Cfg.General.LogLevel = "debug"
	default:
		Cfg.General.LogLevel = "normal" // 保持和 config.ini.example 中的默认值一致
	}

	Cfg.Maintenance.LogMaxSizeMB = rawCfg.Maintenance.LogMaxSizeMB
	Cfg.Maintenance.LogMaxBackups = rawCfg.Maintenance.LogMaxBackups
	Cfg.Maintenance.DBKeepArchivedDays = rawCfg.Maintenance.DBKeepArchivedDays

	return nil
}