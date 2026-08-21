-- Dumps calc-core fixtures and checkpoints for the calc port's archive
-- comparison. One process per corpus source (in-process build reloads keep
-- stale state):
--
--   luajit ../../tools/dump_calc.lua empty
--   luajit ../../tools/dump_calc.lua coc "Builds/3.29 CoC Blazing Salvo Assassin CI.xml"
--
-- Run from .archive/src/. Writes test/testdata/calc_<key>.jsonl containing,
-- per variant (full / noskills / treeonly for builds; empty standalone):
--   <variant>.fixture - everything initEnv reads from the build
--   <variant>.dbs     - modDB/enemyDB/itemModDB state after initEnv
--   <variant>.skills  - active skill summaries (compared once the skill
--                       stage is ported)
local key = arg and arg[1] or "empty"
local buildPath = arg and arg[2]

package.cpath = "../runtime/?.dll;" .. package.cpath
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
dofile("HeadlessWrapper.lua")

-- HeadlessWrapper stubs Inflate to "" but tree specs in build XML are
-- zlib-compressed URLs - without the real thing the spec silently stays a
-- default Scion. Same binding as .claude/skills/cook/tools/pob.lua.
do
	local haveFfi, ffi = pcall(require, "ffi")
	if haveFfi then
		pcall(ffi.cdef, [[
int uncompress(uint8_t *dest, unsigned long *destLen, const uint8_t *source, unsigned long sourceLen);
]])
		local haveZlib, zlib = pcall(ffi.load, "../runtime/zlib1.dll")
		if haveZlib then
			function Inflate(data)
				local cap = #data * 6 + 4096
				for _ = 1, 8 do
					local buf = ffi.new("uint8_t[?]", cap)
					local len = ffi.new("unsigned long[1]", cap)
					local res = zlib.uncompress(buf, len, data, #data)
					if res == 0 then return ffi.string(buf, len[0]) end
					if res ~= -5 then return nil end -- only retry Z_BUF_ERROR
					cap = cap * 2
				end
			end
		end
	end
end

local canon = dofile("../../tools/canon.lua")

-- The calc modules localize `pairs` at load time, so overriding the global
-- BEFORE LoadModule scopes a deterministic iteration order to exactly the
-- Calc* modules: numeric keys ascending, then string keys ascending, then
-- other (table) keys in raw next() order. The Go replay mirrors the same
-- order wherever it iterates these maps. Documented divergence from the
-- vanilla app (whose hash order is random per process anyway); this also
-- makes the dumps byte-stable across runs.
local rawPairs = pairs
local function sortedPairs(t)
	local numKeys, strKeys, otherKeys = {}, {}, {}
	for k in rawPairs(t) do
		local ty = type(k)
		if ty == "number" then
			numKeys[#numKeys + 1] = k
		elseif ty == "string" then
			strKeys[#strKeys + 1] = k
		else
			otherKeys[#otherKeys + 1] = k
		end
	end
	table.sort(numKeys)
	table.sort(strKeys)
	-- Table keys are sets of objects (env.flasks, env.tinctures): their
	-- pairs() order is LuaJIT hash order over addresses, i.e. random per
	-- process. Two flasks granting an identical mod then differ only by
	-- which one is merged first, so the dump was not reproducible. Order
	-- them by item id when every key carries one; the Go replay sorts the
	-- same way (calc.sortedFlasks).
	local allHaveId = #otherKeys > 0
	for _, k in ipairs(otherKeys) do
		if type(k) ~= "table" or type(rawget(k, "id")) ~= "number" then
			allHaveId = false
			break
		end
	end
	if allHaveId then
		table.sort(otherKeys, function(a, b) return a.id < b.id end)
	end
	local keys = {}
	for _, k in ipairs(numKeys) do
		keys[#keys + 1] = k
	end
	for _, k in ipairs(strKeys) do
		keys[#keys + 1] = k
	end
	for _, k in ipairs(otherKeys) do
		keys[#keys + 1] = k
	end
	local i = 0
	return function()
		i = i + 1
		if keys[i] ~= nil then
			return keys[i], t[keys[i]]
		end
	end
end
pairs = sortedPairs
local calcs = LoadModule("Modules/Calcs")
pairs = rawPairs

-- The tree merge iterates pairs(nodeList): LuaJIT hash order, deterministic
-- per table state but not derivable in Go — and the table GROWS mid-initEnv
-- (granted passives), so each call sees its own order. Wrap the function to
-- record the order per call.
-- mergeSkillInstanceMods iterates pairs(stats): string keys, so LuaJIT
-- hash-randomised PER PROCESS - the resulting skillModList order would
-- differ between dump runs and be underivable in Go. Replace it with a
-- sorted-stats replica (body from CalcActiveSkill.lua, including the
-- mergeLevelMod cache) so both sides derive the same deterministic order.
-- Documented divergence from the vanilla app (same technique as the
-- game-data dump's sorted re-passes).
do
	local mergeLevelCache = {}
	local function mergeLevelMod(modList, mod, value)
		if not value then
			modList:AddMod(mod)
			return
		end
		if not mergeLevelCache[mod] then
			mergeLevelCache[mod] = {}
		end
		if mergeLevelCache[mod][value] then
			modList:AddMod(mergeLevelCache[mod][value])
		elseif value then
			local newMod = copyTable(mod, true)
			if type(newMod.value) == "table" then
				newMod.value = copyTable(newMod.value, true)
				if newMod.value.mod then
					newMod.value.mod = copyTable(newMod.value.mod, true)
					newMod.value.mod.value = value
				else
					newMod.value.value = value
				end
			else
				newMod.value = value
			end
			mergeLevelCache[mod][value] = newMod
			modList:AddMod(newMod)
		else
			modList:AddMod(mod)
		end
	end
	calcs.mergeSkillInstanceMods = function(env, modList, skillEffect, extraStats)
		calcLib.validateGemLevel(skillEffect)
		local grantedEffect = skillEffect.grantedEffect
		local stats = calcLib.buildSkillInstanceStats(skillEffect, grantedEffect)
		if extraStats and extraStats[1] then
			for _, stat in pairs(extraStats) do
				stats[stat.key] = (stats[stat.key] or 0) + stat.value
			end
		end
		local statKeys = {}
		for stat in pairs(stats) do
			statKeys[#statKeys + 1] = stat
		end
		table.sort(statKeys)
		for _, stat in ipairs(statKeys) do
			local statValue = stats[stat]
			local map = grantedEffect.statMap[stat]
			if map then
				for _, modOrGroup in ipairs(map) do
					if modOrGroup.name then
						mergeLevelMod(modList, modOrGroup, map.value or statValue * (map.mult or 1) / (map.div or 1) + (map.base or 0))
					else
						for _, mod in ipairs(modOrGroup) do
							mergeLevelMod(modList, mod, modOrGroup.value or statValue * (modOrGroup.mult or 1) / (modOrGroup.div or 1) + (modOrGroup.base or 0))
						end
					end
				end
			end
		end
		modList:AddList(grantedEffect.baseMods)
	end
end

-- The perform checkpoint covers the perform BODY only: the final
-- defence/offence handoff (CalcPerform L3721+) is stubbed out so the
-- captured state is exactly what the ported body must reproduce.
-- calcTotemLife stays real (mid-body call for totem skills).
-- The real handoff functions are kept so later checkpoints can run them
-- explicitly, one stage at a time, on the post-perform-body state.
local realDefence = calcs.defence
local realBuildDefenceEstimations = calcs.buildDefenceEstimations
calcs.defence = function() end
calcs.buildDefenceEstimations = function() end
calcs.triggers = function() end
calcs.mirages = function() return true end
calcs.offence = function() end

local recordedOrders
local recordedNodeSeqs
local origBuildModListForNodeList = calcs.buildModListForNodeList
calcs.buildModListForNodeList = function(env, nodeList, finishJewels)
	if recordedOrders then
		local order = {}
		-- must match the order the wrapped function iterates in (the calc
		-- modules were loaded under sortedPairs)
		for id in sortedPairs(nodeList) do
			order[#order + 1] = id
		end
		recordedOrders[#recordedOrders + 1] = order
	end
	if recordedNodeSeqs then
		recordedNodeSeqs[#recordedNodeSeqs + 1] = {}
	end
	return origBuildModListForNodeList(env, nodeList, finishJewels)
end
-- Record every buildModListForNode call's node id per NodeList call: the
-- tail beyond allocOrders is the extraRadiusNodeList pairs() order, which
-- the radius-jewel replay must follow.
local origBuildModListForNode = calcs.buildModListForNode
calcs.buildModListForNode = function(env, node)
	if recordedNodeSeqs and recordedNodeSeqs[#recordedNodeSeqs] then
		local seq = recordedNodeSeqs[#recordedNodeSeqs]
		seq[#seq + 1] = node.id
	end
	return origBuildModListForNode(env, node)
end

local out = assert(io.open("../../test/testdata/calc_" .. key .. ".jsonl", "w"))
local function emit(name, value)
	out:write('{"k":', canon.quote(name), ',"c":', canon.quote(canon.encode(value)), "}\n")
end

-- ModList objects carry actor/parent backrefs; canon their array part only.
local function modArray(list)
	local o = {}
	for i, mod in ipairs(list or {}) do
		o[i] = mod
	end
	return o
end

-- Scalar-only copy (fixture config input holds strings/numbers/booleans).
local function scalars(t)
	local o = {}
	for k, v in pairs(t or {}) do
		local ty = type(v)
		if ty == "string" or ty == "number" or ty == "boolean" then
			o[k] = v
		end
	end
	return o
end

local function nodeFixture(node)
	return {
		id = node.id,
		type = node.type,
		name = node.name,
		dn = node.dn,
		isTattoo = node.isTattoo,
		overrideType = node.overrideType,
		conqueredBy = node.conqueredBy and true or nil,
		modList = modArray(node.modList),
		keystoneMod = node.keystoneMod,
	}
end

-- Active mod line texts, split explicit vs other, for the replay's
-- FindModifierSubstring (Item.lua:284 merges these two pools).
local function modLinesActive(item)
	local expl, other = {}, {}
	local function collect(lines, dst)
		for _, v in ipairs(lines or {}) do
			if not v.disabled and item:CheckModLineVariant(v) then
				dst[#dst + 1] = v.line
			end
		end
	end
	collect(item.explicitModLines, expl)
	collect(item.enchantModLines, other)
	collect(item.scourgeModLines, other)
	collect(item.implicitModLines, other)
	collect(item.crucibleModLines, other)
	return expl, other
end

local function itemFixture(item)
	assert(not (item.jewelData and item.jewelData.jewelIncEffectFromClassStart),
		"dump_calc: jewelIncEffectFromClassStart unported: " .. item.name)
	-- funcList functions are re-derived by the replay from the item's mod
	-- lines (modparser jewels.go); emit the types for the assertion.
	local funcTypes
	if item.jewelData and item.jewelData.funcList then
		funcTypes = {}
		for i, func in ipairs(item.jewelData.funcList) do
			funcTypes[i] = func.type
		end
	end
	local expl, other = modLinesActive(item)
	local slotML
	if item.slotModList then
		slotML = {}
		for i, l in pairs(item.slotModList) do
			slotML[i] = modArray(l)
		end
	end
	local grantedSkills = {}
	for i, skill in ipairs(item.grantedSkills or {}) do
		grantedSkills[i] = scalars(skill)
	end
	local weaponData
	if item.weaponData then
		weaponData = {}
		for i = 1, 2 do
			if item.weaponData[i] then
				weaponData[i] = scalars(item.weaponData[i])
			end
		end
	end
	return {
		name = item.name,
		modSource = item.modSource,
		title = item.title,
		baseName = item.baseName,
		type = item.type,
		rarity = item.rarity,
		corrupted = item.corrupted or nil,
		shaper = item.shaper,
		elder = item.elder,
		adjudicator = item.adjudicator,
		basilisk = item.basilisk,
		crusader = item.crusader,
		eyrie = item.eyrie,
		foulborn = item.foulborn,
		classRestriction = item.classRestriction,
		limit = item.limit,
		base = item.base and {
			subType = item.base.subType,
			type = item.base.type,
			flask = item.base.flask and { life = item.base.flask.life or nil, mana = item.base.flask.mana or nil } or nil,
		} or nil,
		quality = item.quality,
		modList = item.modList and modArray(item.modList) or nil,
		slotModList = slotML,
		baseModList = item.baseModList and modArray(item.baseModList) or nil,
		buffModList = item.buffModList and modArray(item.buffModList) or nil,
		grantedSkills = grantedSkills,
		requirements = item.requirements and scalars(item.requirements) or nil,
		sockets = item.sockets,
		abyssalSocketCount = item.abyssalSocketCount,
		socketedJewelEffectModifier = item.socketedJewelEffectModifier,
		jewelRadiusIndex = item.jewelRadiusIndex,
		funcTypes = funcTypes,
		jewelData = item.jewelData and scalars(item.jewelData) or nil,
		flaskData = item.flaskData and scalars(item.flaskData) or nil,
		tinctureData = item.tinctureData and scalars(item.tinctureData) or nil,
		armourData = item.armourData and scalars(item.armourData) or nil,
		weaponData = weaponData,
		explicitLines = expl,
		otherLines = other,
	}
end

local function itemsTabFixture(build)
	local itemsTab = build.itemsTab
	-- radiusNodeData collects in-radius nodes that are not allocated (the
	-- replay's extraRadiusNodeList and threshold-data source).
	local radiusNodeData = {}
	local slots = {}
	for i, slot in ipairs(itemsTab.orderedSlots) do
		local containJewelSocket
		if slot.nodeId then
			local specNode = build.spec.nodes[slot.nodeId]
			containJewelSocket = specNode and specNode.containJewelSocket or nil
		end
		local radiusNodes, radiusAttributes
		local slotItem = itemsTab.items[slot.selItemId]
		if slot.nodeId and slotItem and slotItem.jewelRadiusIndex then
			local specNode = build.spec.nodes[slot.nodeId]
			radiusNodes = {}
			if specNode and specNode.nodesInRadius then
				for id, radNode in pairs(specNode.nodesInRadius[slotItem.jewelRadiusIndex]) do
					radiusNodes[id] = radNode.type
					if not build.spec.allocNodes[id] then
						radiusNodeData[id] = radiusNodeData[id] or nodeFixture(build.spec.nodes[id])
					end
				end
			end
			radiusAttributes = specNode and specNode.attributesInRadius
				and scalars(specNode.attributesInRadius[slotItem.jewelRadiusIndex]) or nil
		end
		slots[i] = {
			slotName = slot.slotName,
			label = slot.label,
			slotNum = slot.slotNum,
			weaponSet = slot.weaponSet,
			nodeId = slot.nodeId,
			active = slot.active or nil,
			parentSlotName = slot.parentSlot and slot.parentSlot.slotName or nil,
			itemId = (slot.selItemId and slot.selItemId ~= 0) and slot.selItemId or nil,
			containJewelSocket = containJewelSocket,
			radiusNodes = radiusNodes,
			radiusAttributes = radiusAttributes,
		}
	end
	local items = {}
	for id, item in pairs(itemsTab.items) do
		items[id] = itemFixture(item)
	end
	-- Item sets: minionHasItemSet skills (Animate Guardian) pull weapon
	-- data through them.
	local itemSets = {}
	for id, set in pairs(itemsTab.itemSets) do
		local setSlots = {}
		for slotName, slotData in pairs(set) do
			if type(slotData) == "table" and slotData.selItemId and slotData.selItemId ~= 0 then
				setSlots[slotName] = slotData.selItemId
			end
		end
		itemSets[id] = { useSecondWeaponSet = set.useSecondWeaponSet or nil, slots = setSlots }
	end
	return {
		useSecondWeaponSet = itemsTab.activeItemSet.useSecondWeaponSet or nil,
		slots = slots,
		items = items,
		itemSets = itemSets,
		itemSetOrderList = itemsTab.itemSetOrderList,
	}, radiusNodeData
end

local function gemFixture(build, gem)
	-- explodeSource is an item or tree-node reference
	local explodeItemId, explodeNodeId
	if gem.explodeSource then
		for id, it in pairs(build.itemsTab.items) do
			if it == gem.explodeSource then
				explodeItemId = id
				break
			end
		end
		if not explodeItemId then
			explodeNodeId = gem.explodeSource.id
		end
	end
	return {
		kv = scalars(gem),
		gemDataId = gem.gemData and gem.gemData.id or nil,
		grantedEffectId = gem.grantedEffect and gem.grantedEffect.id or nil,
		explodeSourceItemId = explodeItemId,
		explodeSourceNodeId = explodeNodeId,
	}
end

local function skillsTabFixture(build)
	local skillsTab = build.skillsTab
	local groups = {}
	for i, group in ipairs(skillsTab.socketGroupList) do
		local gems = {}
		for j, gem in ipairs(group.gemList) do
			gems[j] = gemFixture(build, gem)
		end
		groups[i] = { kv = scalars(group), gemList = gems }
	end
	local imbued
	if skillsTab.imbuedSupportBySlot then
		imbued = {}
		for slotName, ge in pairs(skillsTab.imbuedSupportBySlot) do
			imbued[slotName] = ge.id
		end
	end
	return { socketGroupList = groups, imbuedSupportBySlot = imbued }
end


local function dbState(db)
	return { mods = db.mods, conditions = db.conditions, multipliers = db.multipliers }
end

-- The app's load-time calc and each variant's own perform run mutate
-- shared skill tag tables in place (e.g. warcryBuff[1].warcryPowerBonus,
-- CalcPerform L2330). That residue is perform-owned state recomputed on
-- every perform, so scrub it before capturing each variant's
-- post-initEnv checkpoints (documented divergence).
local function scrubPerformResidue(t, seen)
	if seen[t] then
		return
	end
	seen[t] = true
	for k, v in pairs(t) do
		if k == "warcryPowerBonus" then
			t[k] = nil
		elseif type(v) == "table" then
			scrubPerformResidue(v, seen)
		end
	end
end

local function dumpVariant(name, build)
	-- Each variant's own perform run re-creates the shared-table residue;
	-- scrub before capturing this variant's post-initEnv state.
	scrubPerformResidue(data.skills, {})
	local classStats = build.spec.tree.characterData and build.spec.tree.characterData[build.spec.curClassId]
		or build.spec.tree.classes[build.spec.curClassId]
	local allocNodes = {}
	for id, node in pairs(build.spec.allocNodes) do
		allocNodes[id] = nodeFixture(node)
	end
	-- mergeKeystones resolves keystone names through the tree's keystoneMap.
	local keystoneMap = {}
	for name, ksNode in pairs(build.spec.tree.keystoneMap) do
		keystoneMap[name] = modArray(ksNode.modList)
	end
	local itemsTabF, radiusNodeData = itemsTabFixture(build)
	local fixture = {
		characterLevel = build.characterLevel,
		classId = build.spec.curClassId,
		configEnemyLevel = build.configTab.enemyLevel,
		curClassName = build.spec.curClassName,
		treeVersion = build.spec.treeVersion,
		mainSocketGroup = build.mainSocketGroup,
		classStats = { base_str = classStats.base_str, base_dex = classStats.base_dex, base_int = classStats.base_int },
		spectreList = build.spectreList,
		configInput = scalars(build.configTab.input),
		configPlaceholder = scalars(build.configTab.placeholder),
		itemsTab = itemsTabF,
		skillsTab = skillsTabFixture(build),
		configModList = modArray(build.configTab.modList),
		configEnemyModList = modArray(build.configTab.enemyModList),
		partyEnemyModList = modArray(build.partyTab.enemyModList),
		spec = {
			allocNodes = allocNodes,
			keystoneMap = keystoneMap,
			radiusNodeData = radiusNodeData,
			allocatedNotableCount = build.spec.allocatedNotableCount,
			allocatedKeystoneCount = build.spec.allocatedKeystoneCount,
			allocatedMasteryCount = build.spec.allocatedMasteryCount,
			allocatedMasteryTypeCount = build.spec.allocatedMasteryTypeCount,
			allocatedMasteryTypes = build.spec.allocatedMasteryTypes,
			allocatedTattooTypes = build.spec.allocatedTattooTypes,
		},
	}
	emit(name .. ".fixture", fixture)

	recordedOrders = {}
	recordedNodeSeqs = {}
	local env = calcs.initEnv(build, "MAIN")
	emit(name .. ".allocOrders", recordedOrders)
	emit(name .. ".nodeOrders", recordedNodeSeqs)
	recordedOrders = nil
	recordedNodeSeqs = nil
	-- Granted passives (anoints etc.) resolve names through the tree's
	-- notable/ascendancy maps; emit the resolved nodes for the replay.
	local grantedPassiveNodes = {}
	for _, passive in pairs(env.modDB:List(nil, "GrantedPassive")) do
		local node = env.spec.tree.notableMap[passive] or env.spec.tree.ascendancyMap[passive]
		local specNode = node and env.spec.nodes[node.id]
		node = node or build.latestTree.ascendancyMap[passive]
		if node then
			grantedPassiveNodes[passive] = nodeFixture(specNode or node)
		end
	end
	emit(name .. ".grantedPassiveNodes", grantedPassiveNodes)
	-- Granted ascendancy nodes (Forbidden Flame/Flesh) resolve through the
	-- ascendancy maps; emit the resolved nodes for the replay.
	local grantedAscendancyNodes = {}
	for _, ascTbl in pairs(env.modDB:List(nil, "GrantedAscendancyNode")) do
		local node = env.spec.tree.ascendancyMap[ascTbl.name] or build.latestTree.ascendancyMap[ascTbl.name]
		if node then
			grantedAscendancyNodes[ascTbl.name] = nodeFixture(node)
		end
	end
	emit(name .. ".grantedAscendancyNodes", grantedAscendancyNodes)
	-- Energy Blade replaces weapons with synthesized items (initEnv
	-- re-entry); emit the constructed items so the replay can substitute
	-- them instead of porting Item construction.
	local energyBladeItems = {}
	for _, ebSlot in ipairs({ "Weapon 1", "Weapon 2" }) do
		local ebItem = env.player.itemList[ebSlot]
		if ebItem and ebItem.name and ebItem.name:match("^Energy Blade") then
			energyBladeItems[ebSlot] = itemFixture(ebItem)
		end
	end
	emit(name .. ".energyBladeItems", energyBladeItems)
	emit(name .. ".dbs", {
		mod = dbState(env.modDB),
		enemy = dbState(env.enemyDB),
		item = dbState(env.itemModDB),
	})
	local skills = {}
	for i, activeSkill in ipairs(env.player.activeSkillList) do
		skills[i] = {
			name = activeSkill.activeEffect.grantedEffect.name,
			id = activeSkill.activeEffect.grantedEffect.id,
			level = activeSkill.activeEffect.level,
			quality = activeSkill.activeEffect.quality,
			isMain = (env.player.mainSkill == activeSkill) or nil,
		}
	end
	emit(name .. ".skills", skills)
	-- Post-buildActiveSkillModList state per active skill.
	local skillLists = {}
	for i, activeSkill in ipairs(env.player.activeSkillList) do
		local buffs = {}
		for j, buff in ipairs(activeSkill.buffList) do
			local b = scalars(buff)
			b.modList = modArray(buff.modList)
			buffs[j] = b
		end
		skillLists[i] = {
			modList = modArray(activeSkill.skillModList),
			cfg = scalars(activeSkill.skillCfg),
			flags = activeSkill.skillFlags,
			data = scalars(activeSkill.skillData),
			buffs = buffs,
			weapon1Flags = activeSkill.weapon1Flags,
			weapon2Flags = activeSkill.weapon2Flags,
			weapon1Cfg = activeSkill.weapon1Cfg and scalars(activeSkill.weapon1Cfg) or nil,
			weapon1Cond = activeSkill.weapon1Cfg and activeSkill.weapon1Cfg.skillCond or nil,
			weapon2Cfg = activeSkill.weapon2Cfg and scalars(activeSkill.weapon2Cfg) or nil,
			weapon2Cond = activeSkill.weapon2Cfg and activeSkill.weapon2Cfg.skillCond or nil,
			disableReason = activeSkill.disableReason,
			skillPartName = activeSkill.skillPartName,
			skillTotemId = activeSkill.skillTotemId,
			extraSkillModList = modArray(activeSkill.extraSkillModList or {}),
			minionList = activeSkill.minionList,
			minion = activeSkill.minion and {
				type = activeSkill.minion.type,
				level = activeSkill.minion.level,
				hostile = activeSkill.minion.hostile,
				weaponData1 = scalars(activeSkill.minion.weaponData1),
				weaponData2 = scalars(activeSkill.minion.weaponData2),
			} or nil,
		}
	end
	emit(name .. ".skillLists", skillLists)

	-- Run the perform body (tail stubbed above) over the same env and
	-- capture its state. Everything above is post-initEnv; these records
	-- are post-perform-body.
	calcs.perform(env)
	emit(name .. ".performDbs", {
		mod = dbState(env.modDB),
		enemy = dbState(env.enemyDB),
		item = dbState(env.itemModDB),
	})
	emit(name .. ".performOutput", scalars(env.player.output or {}))
	if env.minion then
		emit(name .. ".performMinionDb", dbState(env.minion.modDB))
		emit(name .. ".performMinionOutput", scalars(env.minion.output or {}))
	end

	-- Defence stage, run explicitly on the post-perform-body state (the
	-- reference reaches it at CalcPerform L3721). buildDefenceEstimations
	-- and the offence side stay stubbed, so these records are exactly what
	-- calcs.defence produces. The player and minion calls are back-to-back
	-- here, where the reference interleaves buildDefenceEstimations between
	-- them; the replay does the same, so the comparison stays honest.
	realDefence(env, env.player)
	if env.minion then
		realDefence(env, env.minion)
	end
	emit(name .. ".defenceDbs", {
		mod = dbState(env.modDB),
		enemy = dbState(env.enemyDB),
		item = dbState(env.itemModDB),
	})
	emit(name .. ".defenceOutput", scalars(env.player.output or {}))
	if env.minion then
		emit(name .. ".defenceMinionDb", dbState(env.minion.modDB))
		emit(name .. ".defenceMinionOutput", scalars(env.minion.output or {}))
	end

	-- EHP / max-hit stage (CalcPerform L3723). Same technique: run it
	-- explicitly on the post-defence state, player then minion.
	realBuildDefenceEstimations(env, env.player)
	if env.minion then
		realBuildDefenceEstimations(env, env.minion)
	end
	emit(name .. ".ehpDbs", {
		mod = dbState(env.modDB),
		enemy = dbState(env.enemyDB),
		item = dbState(env.itemModDB),
	})
	emit(name .. ".ehpOutput", scalars(env.player.output or {}))
	if env.minion then
		emit(name .. ".ehpMinionDb", dbState(env.minion.modDB))
		emit(name .. ".ehpMinionOutput", scalars(env.minion.output or {}))
	end
end

if key == "empty" then
	newBuild()
	runCallback("OnFrame")
	dumpVariant("empty", build)
else
	assert(buildPath, "usage: dump_calc.lua <key> <build xml path>")
	local f = assert(io.open(buildPath, "r"))
	local xml = f:read("*a")
	f:close()
	loadBuildFromXML(xml, key)
	runCallback("OnFrame")

	dumpVariant(key .. ".full", build)

	-- progressively strip: no skills, then tree only
	wipeTable(build.skillsTab.socketGroupList)
	dumpVariant(key .. ".noskills", build)

	wipeTable(build.itemsTab.items)
	wipeTable(build.skillsTab.socketGroupList)
	dumpVariant(key .. ".treeonly", build)
end

out:close()
print("dumped calc fixtures: " .. key)
