package baidupcs

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"qbuploader/internal/config"
	"qbuploader/internal/logger"
)

// Uploader 封装了所有与 BaiduPCS-Go 相关的操作。
type Uploader struct {
	executablePath string
	extraArgs      []string
}

// NewUploader 创建一个新的 Uploader 实例。
func NewUploader() *Uploader {
	return &Uploader{
		executablePath: config.Cfg.Uploader.Path,
		extraArgs:      config.Cfg.Uploader.ExtraArgs,
	}
}

// Upload 执行上传操作。
func (u *Uploader) Upload(localPath, remoteDir, torrentName string) error {
	log := logger.Log
	remotePath := fmt.Sprintf("%s/%s", remoteDir, torrentName)

	args := []string{
		"upload",
		localPath,
		remotePath,
	}
	args = append(args, u.extraArgs...)

	log.Infof("  -> 正在上传: %s -> %s", localPath, remotePath)
	log.Debugf("  -> 执行命令: %s %v", u.executablePath, args)

	// 使用带有超时的上下文，防止命令卡死
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour) // 24小时超时
	defer cancel()

	cmd := exec.CommandContext(ctx, u.executablePath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Errorf("BaiduPCS-Go 上传失败。输出: %s, 错误: %s", stdout.String(), stderr.String())
		return fmt.Errorf("执行 BaiduPCS-Go 上传命令失败: %w", err)
	}

	log.Debugf("BaiduPCS-Go 上传成功。输出: %s", stdout.String())
	return nil
}

// CheckFileExists 校验网盘文件是否存在。
func (u *Uploader) CheckFileExists(remoteDir, torrentName string) (bool, error) {
	log := logger.Log
	remotePath := fmt.Sprintf("%s/%s", remoteDir, torrentName)

	args := []string{
		"ls",
		remotePath,
	}

	log.Infof("  -> 正在校验网盘文件: %s", remotePath)
	log.Debugf("  -> 执行命令: %s %v", u.executablePath, args)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // 5分钟超时
	defer cancel()

	cmd := exec.CommandContext(ctx, u.executablePath, args...)

	// 对于 `ls`，我们只关心它是否成功执行（返回码为0）。
	// 如果文件不存在，它会返回非0，并把错误信息打印到 stderr。
	if err := cmd.Run(); err != nil {
		return false, nil // 认为文件不存在，这不是一个程序错误。
	}

	return true, nil
}