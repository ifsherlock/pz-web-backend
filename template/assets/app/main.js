function app() {
            return {
                currentTab: 'server',
                lang: localStorage.getItem('pz_lang') || 'CN', // 记住用户选择
                theme: localStorage.getItem('pz_theme') || 'dark', // 日夜模式
                i18n: {}, // 存放当前语言的 UI 文本
                languageList: [], // 存放从后端获取的语言列表
                loading: false,
                serverConfig: [],
                sandboxConfig: [],
                serverSections: {},
                sandboxSections: {},
                logs: '...',
                status: { pid: 0, uptime: '0s' },
                stats: { cpu_percent: 0, mem_used_mb: 0, mem_total_mb: 0, mem_percent: 0, uptime_sec: 0, game_version: 'unknown' },
                statsTimer: null,
                customGameBranch: '',
                toast: { show: false, message: '', type: 'success' },
                configTooltip: { show: false, key: '', text: '', left: 12, top: 12, maxWidth: 420 },
                modInput: '',
                modLoading: false,
                availableMods: [], // 本地库
                activeMods: [],    // 当前启用列表
                // 合集展开
                collectionInput: '',
                collectionLoading: false,
                collectionItems: [],
                collectionChecked: [],
                collectionSelectAll: false,
                collectionExpandMsg: '',
                currentCollection: { id: '', name: '' },
                addedCollections: [], // 已添加的合集列表 {id,name}
                // 面板设置
                settingsKey: '',
                settingsKeyConfigured: false,
                settingsKeyMasked: '',
                modsLoaded: false,
                logConnected: false,


                init() {
                    // 应用日夜主题
                    document.documentElement.setAttribute('data-theme', this.theme);
                    this.refreshAll();
                    // 启动仪表盘 CPU/内存轮询(每3秒)
                    this.fetchStats();
                    this.statsTimer = setInterval(() => {
                        this.fetchStats();
                    }, 3000);
                    // 监听 Tab 切换，触发sse
                    this.$watch('currentTab', (val) => {
                        if (val === 'monitor') {
                            this.startLogStream();
                        } else {
                            this.stopLogStream(); // 离开页面时断开连接，节省资源
                        }
                        if (val === 'mods') {
                            this.loadModManager();
                        }
                    });
                },

                // 切换日夜模式
                toggleTheme() {
                    this.theme = this.theme === 'dark' ? 'light' : 'dark';
                    localStorage.setItem('pz_theme', this.theme);
                    document.documentElement.setAttribute('data-theme', this.theme);
                },

                // 拉取容器 CPU / 内存统计
                async fetchStats() {
                    try {
                        const res = await fetch('/api/system/stats');
                        if (!res.ok) return;
                        const data = await res.json();
                        this.stats = data;
                    } catch (e) {
                        console.error('fetch stats failed', e);
                    }
                },

                // 运行时长格式化
                formatUptime(sec) {
                    if (!sec) return '-';
                    const d = Math.floor(sec / 86400);
                    const h = Math.floor((sec % 86400) / 3600);
                    const m = Math.floor((sec % 3600) / 60);
                    if (d > 0) return `${d}天 ${h}时`;
                    if (h > 0) return `${h}时 ${m}分`;
                    if (m > 0) return `${m}分 ${sec % 60}秒`;
                    return `${sec}秒`;
                },

                // 内存 GB 格式化(保留两位小数)
                formatGB(mb) {
                    if (!mb && mb !== 0) return '-';
                    return (mb / 1024).toFixed(2);
                },

                // 从文本中提取 Workshop ID：支持纯数字 ID 和 Steam 链接。
                // 例如 https://steamcommunity.com/sharedfiles/filedetails/?id=3782594903
                extractWorkshopIds(input) {
                    const urlRe = /(?:id=)(\d{4,})/i;
                    const pureRe = /^\d{4,}$/;
                    const parts = String(input).split(/[,，;\s]+/).map(s => s.trim()).filter(Boolean);
                    const ids = [];
                    parts.forEach(p => {
                        let m = p.match(urlRe);
                        if (m) { ids.push(m[1]); return; }
                        if (pureRe.test(p)) { ids.push(p); }
                    });
                    return [...new Set(ids)];
                },

                refreshAll() {
                    this.fetchConfig('server');
                    this.fetchConfig('sandbox');
                    // 预留: this.fetchStatus();
                },

                async initI18n() {
                    this.loading = true;
                    try {
                        // 请求语言列表和 UI 文本
                        const res = await fetch(`/api/i18n?lang=${this.lang}`);
                        const data = await res.json();
                        
                        this.languageList = data.languages;
                        this.i18n = data.ui;
                        this.logs = this.i18n.log_refresh_hint || 'Click refresh...';
                        
                        // 加载完 I18n 后再加载配置
                        this.refreshAll();
                    } catch (e) {
                        console.error("Failed to init i18n", e);
                    } finally {
                        this.loading = false;
                    }
                },

                async switchLanguage() {
                    localStorage.setItem('pz_lang', this.lang);
                    // 重新加载 UI 文本和配置项翻译
                    await this.initI18n();
                },

                // 统一获取配置并分组
                async fetchConfig(type) {
                    this.loading = true;
                    try {
                        const res = await fetch(`/api/config/${type}?lang=${this.lang}`);
                        const data = await res.json();
                        
                        // 按 Section 分组
                        const grouped = data.items.reduce((acc, item) => {
                            const section = item.section || 'General';
                            if (!acc[section]) acc[section] = [];
                            acc[section].push(item);
                            return acc;
                        }, {});

                        if (type === 'server') {
                            const branchItem = data.items.find(i => i.key === 'PZ_BRANCH');
                            if (branchItem && !['public', '42.19', 'legacy41', '__custom__'].includes(branchItem.value)) {
                                this.customGameBranch = branchItem.value;
                                branchItem.value = '__custom__';
                            }
                            this.serverConfig = data.items; // 保存原始数组用于提交
                            const sectionOrder = [
                                this.i18n.server_settings || '服务端基础',
                                this.i18n.general_settings || '常规设置',
                                this.i18n.network_settings || '网络与连接',
                                this.i18n.map_settings || '地图',
                                this.i18n.players_pvp || '玩家与PVP',
                                this.i18n.gameplay_rules || '游戏规则',
                                this.i18n.chat_voice || '聊天与语音',
                                this.i18n.world_environment || '世界环境',
                                this.i18n.vehicle_settings || '车辆设置',
                                this.i18n.client_limits || '客户端限制',
                                this.i18n.anticheat || '反作弊',
                                this.i18n.discord_integration || 'Discord集成',
                                this.i18n.server_config || '服务端配置',
                                this.i18n.server_security || '服务端安全'
                            ];
                            const sectionRank = new Map(sectionOrder.map((name, index) => [name, index]));
                            this.serverSections = Object.fromEntries(
                                Object.entries(grouped)
                                    .filter(([, items]) => items.some(item => item.key !== 'Mods' && item.key !== 'WorkshopItems'))
                                    .sort(([left], [right]) => (sectionRank.get(left) ?? 999) - (sectionRank.get(right) ?? 999))
                            );
                        } else {
                            this.sandboxConfig = data.items;
                            this.sandboxSections = grouped;
                        }
                    } catch (e) {
                        this.showToast((this.i18n.msg_config_load_fail || 'Load failed') + ': ' + e, 'error');
                    } finally {
                        this.loading = false;
                    }
                },

                // 加载独立模组页面。首次进入时加载，手动刷新可强制重读服务器配置。
                async loadModManager(force = false) {
                    if (this.modLoading || (this.modsLoaded && !force)) return;
                    this.modLoading = true;
                    try {
                        if (this.serverConfig.length === 0) {
                            await this.fetchConfig('server');
                        }
                        const availableModsResp = await fetch(`/api/mods`);
                        if (!availableModsResp.ok) throw new Error('failed to load local mods');
                        this.availableMods = await availableModsResp.json();
                        await this.loadSettings();

                        const modsItem = this.serverConfig.find(i => i.key === 'Mods');
                        const wsItem = this.serverConfig.find(i => i.key === 'WorkshopItems');
                        const currentModIds = (modsItem ? modsItem.value : "")
                            .split(';')
                            .map(s => s.trim().replace(/^\\/, ''))
                            .filter(Boolean);
                        const currentWsIds = (wsItem ? wsItem.value : "").split(';').filter(Boolean);

                        this.activeMods = [];
                        if (currentWsIds.length === 0) {
                            this.modsLoaded = true;
                            return;
                        }

                        const res = await fetch(`/api/mods/lookup?ids=${currentWsIds.join(',')}`);
                        if (!res.ok) throw new Error('failed to resolve enabled mods');
                        const lookupData = await res.json();
                        currentWsIds.forEach(wid => {
                            const info = lookupData.find(d => d.workshop_id === wid);
                            if (info) {
                                const potentialModIds = info.mod_id.split(',').map(s => s.trim());
                                const enabledSubMods = potentialModIds.filter(pmid => currentModIds.includes(pmid));
                                enabledSubMods.forEach(mid => {
                                    this.activeMods.push({
                                        name: info.name + (enabledSubMods.length > 1 ? ` (${mid})` : ''),
                                        workshop_id: wid,
                                        mod_id: mid
                                    });
                                });
                            } else {
                                this.activeMods.push({
                                    name: `Unknown Item (${wid})`,
                                    workshop_id: wid,
                                    mod_id: '?'
                                });
                            }
                        });
                        this.modsLoaded = true;
                    } catch (e) {
                        this.showToast((this.i18n.msg_config_load_fail || 'Load failed') + ': ' + e.message, 'error');
                    } finally {
                        this.modLoading = false;
                    }
                },

                // 解析输入框并添加 (支持纯数字 ID 和 Steam 链接)
                async lookupAndAddMods() {
                    if (!this.modInput.trim()) return;
                    
                    // 用 URL 正则解析：支持 id=xxx 链接 / 纯数字 / 多行逗号分隔
                    const rawIds = this.extractWorkshopIds(this.modInput);
                    if (rawIds.length === 0) {
                        this.showToast((this.i18n.msg_parse_fail || 'Parse failed') + ': no valid id', 'error');
                        return;
                    }

                    this.modLoading = true;
                    try {
                        const res = await fetch(`/api/mods/lookup?ids=${rawIds.join(',')}`);
                        const data = await res.json();
                        
                        for (const item of data) {
                            this.addModItem(item);
                        }
                        
                        this.modInput = ''; // 清空输入框
                    } catch (e) {
                         this.showToast((this.i18n.msg_parse_fail || 'Parse failed') + ': ' + e, 'error');
                    } finally {
                        this.modLoading = false;
                    }
                },

                // 添加单个 Mod 项目 (处理多 ModID 情况)
                addModItem(item, collection = null) {
                    // 检查 ModID 是否存在
                    let mid = item.mod_id;
                    const collectionMeta = collection ? {
                        collection_id: collection.id,
                        collection_name: collection.name
                    } : {};
                    
                    // 如果包含多个 ID (例如 "ID1,ID2")
                    if (mid.includes(',')) {
                        const choices = mid.split(',');
                        // 全部添加进去，最省事
                        choices.forEach(subId => {
                            this.pushToActive({
                                name: item.name + ` (${subId})`, // 区分名字
                                workshop_id: item.workshop_id,
                                mod_id: subId.trim(),
                                ...collectionMeta
                            });
                        });
                        return;
                    }

                    //  如果 ModID 未知 (?)
                    if (mid === '?' || mid === 'Unknown (Check Page)') {
                         const msg = (this.i18n.prompt_mod_id_manual || '').replace('{0}', item.name);
                        mid = prompt(msg || `Please enter Mod ID for ${item.name}:`);
                        if (!mid) return;
                    }
                    
                    this.pushToActive({
                        name: item.name,
                        workshop_id: item.workshop_id,
                        mod_id: mid,
                        ...collectionMeta
                    });
                },
                //添加到列表并去重
                pushToActive(modObj) {
                    // 检查是否已存在 (根据 ModID)
                    if (this.activeMods.some(m => m.mod_id === modObj.mod_id)) return false;
                    this.activeMods.push(modObj);
                    return true;
                },
                // 从本地库添加
                addFromLocal(mod) {
                    this.pushToActive(mod);
                },

                // 移除
                removeMod(index) {
                    this.activeMods.splice(index, 1);
                    this.syncCollectionChecked();
                },

                clearActiveMods() {
                    this.activeMods = [];
                    this.syncCollectionChecked();
                },

                // 排序
                moveMod(index, delta) {
                    const newIndex = index + delta;
                    if (newIndex < 0 || newIndex >= this.activeMods.length) return;
                    
                    const temp = this.activeMods[index];
                    this.activeMods[index] = this.activeMods[newIndex];
                    this.activeMods[newIndex] = temp;
                },

                // 将页面状态同步回服务器配置中的 Mods / WorkshopItems 字段。
                syncModsToServerConfig() {
                    // 提取 Mod IDs (分号分隔)
                    const modsStr = this.activeMods
                    .map(m => `\\${m.mod_id}`) // 加反斜杠，配置是那么写的。
                    .join(';');
                    
                    // 提取 Workshop IDs (分号分隔，去重)
                    // 注意：过滤掉 '?' 和本地已有的 workshopID
                    const wsIds = [...new Set(this.activeMods.map(m => m.workshop_id).filter(id => id && id !== '?'))];
                    const wsStr = wsIds.join(';');

                    // 更新 serverConfig 数组
                    const modsItem = this.serverConfig.find(i => i.key === 'Mods');
                    const wsItem = this.serverConfig.find(i => i.key === 'WorkshopItems');

                    if (modsItem) modsItem.value = modsStr;
                    if (wsItem) wsItem.value = wsStr;

                },

                saveModsToConfig() {
                    this.syncModsToServerConfig();
                    this.showToast(this.i18n.msg_mod_list_updated, 'success');
                },

                async saveModsConfig(restart) {
                    this.syncModsToServerConfig();
                    await this.saveConfig('server', restart);
                },

                // 展开合集：输入合集 ID 或 Steam 链接，显示全部子模组供勾选
                async expandCollection() {
                    const ids = this.extractWorkshopIds(this.collectionInput || '');
                    if (ids.length === 0) return;
                    const id = ids[0];
                    this.collectionLoading = true;
                    this.collectionExpandMsg = '';
                    this.collectionItems = [];
                    this.collectionChecked = [];
                    this.collectionSelectAll = false;
                    try {
                        const res = await fetch(`/api/mods/collection?id=${encodeURIComponent(id)}`);
                        const data = await res.json();
                        if (!res.ok) {
                            const msg = data.error || '';
                            if (msg.toLowerCase().includes('api key')) {
                                this.collectionExpandMsg = this.i18n.mod_collection_not_key || msg;
                            } else {
                                this.collectionExpandMsg = (this.i18n.mod_collection_error || 'Collection failed') + ': ' + msg;
                            }
                            return;
                        }
                        this.currentCollection = {
                            id: data.collection_id || id,
                            name: data.name || id
                        };
                        this.collectionItems = data.children || [];
                        this.syncCollectionChecked();
                        // 记录已添加的合集（避免重复）
                        if (!this.addedCollections.some(c => c.id === id)) {
                            this.addedCollections.push({ id: id, name: data.name || id });
                        }
                    } catch (e) {
                        this.collectionExpandMsg = (this.i18n.mod_collection_error || 'Collection failed') + ': ' + e;
                    } finally {
                        this.collectionLoading = false;
                    }
                },

                // 从已添加合集列表重新展开某个合集
                async reexpandCollection(id) {
                    this.collectionInput = id;
                    await this.expandCollection();
                },

                // 移除已添加合集记录（仅从列表移除，不影响已勾选子项）
                removeCollectionRecord(idx) {
                    this.addedCollections.splice(idx, 1);
                },

                // 合集勾选会即时同步到右侧已启用列表。
                setCollectionItemSelection(mod, idx, checked, updateSelectAll = true) {
                    this.collectionChecked[idx] = checked;
                    if (checked) {
                        this.addModItem(mod, this.currentCollection);
                    } else {
                        this.activeMods = this.activeMods.filter(active => !(
                            active.collection_id === this.currentCollection.id &&
                            active.workshop_id === mod.workshop_id
                        ));
                    }
                    if (updateSelectAll) {
                        this.collectionSelectAll = this.collectionItems.length > 0 && this.collectionChecked.every(Boolean);
                    }
                },

                setCollectionSelection(checked) {
                    this.collectionItems.forEach((mod, idx) => {
                        this.setCollectionItemSelection(mod, idx, checked, false);
                    });
                    this.collectionSelectAll = checked && this.collectionItems.length > 0;
                },

                syncCollectionChecked() {
                    this.collectionChecked = this.collectionItems.map(mod => this.activeMods.some(active =>
                        active.collection_id === this.currentCollection.id &&
                        active.workshop_id === mod.workshop_id
                    ));
                    this.collectionSelectAll = this.collectionItems.length > 0 && this.collectionChecked.every(Boolean);
                },

                // 添加勾选的合集子模组到已启用列表
                addSelectedCollectionMods() {
                    this.collectionItems.forEach((mod, idx) => {
                        if (this.collectionChecked[idx]) {
                            this.addModItem(mod, this.currentCollection);
                        }
                    });
                    this.collectionExpandMsg = this.i18n.mod_collection_added_notice || this.i18n.mod_collection_add_selected;
                },

                // 加载面板设置(API Key / 内存)
                async loadSettings() {
                    try {
                        const res = await fetch(`/api/settings`);
                        const data = await res.json();
                        this.settingsKeyConfigured = !!data.steam_api_key_configured;
                        // 显示掩码 Key 让用户确认已存放(不显示完整 Key)
                        this.settingsKeyMasked = data.steam_api_key_masked || '';
                    } catch (e) {
                        console.error('load settings failed', e);
                    }
                },

                // 保存面板设置
                async saveSettings() {
                    const key = this.settingsKey.trim();
                    if (!key) return;
                    try {
                        const res = await fetch(`/api/settings`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({
                                steam_api_key: key
                            })
                        });
                        const data = await res.json();
                        if (!res.ok) throw new Error(data.error || 'save failed');
                        // 成功: 记录掩码 Key, 清空输入框但保留"已配置"状态
                        this.settingsKeyConfigured = !!data.steam_api_key_masked;
                        this.settingsKeyMasked = data.steam_api_key_masked || '';
                        this.settingsKey = '';
                        this.showToast(this.i18n.mod_settings_saved || 'Settings saved', 'success');
                    } catch (e) {
                        this.showToast(e.message, 'error');
                    }
                },

                // 保存配置
                async saveConfig(type, restart) {
                    if (restart && !confirm(this.i18n.confirm_save_restart)) return;
                    this.loading = true;
                    try {
                        const sourceItems = type === 'server' ? this.serverConfig : this.sandboxConfig;
                        const items = sourceItems.map(item => {
                            if (type === 'server' && item.key === 'PZ_BRANCH' && item.value === '__custom__') {
                                return { ...item, value: this.customGameBranch.trim() };
                            }
                            return item;
                        });
                        if (type === 'server' && items.some(item => item.key === 'PZ_BRANCH' && !item.value)) {
                            throw new Error('请输入 Steam 分支名');
                        }

                        const res = await fetch(`/api/config/${type}`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ items: items, restart: restart })
                        });

                        if (!res.ok) throw new Error(this.i18n.msg_save_fail);

                        this.showToast(this.i18n.msg_save_success, 'success');

                    } catch (e) {
                        this.showToast(e.message, 'error');
                    } finally {
                        this.loading = false;
                    }
                },

                async performAction(action) {
                    this.loading = true;
                    try {
                        const res = await fetch(`/api/action/${action}`, { method: 'POST' });
                        if (res.ok) {
                             this.showToast((this.i18n.msg_cmd_sent || 'Command Sent') + ': ' + action, 'success');
                        } else {
                            throw new Error(this.i18n.msg_exec_fail);
                        }
                    } catch (e) {
                        this.showToast(e.message, 'error');
                    } finally {
                        this.loading = false;
                    }
                },

                 // 开启 SSE 日志流
                startLogStream() {
                    if (this.eventSource) return; // 避免重复连接

                    this.logs = "Connecting...\n";
                    
                    // 创建 EventSource
                    this.eventSource = new EventSource('/api/logs/stream');

                    this.eventSource.onopen = () => {
                        this.logConnected = true;
                        this.logs = ""; // 连接成功后清空提示
                    };

                    this.eventSource.onmessage = (event) => {
                        // 追加日志
                        this.logs += event.data + "\n";
                        
                        // 限制日志长度，防止浏览器内存爆炸（例如保留最近 5000 字符）
                        if (this.logs.length > 10000) {
                            this.logs = this.logs.slice(-5000);
                        }

                        // 自动滚动到底部
                        this.$nextTick(() => {
                            if (this.$refs.logBox) {
                                this.$refs.logBox.scrollTop = this.$refs.logBox.scrollHeight;
                            }
                        });
                    };

                    this.eventSource.onerror = (err) => {
                        console.error("SSE Error:", err);
                        this.logConnected = false;
                        this.eventSource.close();
                        this.eventSource = null;
                        // 可以选择自动重连，或者手动重连
                        this.logs += "\n[disconnect...]\n";
                    };
                },

                // 关闭 SSE 日志流
                stopLogStream() {
                    if (this.eventSource) {
                        this.eventSource.close();
                        this.eventSource = null;
                        this.logConnected = false;
                    }
                },
                // 切换日志流开关
                toggleLogStream() {
                    if (this.logConnected) {
                        this.stopLogStream();
                    } else {
                        this.startLogStream();
                    }
                },

                showConfigTooltip(event, item) {
                    const text = String(item.tooltip || '').trim();
                    if (!text) return;

                    const targetRect = event.currentTarget.getBoundingClientRect();
                    const maxWidth = Math.max(220, Math.min(420, window.innerWidth - 24));
                    this.configTooltip = {
                        show: true,
                        key: item.key,
                        text,
                        left: 12,
                        top: 12,
                        maxWidth
                    };

                    this.$nextTick(() => {
                        window.requestAnimationFrame(() => {
                            if (!this.configTooltip.show || !this.$refs.configTooltip) return;
                            const tooltipRect = this.$refs.configTooltip.getBoundingClientRect();
                            const viewportPadding = 12;
                            const gap = 10;
                            const centeredLeft = targetRect.left + (targetRect.width - tooltipRect.width) / 2;
                            const maxLeft = window.innerWidth - tooltipRect.width - viewportPadding;
                            let top = targetRect.top - tooltipRect.height - gap;

                            if (top < viewportPadding) {
                                top = targetRect.bottom + gap;
                            }

                            this.configTooltip.left = Math.max(viewportPadding, Math.min(centeredLeft, maxLeft));
                            this.configTooltip.top = Math.max(
                                viewportPadding,
                                Math.min(top, window.innerHeight - tooltipRect.height - viewportPadding)
                            );
                        });
                    });
                },

                hideConfigTooltip() {
                    this.configTooltip.show = false;
                },

                showToast(msg, type = 'success') {
                    this.toast.message = msg;
                    this.toast.type = type;
                    this.toast.show = true;
                    setTimeout(() => this.toast.show = false, 3000);
                },
                // 重启面板 API 调用
                restartPanel() {
                    if (!confirm(this.i18n.restart_in_progress)) return;

                    fetch('/api/service/restart', { method: 'POST' })
                        .then(res => res.json())
                        .then(data => {
                            alert(data.message);
                            // 3秒后刷新页面
                            setTimeout(() => location.reload(), 3000);
                        })
                        .catch(err => alert(this.i18n.restart_failed + err));
                },
                async checkUpdate() {
                    this.loading = true;
                    try {
                        const res = await fetch('/api/system/check_update');
                        const data = await res.json();
                        if (data.new_version) {
                            // 获取模板文本
                            let msg = this.i18n.msg_update_found || 'New version {0} found (Current: {1})\nUpdate now?';
                            
                            // 简单的手动替换占位符
                            msg = msg.replace('{0}', data.new_version)
                                    .replace('{1}', data.current);

                            if (confirm(msg)) {
                                await fetch('/api/system/perform_update', {
                                    method: 'POST',
                                    headers: {'Content-Type': 'application/json'},
                                    body: JSON.stringify({ url: data.download_url })
                                });
                                // 使用 i18n
                                alert(this.i18n.msg_update_performing || 'Update command sent, please refresh later.');
                            }
                        } else {
                            if(data.error) {
                                this.showToast(data.error, 'error');
                                return
                            }
                            // 使用 i18n
                            this.showToast(this.i18n.msg_already_latest || 'Already latest version');
                        }
                    } catch(e) {
                        // 使用 i18n
                        this.showToast(this.i18n.msg_update_check_fail || 'Check update failed', 'error');
                    } finally {
                        this.loading = false;
                    }
                }
            }
        }
