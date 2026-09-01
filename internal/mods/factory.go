package mods

import "net/http"

// NewFileCachedWorkshopClient 创建带文件缓存的 WorkshopClient。
// cachePath: 缓存文件路径（建议在 DEV_MODE 使用项目根目录，生产用 /opt/pz-web-backend/...）。
func NewFileCachedWorkshopClient(cachePath string, httpClient *http.Client) (*WorkshopClient, error) {
	return NewFileCachedWorkshopClientWithKey(cachePath, httpClient, "")
}

// NewFileCachedWorkshopClientWithKey 与 NewFileCachedWorkshopClient 相同，
// 额外支持 Steam API Key（用于 GetCollectionDetails 合集解析）。
func NewFileCachedWorkshopClientWithKey(cachePath string, httpClient *http.Client, apiKey string) (*WorkshopClient, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	store := FileCacheStore{Path: cachePath}
	return NewWorkshopClientWithKey(httpClient, "", store, apiKey)
}
