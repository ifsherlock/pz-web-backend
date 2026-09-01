package mods

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

type WorkshopClient struct {
	httpClient *http.Client
	apiURL     string
	cache      CacheStore

	// apiKey 用于 GetCollectionDetails(合集子项) 接口。
	// 普通单个 Mod 解析(GetPublishedFileDetails)不需要 key。
	apiKey string

	mu    sync.RWMutex
	mem   map[string]ModInfo
	dirty bool
}

func NewWorkshopClient(httpClient *http.Client, apiURL string, cache CacheStore) (*WorkshopClient, error) {
	return NewWorkshopClientWithKey(httpClient, apiURL, cache, "")
}

func NewWorkshopClientWithKey(httpClient *http.Client, apiURL string, cache CacheStore, apiKey string) (*WorkshopClient, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if apiURL == "" {
		apiURL = "https://api.steampowered.com/ISteamRemoteStorage/GetPublishedFileDetails/v1/"
	}

	c := &WorkshopClient{
		httpClient: httpClient,
		apiURL:     apiURL,
		cache:      cache,
		apiKey:     strings.TrimSpace(apiKey),
		mem:        make(map[string]ModInfo),
	}

	if cache != nil {
		if loaded, err := cache.Load(); err == nil && loaded != nil {
			c.mem = loaded
		}
	}
	return c, nil
}

type steamResponse struct {
	Response struct {
		ResultCount          int `json:"resultcount"`
		PublishedFileDetails []struct {
			PublishedFileID string `json:"publishedfileid"`
			Title           string `json:"title"`
			Description     string `json:"description"`
			FileType        int    `json:"filetype"`
		} `json:"publishedfiledetails"`
	} `json:"response"`
}

// collectionResponse GetCollectionDetails 的响应结构。
type collectionResponse struct {
	Response struct {
		ResultCount int `json:"resultcount"`
		Collections []struct {
			CollectionID string `json:"collectionid"`
			Children     []struct {
				PublishedFileID string `json:"publishedfileid"`
			} `json:"children"`
		} `json:"collections"`
	} `json:"response"`
}

// IsCollection 判断给定 workshopID 是否为合集。
// Steam 的 GetPublishedFileDetails 不总是返回 filetype，
// 因此最可靠的方式是直接尝试 GetCollectionDetails：合集能返回 children。
func (c *WorkshopClient) IsCollection(workshopID string) (bool, error) {
	children, err := c.FetchCollectionChildren(workshopID)
	if err != nil {
		// 缺 API Key 或网络错误时，无法判断；返回错误由上层提示。
		return false, err
	}
	return len(children) > 0, nil
}

// FetchCollectionChildren 获取合集内所有子项的 Workshop ID。
// 需要 c.apiKey；合集本身不可被当作单个 Mod 订阅。
func (c *WorkshopClient) FetchCollectionChildren(collectionID string) ([]string, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("steam api key is required to expand collections")
	}

	apiURL := "https://api.steampowered.com/ISteamRemoteStorage/GetCollectionDetails/v1/"
	form := url.Values{}
	form.Set("collectioncount", "1")
	form.Set("collectionids[0]", collectionID)
	form.Set("key", c.apiKey)

	resp, err := c.httpClient.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var col collectionResponse
	if err := json.Unmarshal(body, &col); err != nil {
		// Steam 在缺 Key / ID 无效时可能返回 HTML 错误页，转成友好错误。
		return nil, fmt.Errorf("steam returned an invalid response (check API Key or collection id)")
	}

	if len(col.Response.Collections) == 0 {
		return nil, fmt.Errorf("collection not found")
	}

	var ids []string
	for _, child := range col.Response.Collections[0].Children {
		ids = append(ids, child.PublishedFileID)
	}
	return ids, nil
}

func (c *WorkshopClient) FetchWorkshopInfo(workshopID string) (ModInfo, error) {
	c.mu.RLock()
	if info, ok := c.mem[workshopID]; ok {
		c.mu.RUnlock()
		return info, nil
	}
	c.mu.RUnlock()

	data := url.Values{}
	data.Set("itemcount", "1")
	data.Set("publishedfileids[0]", workshopID)

	resp, err := c.httpClient.Post(c.apiURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return ModInfo{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ModInfo{}, err
	}

	var steamResp steamResponse
	if err := json.Unmarshal(body, &steamResp); err != nil {
		return ModInfo{}, err
	}

	if steamResp.Response.ResultCount <= 0 || len(steamResp.Response.PublishedFileDetails) == 0 {
		return ModInfo{}, fmt.Errorf("mod not found")
	}

	details := steamResp.Response.PublishedFileDetails[0]
	modID := extractModID(details.Description)
	if modID == "" {
		modID = "?"
	}

	info := ModInfo{
		Name:          details.Title,
		WorkshopID:    workshopID,
		ModID:         modID,
		Description:   details.Description,
		IsCollection:  details.FileType == 2,
		CollectionURL: collectionURL(workshopID),
	}

	c.mu.Lock()
	c.mem[workshopID] = info
	c.dirty = true
	c.mu.Unlock()

	if c.cache != nil {
		go c.flush()
	}

	return info, nil
}

func collectionURL(workshopID string) string {
	return fmt.Sprintf("https://steamcommunity.com/sharedfiles/filedetails/?id=%s", workshopID)
}

// SetSteamAPIKey 动态更新 API Key(合集展开即时生效，无需重启)。
func (c *WorkshopClient) SetSteamAPIKey(key string) {
	c.mu.Lock()
	c.apiKey = strings.TrimSpace(key)
	c.mu.Unlock()
}

func (c *WorkshopClient) flush() {
	c.mu.RLock()
	if !c.dirty {
		c.mu.RUnlock()
		return
	}
	snapshot := make(map[string]ModInfo, len(c.mem))
	for k, v := range c.mem {
		snapshot[k] = v
	}
	c.mu.RUnlock()

	_ = c.cache.Save(snapshot)

	c.mu.Lock()
	c.dirty = false
	c.mu.Unlock()
}

func extractModID(desc string) string {
	re := regexp.MustCompile(`(?i)Mod\s*ID\s*:\s*([^\r\n<]+)`)
	matches := re.FindAllStringSubmatch(desc, -1)

	var ids []string
	seen := make(map[string]bool)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		id := strings.TrimSpace(m[1])
		id = strings.Trim(id, "[]")
		if id == "" || seen[id] {
			continue
		}
		ids = append(ids, id)
		seen[id] = true
	}

	if len(ids) == 0 {
		return ""
	}
	return strings.Join(ids, ",")
}
