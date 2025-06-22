package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"qbuploader/internal/config"
	"qbuploader/internal/logger"

	_ "github.com/mattn/go-sqlite3" // 匿名导入 sqlite3 驱动
)

// DB 是一个全局的、可供其他包使用的数据库连接实例。
var DB *sql.DB

const (
	// 定义数据表创建的 SQL 语句
	createTableSQL = `
	CREATE TABLE IF NOT EXISTS tasks (
		info_hash     TEXT PRIMARY KEY,
		torrent_name  TEXT,
		upload_status TEXT NOT NULL DEFAULT 'pending',
		message       TEXT,
		created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	// 创建一个触发器，在更新行时自动更新 updated_at 字段
	createTriggerSQL = `
	CREATE TRIGGER IF NOT EXISTS update_tasks_updated_at
	AFTER UPDATE ON tasks FOR EACH ROW
	BEGIN
		UPDATE tasks SET updated_at = CURRENT_TIMESTAMP WHERE info_hash = OLD.info_hash;
	END;`
)

// Task 结构体代表数据库中的一条任务记录。
type Task struct {
	InfoHash     string
	TorrentName  string
	UploadStatus string
	Message      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Init 函数负责连接到数据库，并在必要时进行初始化。
func Init() error {
	log := logger.Log
	log.Debug("正在初始化数据库模块...")

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("无法获取当前工作目录: %w", err)
	}
	dbPath := filepath.Join(wd, "database.db")
	log.Debugf("数据库文件路径: %s", dbPath)

	db, err := sql.Open("sqlite3", dbPath)
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

// === 数据操作函数 ===

// AddTask 添加一个新任务到数据库，或在已存在时忽略。
func AddTask(infoHash, torrentName string) error {
	// "OR IGNORE" 确保如果 info_hash 已存在，此命令不会报错，而是被静默忽略。
	query := `INSERT OR IGNORE INTO tasks (info_hash, torrent_name) VALUES (?, ?)`
	_, err := DB.Exec(query, infoHash, torrentName)
	if err != nil {
		return fmt.Errorf("添加任务 %s 到数据库失败: %w", torrentName, err)
	}
	return nil
}

// UpdateTaskStatus 更新指定任务的上传状态和消息。
func UpdateTaskStatus(infoHash, status, message string) error {
	query := `UPDATE tasks SET upload_status = ?, message = ? WHERE info_hash = ?`
	_, err := DB.Exec(query, status, message, infoHash)
	if err != nil {
		return fmt.Errorf("更新任务 %s 状态失败: %w", infoHash, err)
	}
	return nil
}

// GetSuccessfulUploads 获取所有已成功上传任务的 InfoHash。
// 返回一个 map[string]bool 格式的数据，便于快速查找。
func GetSuccessfulUploads() (map[string]bool, error) {
	query := `SELECT info_hash FROM tasks WHERE upload_status = 'success'`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询已上传任务失败: %w", err)
	}
	defer rows.Close()

	hashes := make(map[string]bool)
	for rows.Next() {
		var infoHash string
		if err := rows.Scan(&infoHash); err != nil {
			return nil, fmt.Errorf("扫描任务哈希失败: %w", err)
		}
		hashes[infoHash] = true
	}
	return hashes, nil
}

// PruneOldTasks 清理掉N天前的已归档任务记录。
func PruneOldTasks() (int64, error) {
	days := config.Cfg.Maintenance.DBKeepArchivedDays
	if days <= 0 {
		return 0, nil // 如果配置为0或负数，则不清理
	}

	query := `DELETE FROM tasks WHERE upload_status = 'archived' AND updated_at < date('now', '-' || ? || ' day')`
	result, err := DB.Exec(query, days)
	if err != nil {
		return 0, fmt.Errorf("清理旧数据库记录失败: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}

// ArchiveTask 将任务状态标记为已归档。
func ArchiveTask(infoHash string) error {
	return UpdateTaskStatus(infoHash, "archived", "任务已完成并归档")
}
