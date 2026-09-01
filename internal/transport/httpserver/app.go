package httpserver

import (
	"net/http"
	"time"

	"pz-web-backend/internal/application/configapp"
	"pz-web-backend/internal/application/i18napp"
	"pz-web-backend/internal/application/modsapp"
	"pz-web-backend/internal/application/updateapp"
	"pz-web-backend/internal/config"
	"pz-web-backend/internal/i18n"
	"pz-web-backend/internal/infra/executil"
	"pz-web-backend/internal/infra/fs"
	"pz-web-backend/internal/infra/logtail"
	"pz-web-backend/internal/infra/pzpaths"
	"pz-web-backend/internal/infra/supervisor"
	"pz-web-backend/internal/mods"
	sysupdate "pz-web-backend/internal/system/update"
)

type BuildInfo struct {
	Version    string
	GithubRepo string
	CommitSHA  string
	BuildTime  string
}

type App struct {
	BaseDataDir string
	BaseGameDir string
	Build       BuildInfo
	LogPath     string

	Config config.Service
	I18n   *i18n.Loader

	ConfigApp configapp.Service
	I18nApp   i18napp.Service
	ModsApp   modsapp.Service
	UpdateApp updateapp.Service
	LogTailer logtail.Tailer

	Settings *settingsStore
}

func NewApp(baseDataDir string, baseGameDir string, serverName string, logPath string, build BuildInfo, devMode bool) App {
	osfs := fs.OSFS{}
	runner := executil.OSRunner{}
	restarter := supervisor.SupervisorctlRestarter{Runner: runner}
	tailer := logtail.OSTailer{}

	resolvedServerName := configapp.ResolveServerName(osfs, baseDataDir, serverName)

	loader := i18n.NewLoader(baseGameDir)
	configSvc := config.Service{
		I18n: loader,
		SectionLabel: func(lang string, sectionKey string) string {
			if langMap, ok := i18n.WebUIResources[lang]; ok {
				if val, ok := langMap[sectionKey]; ok && val != "" {
					return val
				}
			}
			if langMap, ok := i18n.WebUIResources["EN"]; ok {
				if val, ok := langMap[sectionKey]; ok && val != "" {
					return val
				}
			}
			return sectionKey
		},
	}

	installDir := pzpaths.DefaultInstallDir(devMode)

	// 加载面板持久化设置(Steam API Key / 内存上限等)。
	settingsStore := newSettingsStore(osfs, installDir)

	// Steam API Key 用于解析创意工坊合集(GetCollectionDetails)，
	// 普通单个 Mod 解析不需要。
	workshopClient := mustDefaultWorkshopClient(devMode, settingsStore.Load().SteamAPIKey)

	updateChecker := sysupdate.Service{
		HTTPClient:     &http.Client{Timeout: 10 * time.Second},
		GithubRepo:     build.GithubRepo,
		CurrentVersion: build.Version,
	}
	updateSvc := updateapp.NewService(devMode, updateChecker)

	return App{
		BaseDataDir: baseDataDir,
		BaseGameDir: baseGameDir,
		Build:       build,
		LogPath:     logPath,
		I18n:        loader,
		Config:      configSvc,
		Settings:    settingsStore,

		ConfigApp: configapp.Service{
			BaseDataDir: baseDataDir,
			ServerName:  resolvedServerName,
			DevMode:     devMode,
			Config:      configSvc,
			FS:          osfs,
			Runner:      runner,
			Restarter:   restarter,
		},
		I18nApp: i18napp.Service{
			BaseGameDir: baseGameDir,
			FS:          osfs,
		},
		ModsApp: modsapp.Service{
			InstallDir: installDir,
			Workshop:   workshopClient,
			Collection: workshopClient,
		},
		UpdateApp: updateSvc,
		LogTailer: tailer,
	}
}

func mustDefaultWorkshopClient(devMode bool, apiKey string) *mods.WorkshopClient {
	path := pzpaths.WorkshopCachePath(devMode)
	client, err := mods.NewFileCachedWorkshopClientWithKey(path, &http.Client{Timeout: 10 * time.Second}, apiKey)
	if err != nil {
		panic(err)
	}
	return client
}
