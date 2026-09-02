const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const source = fs.readFileSync(path.join(__dirname, '..', '..', 'template', 'assets', 'app', 'main.js'), 'utf8');
const context = vm.createContext({
    console,
    localStorage: { getItem: () => null },
    window: { innerWidth: 1024, innerHeight: 768, requestAnimationFrame: callback => callback() }
});
vm.runInContext(source, context);

function createState() {
    const state = context.app();
    state.i18n = {};
    return state;
}

test('checking a collection item adds tagged active mods and unchecking removes them', () => {
    const state = createState();
    const item = { name: 'Bundle Mod', workshop_id: '100', mod_id: 'A,B' };
    state.currentCollection = { id: '900', name: 'Starter Collection' };
    state.collectionItems = [item];
    state.collectionChecked = [false];

    state.setCollectionItemSelection(item, 0, true);

    assert.equal(state.activeMods.length, 2);
    assert.deepEqual(Array.from(state.activeMods, mod => mod.mod_id), ['A', 'B']);
    assert.ok(state.activeMods.every(mod => mod.collection_id === '900'));
    assert.equal(state.collectionChecked[0], true);

    state.setCollectionItemSelection(item, 0, false);

    assert.equal(state.activeMods.length, 0);
    assert.equal(state.collectionChecked[0], false);
});

test('select all synchronizes collection mods without removing standalone duplicates', () => {
    const state = createState();
    const items = [
        { name: 'Already Enabled', workshop_id: '100', mod_id: 'A' },
        { name: 'Collection Only', workshop_id: '200', mod_id: 'B' }
    ];
    state.activeMods = [{ name: 'Standalone', workshop_id: '100', mod_id: 'A' }];
    state.currentCollection = { id: '900', name: 'Starter Collection' };
    state.collectionItems = items;
    state.collectionChecked = [false, false];

    state.setCollectionSelection(true);

    assert.equal(state.activeMods.length, 2);
    assert.equal(state.activeMods.find(mod => mod.mod_id === 'A').collection_id, undefined);
    assert.equal(state.activeMods.find(mod => mod.mod_id === 'B').collection_id, '900');
    assert.deepEqual(Array.from(state.collectionChecked), [true, true]);

    state.setCollectionSelection(false);

    assert.deepEqual(Array.from(state.activeMods, mod => mod.mod_id), ['A']);
    assert.deepEqual(Array.from(state.collectionChecked), [false, false]);
});

test('saving an empty API key is a no-op', async () => {
    const state = createState();
    state.settingsKey = '   ';
    await state.saveSettings();
    assert.equal(state.settingsKeyConfigured, false);
});

test('saving the standalone mods page updates server fields before persistence', async () => {
    const state = createState();
    state.serverConfig = [
        { key: 'Mods', value: '' },
        { key: 'WorkshopItems', value: '' },
        { key: 'MaxPlayers', value: '8' }
    ];
    state.activeMods = [
        { name: 'A', workshop_id: '100', mod_id: 'ModA' },
        { name: 'B', workshop_id: '100', mod_id: 'ModB' }
    ];
    let saved = null;
    state.saveConfig = async (type, restart) => {
        saved = { type, restart };
    };

    await state.saveModsConfig(true);

    assert.equal(state.serverConfig[0].value, '\\ModA;\\ModB');
    assert.equal(state.serverConfig[1].value, '100');
    assert.deepEqual(saved, { type: 'server', restart: true });
});

test('configuration explanations open above a key and can be dismissed', () => {
    const state = createState();
    state.$refs = {
        configTooltip: {
            getBoundingClientRect: () => ({ width: 320, height: 80 })
        }
    };
    state.$nextTick = callback => callback();
    const event = {
        currentTarget: {
            getBoundingClientRect: () => ({ left: 200, right: 260, top: 180, bottom: 200, width: 60, height: 20 })
        }
    };

    state.showConfigTooltip(event, { key: 'PVP', tooltip: '玩家可以攻击其他玩家.' });

    assert.equal(state.configTooltip.show, true);
    assert.equal(state.configTooltip.key, 'PVP');
    assert.equal(state.configTooltip.text, '玩家可以攻击其他玩家.');
    assert.equal(state.configTooltip.top, 90);
    assert.equal(state.configTooltip.left, 70);

    state.hideConfigTooltip();
    assert.equal(state.configTooltip.show, false);
});
