-- Path of Building
--
-- Module: Config Tab
-- Configuration state for the current build.
--
local t_insert = table.insert
local m_min = math.min
local s_upper = string.upper

local varList = LoadModule("Modules/ConfigOptions")

local ConfigTabClass = newClass("ConfigTab", "UndoHandler", function(self, build)
	self.UndoHandler()

	self.build = build

	self.input = { }
	self.placeholder = { }
	self.defaultState = { }

	-- Initialise config sets
	self.configSets = { }
	self.configSetOrderList = { 1 }
	self:NewConfigSet(1)
	self:SetActiveConfigSet(1, true)

	self.enemyLevel = 1

	-- A misc calculator function which is updated by the build when it is rebuilt
	---@type fun(): table
	self.calcFunc = nil
	-- A calculator base output matching the calcFunc which is updated by the build when it is rebuilt
	---@type table
	self.calcBase = nil

	-- Per-option state accessors; config option `apply` functions and importers
	-- use these to read list selections and set placeholder values
	self.varControls = { }

	-- Record the default state of every config option; used to omit defaults when saving
	for _, varData in ipairs(varList) do
		if varData.var then
			if varData.type == "check" then
				self.defaultState[varData.var] = varData.defaultState or false
			elseif varData.type == "count" or varData.type == "integer" or varData.type == "countAllowZero" or varData.type == "float" then
				self.defaultState[varData.var] = varData.defaultState or 0
			elseif varData.type == "list" then
				self.defaultState[varData.var] = varData.list[varData.defaultIndex or 1].val
			elseif varData.type == "text" then
				self.defaultState[varData.var] = varData.defaultState or ""
			else
				self.defaultState[varData.var] = varData.defaultState
			end
			self.varControls[varData.var] = self:NewVarControl(varData)
		end
	end
end)

