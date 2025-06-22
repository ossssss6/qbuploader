package qbittorrent

import (
	"fmt"

	"github.com/autobrr/go-qbittorrent"
	"qbuploader/internal/config"
	"qbuploader/internal/logger"
)

// Client 是一个封装好的 qBittorrent 客户端。
// 我们在自己的结构体里包裹了第三方库的 Client，
// 这样未来如果更换库，只需要修改这个文件，而不会影响到其他业务逻辑。
type Client struct {
	*qbittorrent.Client
}

// NewClient 创建并返回一个已登录的 qBittorrent 客户端实例。
func NewClient() (*Client, error) {
	log := logger.Log
	cfg := config.Cfg.QBittorrent

	log.Debugf("正在创建 qBittorrent 客户端，目标主机: %s", cfg.Host)

	// 使用从 config.ini 读取的配置来创建客户端配置
	clientCfg := qbittorrent.Config{
		Host:     cfg.Host,
		Username: cfg.Username,
		Password: cfg.Password,
	}

	// 创建第三方库的客户端实例
	client := qbittorrent.NewClient(clientCfg)

	log.Debug("正在登录 qBittorrent...")
	if err := client.Login(); err != nil {
		return nil, fmt.Errorf("登录 qBittorrent 失败: %w", err)
	}

	// 使用正确的函数名 GetAppVersion 获取版本信息，以验证连接
	appVersion, err := client.GetAppVersion()
	if err != nil {
		return nil, fmt.Errorf("获取 qBittorrent 版本信息失败: %w", err)
	}
	log.Infof("成功连接到 qBittorrent！版本: %s", appVersion)

	// 返回我们自己封装后的 Client
	return &Client{client}, nil
}

// GetTorrents 封装了获取所有种子列表的操作。
func (c *Client) GetTorrents() ([]qbittorrent.Torrent, error) {
	// 使用正确的参数类型 TorrentFilterOptions{}
	filterOptions := qbittorrent.TorrentFilterOptions{}
	torrents, err := c.Client.GetTorrents(filterOptions)
	if err != nil {
		return nil, fmt.Errorf("获取种子列表失败: %w", err)
	}
	return torrents, nil
}

// DeleteTorrents 封装了删除指定种子的操作。
func (c *Client) DeleteTorrents(infoHashes []string, deleteFiles bool) error {
	// 使用 v1.14.0 版本中正确的函数名 DeleteTorrents
	if err := c.Client.DeleteTorrents(infoHashes, deleteFiles); err != nil {
		return fmt.Errorf("删除种子失败: %w", err)
	}
	return nil
}