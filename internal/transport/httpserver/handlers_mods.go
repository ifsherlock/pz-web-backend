package httpserver

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (a App) handleModsLookup(c *gin.Context) {
	idsStr := c.Query("ids")
	if idsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}

	targetIds := strings.Split(idsStr, ",")
	var results []ModInfo

	lookup, _ := a.ModsApp.Lookup(targetIds)
	for _, item := range lookup {
		if item.Err != nil {
			results = append(results, ModInfo{
				Name:       "Network Error / Invalid ID",
				WorkshopID: item.WorkshopID,
				ModID:      "?",
			})
			continue
		}
		for _, m := range item.Mods {
			results = append(results, m)
		}
	}

	c.JSON(http.StatusOK, results)
}

// handleExpandCollection 展开合集：输入合集 ID，返回合集信息 + 全部子 Mod 的详情。
// 需要一个有效的 Steam API Key(见 settings 接口)。
func (a App) handleExpandCollection(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	// 先查合集基本信息(名称等)
	info, err := a.ModsApp.FetchWorkshopInfo(id)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	children, err := a.ModsApp.FetchCollectionChildren(id)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if len(children) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not a collection or empty"})
		return
	}

	// 逐个解析子项信息
	var mods []ModInfo
	for _, childID := range children {
		child, err := a.ModsApp.FetchWorkshopInfo(childID)
		if err != nil {
			mods = append(mods, ModInfo{
				Name:       "Unknown Item",
				WorkshopID: childID,
				ModID:      "?",
			})
			continue
		}
		mods = append(mods, child)
	}

	c.JSON(http.StatusOK, gin.H{
		"collection_id": id,
		"name":          info.Name,
		"children":      mods,
	})
}

func (a App) handleListLocalMods(c *gin.Context) {
	localMods, _ := a.ModsApp.ListLocalMods()
	if localMods == nil {
		c.JSON(http.StatusOK, []ModInfo{})
		return
	}
	c.JSON(http.StatusOK, localMods)
}