-- Creates the state accessor for one config option
function ConfigTabClass:NewVarControl(varData)
	local configTab = self
	local varControl = { varData = varData, list = varData.list }

	local function implyCond()
		local mainEnv = configTab.build.calcsTab.mainEnv
		if configTab.configSets[configTab.activeConfigSetId].input[varData.var] then
			if varData.implyCondList then
				for _, implyCond in ipairs(varData.implyCondList) do
					if (implyCond and mainEnv.conditionsUsed[implyCond]) then
						return true
					end
				end
			end
			if (varData.implyCond and mainEnv.conditionsUsed[varData.implyCond]) or
			   (varData.implyMinionCond and mainEnv.minionConditionsUsed[varData.implyMinionCond]) or
			   (varData.implyEnemyCond and mainEnv.enemyConditionsUsed[varData.implyEnemyCond]) then
				return true
			end
		end

		return false
	end

	local function listOrSingleIfOption(ifOption, ifFunc)
		return function()
			if type(ifOption) == "table" then
				for _, ifOpt in ipairs(ifOption) do
					if ifFunc(ifOpt) then
						return true
					end
				end
			end
			return ifFunc(ifOption)
		end
	end

	-- Conditions that determine whether the option is relevant to the current build
	local shownFuncs = {}
	if varData.ifNode then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifNode, function(ifOption)
			if configTab.build.spec.allocNodes[ifOption] then
				return true
			end
			local node = configTab.build.spec.nodes[ifOption]
			if node and node.type == "Keystone" then
				return configTab.build.calcsTab.mainEnv.keystonesAdded[node.dn]
			end
		end))
	end
	if varData.ifOption then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifOption, function(ifOption)
			return configTab.configSets[configTab.activeConfigSetId].input[ifOption]
		end))
	end
	if varData.ifCond then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifCond, function(ifOption)
			if implyCond() then
				return true
			end
			return configTab.build.calcsTab.mainEnv.conditionsUsed[ifOption]
		end))
	end
	if varData.ifMinionCond then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifMinionCond, function(ifOption)
			if implyCond() then
				return true
			end
			return configTab.build.calcsTab.mainEnv.minionConditionsUsed[ifOption]
		end))
	end
	if varData.ifEnemyCond then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifEnemyCond, function(ifOption)
			if implyCond() then
				return true
			end
			return configTab.build.calcsTab.mainEnv.enemyConditionsUsed[ifOption]
		end))
	end
	if varData.ifCondTrue then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifCondTrue, function(ifOption)
			return configTab.build.calcsTab.mainEnv.player.modDB.conditions[ifOption]
		end))
	end
	if varData.ifMult then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifMult, function(ifOption)
			if implyCond() then
				return true
			end
			return configTab.build.calcsTab.mainEnv.multipliersUsed[ifOption]
		end))
	end
	if varData.ifEnemyMult then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifEnemyMult, function(ifOption)
			if implyCond() then
				return true
			end
			return configTab.build.calcsTab.mainEnv.enemyMultipliersUsed[ifOption]
		end))
	end
	if varData.ifStat then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifStat, function(ifOption)
			if implyCond() then
				return true
			end
			return configTab.build.calcsTab.mainEnv.perStatsUsed[ifOption] or configTab.build.calcsTab.mainEnv.enemyMultipliersUsed[ifOption]
		end))
	end
	if varData.ifEnemyStat then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifEnemyStat, function(ifOption)
			if implyCond() then
				return true
			end
			return configTab.build.calcsTab.mainEnv.enemyPerStatsUsed[ifOption]
		end))
	end
	if varData.ifTagType then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifTagType, function(ifOption)
			if implyCond() then
				return true
			end
			return configTab.build.calcsTab.mainEnv.tagTypesUsed[ifOption]
		end))
	end
	if varData.ifFlag then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifFlag, function(ifOption)
			local skillModList = configTab.build.calcsTab.mainEnv.player.mainSkill.skillModList
			local skillFlags = configTab.build.calcsTab.mainEnv.player.mainSkill.skillFlags
			-- Check both the skill mods for flags and flags that are set via calcPerform
			return skillFlags[ifOption] or skillModList:Flag(nil, ifOption)
		end))
	end
	if varData.ifMod then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifMod, function(ifOption)
			if implyCond() then
				return true
			end
			return configTab.build.calcsTab.mainEnv.modsUsed[ifOption]
		end))
	end
	if varData.ifSkill then
		if varData.includeTransfigured then
			t_insert(shownFuncs, listOrSingleIfOption(varData.ifSkill, function(ifOption)
				if not calcLib.getGameIdFromGemName(ifOption, true) then
					return false
				end
				for skill,_ in pairs(configTab.build.calcsTab.mainEnv.skillsUsed) do
					if calcLib.isGemIdSame(skill, ifOption, true) then
						return true
					end
				end
				return false
			end))
		else
			t_insert(shownFuncs, listOrSingleIfOption(varData.ifSkill, function(ifOption)
				return configTab.build.calcsTab.mainEnv.skillsUsed[ifOption]
			end))
		end
	end
	if varData.ifSkillFlag then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifSkillFlag, function(ifOption)
			for _, activeSkill in ipairs(configTab.build.calcsTab.mainEnv.player.activeSkillList) do
				if activeSkill.skillFlags[ifOption] then
					return true
				end
			end
			return false
		end))
	end
	if varData.ifSkillData then
		t_insert(shownFuncs, listOrSingleIfOption(varData.ifSkillData, function(ifOption)
			for _, activeSkill in ipairs(configTab.build.calcsTab.mainEnv.player.activeSkillList) do
				if activeSkill.skillData[ifOption] then
					return true
				end
			end
			return false
		end))
	end

	local function innerShown()
		for _, shownFunc in ipairs(shownFuncs) do
			if not shownFunc() then
				return false
			end
		end
		return true
	end

	-- Whether the option applies to the current build; a set non-default value
	-- also shows, so an invalid state stays observable
	varControl.shown = function()
		local shown = innerShown()
		if varData.hideIfInvalid then
			return shown
		end
		local cur = configTab.configSets[configTab.activeConfigSetId].input[varData.var]
		local def = configTab:GetDefaultState(varData.var, type(cur))
		return not shown and cur ~= nil and cur ~= def or shown
	end
	-- The selected index of a list option, derived from the current input value
	setmetatable(varControl, { __index = function(t, key)
		if key == "selIndex" then
			local cur = configTab.configSets[configTab.activeConfigSetId].input[varData.var]
			if varData.list then
				for i, option in ipairs(varData.list) do
					if option.val == cur then
						return i
					end
				end
			end
			return varData.defaultIndex or 1
		end
	end })
	function varControl:SetPlaceholder(text, notify)
		self.placeholder = tostring(text)
		if notify then
			local configSet = configTab.configSets[configTab.activeConfigSetId]
			if varData.type == "count" or varData.type == "integer" or varData.type == "countAllowZero" or varData.type == "float" then
				configSet.placeholder[varData.var] = tonumber(text)
			else
				configSet.placeholder[varData.var] = tostring(text)
			end
			configTab.build.buildFlag = true
		end
	end
	function varControl:SetSel(newSel, noCallSelFunc)
		if not noCallSelFunc then
			configTab:SetOptionByIndex(varData.var, newSel)
		end
	end
	return varControl
