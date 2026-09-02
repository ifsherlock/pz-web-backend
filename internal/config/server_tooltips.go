package config

// serverTooltipCN fills the small set of server keys whose game translation
// packages have no useful tooltip.
var serverTooltipCN = map[string]string{
	"ChatStreams":                      "用逗号分隔允许的聊天频道，例如 s（说）、r（半径）、a（全局）、w（耳语）、y（大喊）、sh（安全屋）、f（阵营）、all（全部）。",
	"UsernameDisguises":                "允许玩家使用伪装用户名。",
	"HideDisguisedUserName":            "启用后，伪装用户名不会显示真实用户名。",
	"SwitchZombiesOwnershipEachUpdate": "每次服务器更新时重新分配僵尸归属，用于多人同步。",
	"SafetyDisconnectDelay":            "玩家关闭 PVP 安全模式后的生效延迟，单位为秒。",
	"UDPPort":                          "Steam/游戏 UDP 端口，单位为端口号；通常与 Docker 映射的 16262 对应。",
	"DenyLoginOnOverloadedServer":      "服务器负载过高时拒绝新玩家登录，避免进一步恶化同步。",
	"SafehouseDisableDisguises":        "安全屋内禁止使用用户名伪装。",
	"MaxSafezoneSize":                  "安全区允许的最大面积，单位为地图格平方。",
	"War":                              "启用玩家战争模式；具体开始延迟和持续时间由同组参数控制。",
	"SneakModeHideFromOtherPlayers":    "启用后，潜行时会降低其他玩家发现你的可见性。",
	"UltraSpeedDoesnotAffectToAnimals": "启用后，极速时间不会影响动物状态推进。",
	"SpeedLimit":                       "玩家移动速度上限，单位为游戏内部速度值。",
	"LoginQueueEnabled":                "启用登录队列，服务器满员时让玩家排队等待。",
	"LoginQueueConnectTimeout":         "登录队列连接超时时间，单位为秒。",
	"BanKickGlobalSound":               "管理员封禁或踢出玩家时播放全局提示音。",
	"BackupsCount":                     "游戏内自动备份保留数量，单位为份；面板备份另有独立保留上限。",
	"BackupsOnStart":                   "服务端启动时创建一份游戏内备份。",
	"BackupsOnVersionChange":           "检测到游戏版本变化时创建一份游戏内备份。",
	"BackupsPeriod":                    "游戏内自动备份周期，单位为小时；0 表示关闭周期备份。",
	"UsePhysicsHitReaction":            "使用物理受击反应，可能影响服务器性能和客户端表现。",
	"ChatMessageCharacterLimit":        "单条聊天消息最大字符数，单位为字符。",
	"ChatMessageSlowModeTime":          "同一玩家发送聊天消息的最短间隔，单位为秒。",
	"RCONPassword":                     "RCON（远程控制台）密码。外部 RCON 客户端可用它连接服务器执行管理命令；不用 RCON 时留空。",
}
