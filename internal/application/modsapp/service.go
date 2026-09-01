package modsapp

import (
	"fmt"

	"pz-web-backend/internal/mods"
)

type WorkshopFetcher interface {
	FetchWorkshopInfo(workshopID string) (mods.ModInfo, error)
}

type CollectionFetcher interface {
	IsCollection(workshopID string) (bool, error)
	FetchCollectionChildren(collectionID string) ([]string, error)
}

type Service struct {
	InstallDir string
	Workshop   WorkshopFetcher
	Collection CollectionFetcher
}

type LookupResult struct {
	WorkshopID string
	Mods       []mods.ModInfo
	Source     string // local | steam
	Err        error
}

func (s Service) ListLocalMods() ([]mods.ModInfo, error) {
	if s.InstallDir == "" {
		return nil, fmt.Errorf("install dir is empty")
	}
	return mods.ScanLocalMods(s.InstallDir)
}

// IsCollection 判断 workshopID 是否为合集。
func (s Service) IsCollection(workshopID string) (bool, error) {
	if s.Collection == nil {
		return false, fmt.Errorf("collection support not configured")
	}
	return s.Collection.IsCollection(workshopID)
}

// FetchCollectionChildren 获取合集内全部子项 Workshop ID。
func (s Service) FetchCollectionChildren(collectionID string) ([]string, error) {
	if s.Collection == nil {
		return nil, fmt.Errorf("collection support not configured")
	}
	return s.Collection.FetchCollectionChildren(collectionID)
}

// FetchWorkshopInfo 查询单个 workshop 物品信息。
func (s Service) FetchWorkshopInfo(workshopID string) (mods.ModInfo, error) {
	if s.Workshop == nil {
		return mods.ModInfo{}, fmt.Errorf("workshop fetcher not configured")
	}
	return s.Workshop.FetchWorkshopInfo(workshopID)
}

func (s Service) Lookup(workshopIDs []string) ([]LookupResult, error) {
	localMods, err := s.ListLocalMods()
	if err != nil {
		// 本地扫描失败不影响 Steam 查询（仍返回结果），错误由每项携带。
		localMods = nil
	}

	results := make([]LookupResult, 0, len(workshopIDs))
	for _, wid := range workshopIDs {
		var matched []mods.ModInfo
		for _, lm := range localMods {
			if lm.WorkshopID == wid {
				matched = append(matched, lm)
			}
		}
		if len(matched) > 0 {
			results = append(results, LookupResult{
				WorkshopID: wid,
				Mods:       matched,
				Source:     "local",
			})
			continue
		}

		if s.Workshop == nil {
			results = append(results, LookupResult{
				WorkshopID: wid,
				Source:     "steam",
				Err:        fmt.Errorf("workshop fetcher not configured"),
			})
			continue
		}

		info, fetchErr := s.Workshop.FetchWorkshopInfo(wid)
		if fetchErr != nil {
			results = append(results, LookupResult{
				WorkshopID: wid,
				Source:     "steam",
				Err:        fetchErr,
			})
			continue
		}

		results = append(results, LookupResult{
			WorkshopID: wid,
			Mods:       []mods.ModInfo{info},
			Source:     "steam",
		})
	}

	return results, nil
}