end

function ConfigTabClass:Load(xml, fileName)
	self.activeConfigSetId = 1
	self.configSets = { }
	self.configSetOrderList = { 1 }

	local function setInputAndPlaceholder(node, configSetId)
		if node.elem == "Input" then
			if not node.attrib.name then
				launch:ShowErrMsg("^1Error parsing '%s': 'Input' element missing name attribute", fileName)
				return true
			end
			if node.attrib.number then
				self.configSets[configSetId].input[node.attrib.name] = tonumber(node.attrib.number)
			elseif node.attrib.string then
				if node.attrib.name == "enemyIsBoss" then
					self.configSets[configSetId].input[node.attrib.name] = node.attrib.string:lower():gsub("(%l)(%w*)", function(a,b) return s_upper(a)..b end)
					:gsub("Uber Atziri", "Boss"):gsub("Shaper", "Pinnacle"):gsub("Sirus", "Pinnacle")
				-- backwards compat <=3.20, Uber Atziri Flameblast -> Atziri Flameblast
				elseif node.attrib.name == "presetBossSkills" then
					self.configSets[configSetId].input[node.attrib.name] = node.attrib.string:gsub("^Uber ", "")
				else
					self.configSets[configSetId].input[node.attrib.name] = node.attrib.string
				end
			elseif node.attrib.boolean then
				self.configSets[configSetId].input[node.attrib.name] = node.attrib.boolean == "true"
			else
				launch:ShowErrMsg("^1Error parsing '%s': 'Input' element missing number, string or boolean attribute", fileName)
				return true
			end
		elseif node.elem == "Placeholder" then
			if not node.attrib.name then
				launch:ShowErrMsg("^1Error parsing '%s': 'Placeholder' element missing name attribute", fileName)
				return true
			end
			if node.attrib.number then
				self.configSets[configSetId].placeholder[node.attrib.name] = tonumber(node.attrib.number)
			elseif node.attrib.string then
				self.configSets[configSetId].input[node.attrib.name] = node.attrib.string
			else
				launch:ShowErrMsg("^1Error parsing '%s': 'Placeholder' element missing number", fileName)
				return true
			end
		end
	end

	-- Catch special case of empty Config
	if xml.empty then
		self:NewConfigSet(1, "Default")
	end
	for index, node in ipairs(xml) do
		if node.elem ~= "ConfigSet" then
			if not self.configSets[1] then
				self:NewConfigSet(1, "Default")
			end
			if node.elem == "CustomModifierBlock" then
				local block = {
					title = node.attrib.title or "Default",
					enabled = (node.attrib.enabled == "true" or node.attrib.enabled == nil),
					text = node[1] or ""
				}
				t_insert(self.configSets[1].customModsList, block)
			else
				setInputAndPlaceholder(node, 1)
			end
		else
			local configSetId = tonumber(node.attrib.id)
			self:NewConfigSet(configSetId, node.attrib.title or "Default")
			self.configSetOrderList[index] = configSetId
			self.configSets[configSetId].customModsList = { }
			for _, child in ipairs(node) do
				if child.elem == "CustomModifierBlock" then
					local block = {
						title = child.attrib.title or "Default",
						enabled = (child.attrib.enabled == "true" or child.attrib.enabled == nil),
						text = child[1] or ""
					}
					t_insert(self.configSets[configSetId].customModsList, block)
				else
					setInputAndPlaceholder(child, configSetId)
				end
			end
		end
	end

	-- Migration check for legacy builds
	for _, configSetId in ipairs(self.configSetOrderList) do
		local configSet = self.configSets[configSetId]
		local legacyText = configSet.input and configSet.input.customMods or ""
		if legacyText ~= "" and (not configSet.customModsList or #configSet.customModsList == 0 or (#configSet.customModsList == 1 and (configSet.customModsList[1].text or "") == "")) then
			configSet.customModsList = { { title = "Default", enabled = true, text = legacyText } }
		elseif not configSet.customModsList or #configSet.customModsList == 0 then
			configSet.customModsList = { { title = "Default", enabled = true, text = "" } }
		end
		if configSet.input then
			configSet.input.customMods = nil
		end
	end

	self:SetActiveConfigSet(tonumber(xml.attrib.activeConfigSet) or 1)
	self:ResetUndo()
end

function ConfigTabClass:GetDefaultState(var, varType)
	if self.configSets[self.activeConfigSetId].placeholder[var] ~= nil then
		return self.configSets[self.activeConfigSetId].placeholder[var]
	end

	if self.defaultState[var] ~= nil then
		return self.defaultState[var]
	end

	if varType == "number" then
		return 0
	elseif varType == "boolean" then
		return false
	elseif varType == "string" then
		return ""
	else
		return nil
	end
end

function ConfigTabClass:Save(xml)
	xml.attrib = {
		activeConfigSet = tostring(self.activeConfigSetId)
	}
	for _, configSetId in ipairs(self.configSetOrderList) do
		local configSet = self.configSets[configSetId]
		local child = { elem = "ConfigSet", attrib = { id = tostring(configSetId), title = configSet.title } }
		t_insert(xml, child)

		for k, v in pairs(configSet.input) do
			if v ~= self:GetDefaultState(k, type(v)) then
				local node = { elem = "Input", attrib = { name = k } }
				if type(v) == "number" then
					node.attrib.number = tostring(v)
				elseif type(v) == "boolean" then
					node.attrib.boolean = tostring(v)
				else
					node.attrib.string = tostring(v)
				end
				t_insert(child, node)
			end
		end
		for k, v in pairs(configSet.placeholder) do
			local node = { elem = "Placeholder", attrib = { name = k } }
			if type(v) == "number" then
				node.attrib.number = tostring(v)
			else
				node.attrib.string = tostring(v)
			end
			t_insert(child, node)
		end
		if configSet.customModsList then
			for _, block in ipairs(configSet.customModsList) do
				local blockNode = {
					elem = "CustomModifierBlock",
					attrib = {
						title = block.title or "Default",
						enabled = tostring(block.enabled ~= false)
					},
					[1] = block.text or ""
				}
				t_insert(child, blockNode)
			end
		end
	end
end

-- Sets a config option to the given value, as the config UI would
function ConfigTabClass:SetOption(varName, value)
	self.configSets[self.activeConfigSetId].input[varName] = value
	self:AddUndoState()
	self:BuildModList()
	self.build.buildFlag = true
end

-- Sets a list-type config option by the index of the wanted option
function ConfigTabClass:SetOptionByIndex(varName, index)
	for _, varData in ipairs(varList) do
		if varData.var == varName and varData.list then
			self:SetOption(varName, varData.list[index].val)
			return
		end
	end
end

function ConfigTabClass:UpdateLevel()
	local input = self.configSets[self.activeConfigSetId].input
	local placeholder = self.configSets[self.activeConfigSetId].placeholder
	if input.enemyLevel and input.enemyLevel > 0 then
		self.enemyLevel = m_min(data.misc.MaxEnemyLevel, input.enemyLevel)
	elseif placeholder.enemyLevel and placeholder.enemyLevel > 0 then
		self.enemyLevel = m_min(data.misc.MaxEnemyLevel, placeholder.enemyLevel)
	else
		self.enemyLevel = m_min(data.misc.MaxEnemyLevel, self.build.characterLevel)
	end
end

function ConfigTabClass:BuildModList()
	local modList = new("ModList")
	self.modList = modList
	local enemyModList = new("ModList")
	self.enemyModList = enemyModList
	local input = self.configSets[self.activeConfigSetId].input
	local placeholder = self.configSets[self.activeConfigSetId].placeholder
	self:UpdateLevel() -- enemy level handled here because it's needed to correctly set boss stats
	for _, varData in ipairs(varList) do
		if varData.apply then
			if varData.type == "check" then
				if input[varData.var] then
					varData.apply(true, modList, enemyModList, self.build)
				end
			elseif varData.type == "count" or varData.type == "integer" or varData.type == "countAllowZero" or varData.type == "float" then
				if input[varData.var] and (input[varData.var] ~= 0 or varData.type == "countAllowZero") then
					varData.apply(input[varData.var], modList, enemyModList, self.build)
				elseif placeholder[varData.var] and (placeholder[varData.var] ~= 0 or varData.type == "countAllowZero") then
					varData.apply(placeholder[varData.var], modList, enemyModList, self.build)
				end
			elseif varData.type == "list" then
				if input[varData.var] then
					varData.apply(input[varData.var], modList, enemyModList, self.build)
				end
			elseif varData.type == "text" then
				if input[varData.var] then
					varData.apply(input[varData.var], modList, enemyModList, self.build)
				end
			end
		end
	end

	-- Apply Custom Modifier groups
	local customModsList = self.configSets[self.activeConfigSetId].customModsList
	local hasBlockText = false
	if customModsList then
		for _, block in ipairs(customModsList) do
			if block.enabled ~= false and block.text and #block.text > 0 then
				hasBlockText = true
				for line in block.text:gmatch("([^\n]*)\n?") do
					local strippedLine = StripEscapes(line):match("^%s*(.-)%s*$")
					local mods, extra = modLib.parseMod(strippedLine)
					if mods and not extra then
						local source = "Custom:" .. (block.title or "Default")
						for i = 1, #mods do
							local mod = mods[i]
							if mod then
								mod = modLib.setSource(mod, source)
								modList:AddMod(mod)
							end
						end
					end
				end
			end
		end
	end
	-- Fallback for tests/headless
	if not hasBlockText and input.customMods and #input.customMods > 0 then
		for line in input.customMods:gmatch("([^\n]*)\n?") do
			local strippedLine = StripEscapes(line):match("^%s*(.-)%s*$")
			local mods, extra = modLib.parseMod(strippedLine)
			if mods and not extra then
				local source = "Custom"
				for i = 1, #mods do
					local mod = mods[i]
					if mod then
						mod = modLib.setSource(mod, source)
						modList:AddMod(mod)
					end
				end
			end
		end
	end
end

function ConfigTabClass:ImportCalcSettings()
	local input = self.configSets[self.activeConfigSetId].input
	local calcsInput = self.build.calcsTab.input
	local function import(old, new)
		input[new] = calcsInput[old]
		calcsInput[old] = nil
	end
	import("Cond_LowLife", "conditionLowLife")
	import("Cond_FullLife", "conditionFullLife")
	import("Cond_LowMana", "conditionLowMana")
	import("Cond_FullMana", "conditionFullMana")
	import("buff_power", "usePowerCharges")
	import("buff_frenzy", "useFrenzyCharges")
	import("buff_endurance", "useEnduranceCharges")
	import("CondBuff_Onslaught", "buffOnslaught")
	import("CondBuff_Phasing", "buffPhasing")
	import("CondBuff_Fortify", "buffFortify")
	import("CondBuff_UsingFlask", "conditionUsingFlask")
	import("buff_pendulum", "usePendulum")
	import("CondEff_EnemyCursed", "conditionEnemyCursed")
	import("CondEff_EnemyBleeding", "conditionEnemyBleeding")
	import("CondEff_EnemyPoisoned", "conditionEnemyPoisoned")
	import("CondEff_EnemyBurning", "conditionEnemyBurning")
	import("CondEff_EnemyIgnited", "conditionEnemyIgnited")
	import("CondEff_EnemyChilled", "conditionEnemyChilled")
	import("CondEff_EnemyFrozen", "conditionEnemyFrozen")
	import("CondEff_EnemyShocked", "conditionEnemyShocked")
	import("effective_physicalRed", "enemyPhysicalReduction")
	import("effective_fireResist", "enemyFireResist")
	import("effective_coldResist", "enemyColdResist")
	import("effective_lightningResist", "enemyLightningResist")
	import("effective_chaosResist", "enemyChaosResist")
	import("effective_enemyIsBoss", "enemyIsBoss")
	self:BuildModList()
end

function ConfigTabClass:CreateUndoState()
	local configSet = self.configSets[self.activeConfigSetId]
	return {
		input = copyTable(configSet.input),
		customModsList = copyTable(configSet.customModsList)
	}
end

function ConfigTabClass:RestoreUndoState(state)
	local configSet = self.configSets[self.activeConfigSetId]
	if type(state) == "table" and state.input then
		wipeTable(configSet.input)
		for k, v in pairs(state.input) do
			configSet.input[k] = v
		end
		if state.customModsList then
			configSet.customModsList = copyTable(state.customModsList)
		end
	else
		wipeTable(configSet.input)
		for k, v in pairs(state) do
			configSet.input[k] = v
		end
	end
	self:BuildModList()
end

-- Creates a new config set
function ConfigTabClass:NewConfigSet(configSetId, title)
	local configSet = { id = configSetId, title = title, input = { }, placeholder = { }, customModsList = { { title = "Default", enabled = true, text = "" } } }
	if not configSetId then
		configSet.id = 1
		while self.configSets[configSet.id] do
			configSet.id = configSet.id + 1
		end
	end
	-- there are default values for input and placeholder that every new config set needs to have
	for _, varData in ipairs(varList) do
		if varData.var then
			configSet.input[varData.var] = varData.defaultState
			configSet.placeholder[varData.var] = varData.defaultPlaceholderState
			if varData.defaultIndex then
				configSet.input[varData.var] = varData.list[varData.defaultIndex].val
			end
		end
	end
	self.configSets[configSet.id] = configSet
	return configSet
end

-- Changes the active config set
function ConfigTabClass:SetActiveConfigSet(configSetId, init)
	-- Initialize config sets if needed
	if not self.configSetOrderList[1] then
		self.configSetOrderList[1] = 1
		self:NewConfigSet(1)
	end

	if not configSetId then
		configSetId = self.activeConfigSetId
	end

	if not self.configSets[configSetId] then
		configSetId = self.configSetOrderList[1]
	end

	self.input = self.configSets[configSetId].input
	self.placeholder = self.configSets[configSetId].placeholder
	self.activeConfigSetId = configSetId

	if not init then
		self:BuildModList()
	end
	self.build.buildFlag = true
	self.build:SyncLoadouts()
end
