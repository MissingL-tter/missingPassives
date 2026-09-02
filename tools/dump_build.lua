-- Dumps one loaded build: the fixture every port stage reads from it
-- (build, items, skills, spec, config, calc) and the calc's per-stage
-- checkpoints, for the archive
-- comparison. One process per corpus source (in-process build reloads keep
-- stale state):
--
--   luajit ../../tools/dump_build.lua empty
--   luajit ../../tools/dump_build.lua coc "Builds/3.29 CoC Blazing Salvo Assassin CI.xml"
--
-- Run from .archive/src/. Writes test/testdata/build_<key>.jsonl containing,
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
-- BEFORE LoadModule scopes this to exactly the Calc* modules. It exists to
-- make the dump reproducible, and normalises ONLY what LuaJIT itself does
-- not keep stable across processes:
--   numeric keys : left in LuaJIT's own next() order. That order is stable
--                  per process (measured 2026-09-01: identical across runs,
--                  and not ascending), so it is real reference behaviour
--                  and is recorded as-is. It used to be sorted here, which
--                  meant the dump never held PoB's actual node order.
--   string keys  : sorted ascending. LuaJIT randomises string hashing per
--                  process, so there is no stable order to record.
--   table keys   : see below.
-- Numeric keys come first, then strings, then tables; real pairs() would
-- interleave hash-part numerics with strings, but string order is already
-- invented, so that boundary carries no information either way.
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
	elseif #otherKeys > 1 and dumpBuild and dumpBuild.skillsTab then
		-- Socket-group-keyed sets (supportLists[slotName]): the granted-skill
		-- support gather pairs() over them, so their order reaches
		-- appliedSupportList and from there the compared mod lists. Order by
		-- the group's position in socketGroupList -- the order the Go replay
		-- iterates naturally. Groups created mid-initEnv are t_insert'ed into
		-- the same list, so the scan finds every key or leaves raw order.
		local pos = {}
		local all = true
		for _, k in ipairs(otherKeys) do
			local found
			for i, g in ipairs(dumpBuild.skillsTab.socketGroupList) do
				if g == k then
					found = i
					break
				end
			end
			if not found then
				all = false
				break
			end
			pos[k] = found
		end
		if all then
			table.sort(otherKeys, function(a, b) return pos[a] < pos[b] end)
		end
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

