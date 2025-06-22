package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

var Cfg *Config

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
		// TargetRatio 和 TargetSeedingHours 不再使用，但暂时保留以防未来需要
		TargetRatio           float64
		TargetSeedingHours    int
		ActionAfterProcess    string
		Cleanup_Target_States string // <<<--- 【新增】在最终使用的 Config 结构体中添加
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
		TargetRatio           float64 `ini:"Target_Ratio"`
		TargetSeedingHours    int     `ini:"Target_Seeding_Hours"`
		ActionAfterProcess    string  `ini:"Action_After_Process"`
		Cleanup_Target_States string  `ini:"Cleanup_Target_States"` // <<<--- 【新增】在原始 rawConfig 结构体中添加
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

func Init() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("无法获取当前工作目录: %w", err)
	}
	configPath := filepath.Join(wd, "config.ini")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("配置文件 'config.ini' 不存在！请将 'config.ini.example' 复制一份并重命名为 'config.ini'")
	}

	iniCfg, err := ini.Load(configPath)
	if err != nil {
		return fmt.Errorf("无法加载 config.ini: %w", err)
	}

	rawCfg := new(rawConfig)
	if err = iniCfg.MapTo(rawCfg); err != nil {
		return fmt.Errorf("解析 config.ini 失败: %w", err)
	}

	Cfg = new(Config)
	// ... (其他赋值不变)
	Cfg.Uploader.Path = rawCfg.Uploader.Path
	Cfg.Uploader.RemoteDir = rawCfg.Uploader.MyCloudFolder
	Cfg.Uploader.ExtraArgs = strings.Fields(rawCfg.Uploader.ExtraArgs)
	Cfg.QBittorrent.Host = rawCfg.QBittorrent.Host
	Cfg.QBittorrent.Username = rawCfg.QBittorrent.Username
	Cfg.QBittorrent.Password = rawCfg.QBittorrent.Password

	// SeedingPolicy 部分
	Cfg.SeedingPolicy.TargetRatio = rawCfg.SeedingPolicy.TargetRatio
	Cfg.SeedingPolicy.TargetSeedingHours = rawCfg.SeedingPolicy.TargetSeedingHours
	Cfg.SeedingPolicy.ActionAfterProcess = rawCfg.SeedingPolicy.ActionAfterProcess
	Cfg.SeedingPolicy.Cleanup_Target_States = rawCfg.SeedingPolicy.Cleanup_Target_States // <<<--- 【新增】将读取到的值赋给最终配置

	// ... (其他赋值不变)
	switch strings.ToLower(rawCfg.General.LogMode) {
	case "debug":
		Cfg.General.LogLevel = "debug"
	default:
		Cfg.General.LogLevel = "normal"
	}
	Cfg.Maintenance.LogMaxSizeMB = rawCfg.Maintenance.LogMaxSizeMB
	Cfg.Maintenance.LogMaxBackups = rawCfg.Maintenance.LogMaxBackups
	Cfg.Maintenance.DBKeepArchivedDays = rawCfg.Maintenance.DBKeepArchivedDays

	return nil
}
