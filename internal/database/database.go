package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"qbuploader/internal/logger" // 注意：它不再导入 config

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

const (
	createTableSQL = `
	CREATE TABLE IF NOT EXISTS tasks (
		info_hash     TEXT PRIMARY KEY,
		torrent_name  TEXT,
		upload_status TEXT NOT NULL DEFAULT 'pending',
		message       TEXT,
		created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	createTriggerSQL = `
	CREATE TRIGGER IF NOT EXISTS update_tasks_updated_at
	AFTER UPDATE ON tasks FOR EACH ROW
	BEGIN
		UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE info_hash = OLD.info_hash;
	END;`
)

type Task struct {
	InfoHash     string
	TorrentName  string
	UploadStatus string
	Message      sql.NullString
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func Init() error {
	log := logger.Log
	log.Debug("正在初始化数据库模块...")

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("无法获取当前工作目录: %w", err)
	}
	dbPath := filepath.Join(wd, "database.db")
	log.Debugf("数据库文件路径: %s", dbPath)

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}
	if err = db.Ping(); err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	log.Debug("正在检查并创建数据表...")
	if _, err = db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("创建 'tasks' 表失败: %w", err)
	}
	if _, err = db.Exec(createTriggerSQL); err != nil {
		return fmt.Errorf("创建 'updated_at' 触发器失败: %w", err)
	}

	DB = db
	log.Debug("数据库初始化成功！")
	return nil
}