-- Data.lua:1039 sets `grantedEffect.statMap._grantedEffect = grantedEffect`
-- inside a pairs(data.skills) loop. Two skills can share one statMap table
-- (ExplosiveTrapAltX aliases ExplosiveTrap's), so the backref is
-- last-writer-wins over an order that varies per process -- and the lazy
-- statMap copies then stamp whichever skill won as the mod source, even for
-- the other skill. Settle it in sorted id order, matching the same
-- re-assignment dump_gamedata makes.
do
	local ids = {}
	for id in rawPairs(data.skills) do
		ids[#ids + 1] = id
	end
	table.sort(ids)
	-- The mod SOURCES on shared tables are last-writer-wins under the same
	-- pairs(data.skills) loop, so re-stamp them in sorted id order too --
	-- the exact reassign dump_gamedata makes. Without this a shared statMap
	-- entry's source (ExplosiveTrap vs ExplosiveTrapAltX) flips per process.
	local function reassign(el, modSource, id)
		if type(el) ~= "table" then
			return
		end
		if rawget(el, "source") ~= nil then
			el.source = modSource
		end
		local v = rawget(el, "value")
		if type(v) == "table" and type(rawget(v, "mod")) == "table" and rawget(rawget(v, "mod"), "source") ~= nil then
			rawget(v, "mod").source = "Skill:" .. id
		end
		for _, inner in ipairs(el) do
			reassign(inner, modSource, id)
		end
	end
	for _, id in ipairs(ids) do
		local ge = data.skills[id]
		local modSource = "Skill:" .. id
		for _, list in ipairs({ ge.baseMods, ge.qualityMods, ge.levelMods }) do
			if type(list) == "table" then
				for _, el in ipairs(list) do
					reassign(el, modSource, id)
				end
			end
		end
		if ge.statMap then
			ge.statMap._grantedEffect = ge
			for sk, entry in rawPairs(ge.statMap) do
				if sk ~= "_grantedEffect" then
					for _, el in ipairs(entry) do
						reassign(el, modSource, id)
					end
				end
			end
		end
	end
end

local calcs = LoadModule("Modules/Calcs")
pairs = rawPairs

-- The tree merge iterates pairs(nodeList): LuaJIT hash order, deterministic
-- per table state but not derivable in Go — and the table GROWS mid-initEnv
-- (granted passives), so each call sees its own order. Wrap the function to
-- record the order per call.

-- The perform checkpoint covers the perform BODY only: the final
-- defence/offence handoff (CalcPerform L3721+) is stubbed out so the
-- captured state is exactly what the ported body must reproduce.
-- calcTotemLife stays real (mid-body call for totem skills).
-- The real handoff functions are kept so later checkpoints can run them
-- explicitly, one stage at a time, on the post-perform-body state.
local realDefence = calcs.defence
local realBuildDefenceEstimations = calcs.buildDefenceEstimations
local realTriggers = calcs.triggers
local realMirages = calcs.mirages
local realOffence = calcs.offence
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
	-- env.extraRadiusNodeList is the one numeric-keyed table whose pairs()
	-- order is NOT reproducible across processes under this harness
	-- (measured 2026-09-01: 44 of 98 finishJewels walks differed between two
	-- full regenerations - same key set, alloc prefix intact, order diverging
	-- part-way through the tail). allocNodes, by contrast, was identical in
	-- all 98. Cause: TreeData/<ver>/tree.lua puts one string key, ["root"],
	-- in the numeric-keyed nodes constructor; LuaJIT seeds string hashes per
	-- process, so its slot displaces colliding numeric keys and the layout
	-- stays perturbed after PassiveTree deletes it. pairs(tree.nodes) is
	-- therefore per-process, nodesInRadius is filled from that walk, and
	-- this table from nodesInRadius - the reference's own order here is
	-- random. Rebuild it with keys inserted ascending so LuaJIT lays it out
	-- the same way every run. A referee modification, listed in knowledge.md
	-- 4.2 with the alternative in later.md 4; it normalises layout only.
	if finishJewels and env.extraRadiusNodeList and next(env.extraRadiusNodeList) then
		local ids = {}
		for id in rawPairs(env.extraRadiusNodeList) do
			ids[#ids + 1] = id
		end
		table.sort(ids)
		local rebuilt = {}
		for _, id in ipairs(ids) do
			rebuilt[id] = env.extraRadiusNodeList[id]
		end
		env.extraRadiusNodeList = rebuilt
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

local out = assert(io.open("../../test/testdata/build_" .. key .. ".jsonl", "w"))
local function emit(name, value)
	out:write('{"k":', canon.quote(name), ',"c":', canon.quote(canon.encode(value)), "}\n")
end

-- Fixtures are replay input, not compared canon: emit them round-trippably
-- so the Go side starts from the same doubles the reference had.
local function emitFixture(name, value)
	out:write('{"k":', canon.quote(name), ',"c":', canon.quote(canon.encodeExact(value)), "}\n")
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

-- GlobalCache snapshot for the trigger stage. Per uuid: the entry's own
-- scalar fields plus the scalar slices of the cached env that CalcTriggers
-- reaches into (player output incl. the MainHand/OffHand sub-tables, the
-- cached main skill's skillData, and the cached ActiveSkill's skillData).
local function cacheState(env)
	local o = {}
	for uuid, entry in pairs(GlobalCache.cachedData[env.mode] or {}) do
		local e = {
			Name = entry.Name,
			Speed = entry.Speed,
			HitSpeed = entry.HitSpeed,
			ManaCost = entry.ManaCost,
			LifeCost = entry.LifeCost,
			ESCost = entry.ESCost,
			RageCost = entry.RageCost,
			HitChance = entry.HitChance,
			AccuracyHitChance = entry.AccuracyHitChance,
			PreEffectiveCritChance = entry.PreEffectiveCritChance,
			CritChance = entry.CritChance,
			TotalDPS = entry.TotalDPS,
		}
		local cachedEnv = entry.Env
		if cachedEnv and cachedEnv.player then
			local po = cachedEnv.player.output or {}
			e.output = scalars(po)
			e.outputMainHand = scalars(po.MainHand or {})
			e.outputOffHand = scalars(po.OffHand or {})
			if cachedEnv.player.mainSkill then
				e.mainSkillData = scalars(cachedEnv.player.mainSkill.skillData or {})
			end
		end
		if entry.ActiveSkill then
			e.activeSkillData = scalars(entry.ActiveSkill.skillData or {})
		end
		o[uuid] = e
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
		-- Spec-computed (PassiveSpec:SetNodeDistanceToClassStart); CalcSetup
		-- L959 scales a Split Personality socket's effect by it.
		distanceToClassStart = node.distanceToClassStart,
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
	-- GlobalCache is a global, and the progressive strip below mutates one
	-- loaded build in place, so without this a reduced variant would inherit
	-- the full build's cache. wipeGlobalCache immediately before
	-- calcs.buildOutput is exactly what Build.lua:675 does; the real
	-- defence/offence functions have to be back for it, since the app's fill
	-- runs them.
	calcs.defence = realDefence
	calcs.buildDefenceEstimations = realBuildDefenceEstimations
	calcs.triggers = realTriggers
	calcs.mirages = realMirages
	calcs.offence = realOffence
	wipeGlobalCache()
	calcs.buildOutput(build, "MAIN")
	calcs.defence = function() end
	calcs.buildDefenceEstimations = function() end
	calcs.triggers = function() end
	calcs.mirages = function() return true end
	calcs.offence = function() end
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
	emitFixture(name .. ".fixture", fixture)

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
	emitFixture(name .. ".grantedPassiveNodes", grantedPassiveNodes)
	-- Granted ascendancy nodes (Forbidden Flame/Flesh) resolve through the
	-- ascendancy maps; emit the resolved nodes for the replay.
	local grantedAscendancyNodes = {}
	for _, ascTbl in pairs(env.modDB:List(nil, "GrantedAscendancyNode")) do
		local node = env.spec.tree.ascendancyMap[ascTbl.name] or build.latestTree.ascendancyMap[ascTbl.name]
		if node then
			grantedAscendancyNodes[ascTbl.name] = nodeFixture(node)
		end
	end
	emitFixture(name .. ".grantedAscendancyNodes", grantedAscendancyNodes)
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
	emitFixture(name .. ".energyBladeItems", energyBladeItems)
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

	-- GlobalCache snapshot, taken immediately before calcs.triggers reads
	-- it. The cache is filled by the app's own buildOutput during OnFrame
	-- (one entry per skill), then the dump's calcs.perform overwrites the
	-- main skill's entry with a pre-offence one. Triggers is the only stage
	-- that reads it, and it reads a bounded set of fields; the replay
	-- consumes this as a fixture rather than re-deriving it, exactly like
	-- allocOrders/energyBladeItems. Enumerated from:
	--   grep -oE "GlobalCache\.cachedData\[env\.mode\]\[[a-zA-Z]+\]\.[A-Za-z.]+" CalcTriggers.lua
	emit(name .. ".globalCache", cacheState(env))

	-- Offence stage (CalcPerform L3726-3729), run explicitly on the
	-- post-EHP state. calcs.triggers runs first because offence reads the
	-- trigger rate it writes; mirages decides whether offence runs at all.
	realTriggers(env, env.player)
	-- Triggers gets its own checkpoint: it is a separate 1.6k-line module
	-- and offence reads the trigger rate it writes, so porting it first
	-- gives an independently verifiable milestone.
	emit(name .. ".triggersDbs", {
		mod = dbState(env.modDB),
		enemy = dbState(env.enemyDB),
		item = dbState(env.itemModDB),
	})
	emit(name .. ".triggersOutput", scalars(env.player.output or {}))
	emit(name .. ".triggersSkillData", scalars(env.player.mainSkill.skillData or {}))
	-- The mirage paths build a whole second environment (copyActiveSkill's
	-- calcs.initEnv + a nested calcs.perform), so the node orders that
	-- second initEnv consumes have to be recorded too -- and the nested
	-- perform sees the same stubbed handoff as the outer one, so its output
	-- is the body-only state the Go replay produces.
	recordedOrders = {}
	recordedNodeSeqs = {}
	local mirageHandled = realMirages(env)
	emit(name .. ".mirageAllocOrders", recordedOrders)
	emit(name .. ".mirageNodeOrders", recordedNodeSeqs)
	recordedOrders = nil
	recordedNodeSeqs = nil
	local mirage = env.player.mainSkill.mirage
	if mirage then
		emit(name .. ".mirage", {
			name = mirage.name,
			count = mirage.count,
			skillPart = mirage.skillPart,
			skillPartName = mirage.skillPartName,
			handled = mirageHandled,
		})
		emit(name .. ".mirageOutput", scalars(mirage.output or {}))
	end
	if not mirageHandled then
		realOffence(env, env.player, env.player.mainSkill)
	end
	emit(name .. ".offenceDbs", {
		mod = dbState(env.modDB),
		enemy = dbState(env.enemyDB),
		item = dbState(env.itemModDB),
	})
	emit(name .. ".offenceOutput", scalars(env.player.output or {}))
	-- The main skill's own output bag is where offence puts most of its
	-- product (damage, speed, DPS); the player output only carries the
	-- summary.
	emit(name .. ".offenceSkillOutput", scalars(env.player.mainSkill.skillData or {}))
	if env.minion then
		realTriggers(env, env.minion)
		emit(name .. ".triggersMinionOutput", scalars(env.minion.output or {}))
		emit(name .. ".triggersMinionSkillData", scalars(env.minion.mainSkill.skillData or {}))
		realOffence(env, env.minion, env.minion.mainSkill)
		emit(name .. ".offenceMinionDb", dbState(env.minion.modDB))
		emit(name .. ".offenceMinionOutput", scalars(env.minion.output or {}))
	end
end

if key == "empty" then
	newBuild()
	dumpBuild = build
	runCallback("OnFrame")
	dumpVariant("empty", build)
else
	assert(buildPath, "usage: dump_build.lua <key> <build xml path>")
	local f = assert(io.open(buildPath, "r"))
	local xml = f:read("*a")
	f:close()
	loadBuildFromXML(xml, key)
	dumpBuild = build
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
print("dumped build: " .. key)
