-- Path of Building
--
-- Module: Skills Tab
-- Skill/socket group state for the current build.
--
local pairs = pairs
local ipairs = ipairs
local t_insert = table.insert
local m_min = math.min
local m_max = math.max

local SkillsTabClass = newClass("SkillsTab", "UndoHandler", function(self, build)
	self.UndoHandler()

	self.build = build

	self.socketGroupList = { }

	self.sortGemsByDPS = true
	self.sortGemsByDPSField = "CombinedDPS"
	self.showSupportGemTypes = "ALL"
	self.showLegacyGems = false
	self.defaultGemLevel = "normalMaximum"
	self.defaultGemQuality = main.defaultGemQuality

	-- self.imbuedSupportBySlot is used by CalcSetup to add an ExtraSupport mod of the selected gem
	-- Each socketGroup has its own "imbuedSupport" and is saved to the xml to load when changing sockets or loading a build
	self.imbuedSupportBySlot = { }

	-- Initialise skill sets
	self.skillSets = { }
	self.skillSetOrderList = { 1 }
	self:NewSkillSet(1)
	self:SetActiveSkillSet(1)

	self.controls = {
		optimiseSockets = { onClick = function()
			self:OptimiseSockets()
		end },
	}
end)

-- Rebuild the displayed group's item's sockets to match the groups assigned to it
function SkillsTabClass:OptimiseSockets()
	local slotName = self.displayGroup and self.displayGroup.slot
	local slot = slotName and self.build.itemsTab.slots[slotName]
	local item = slot and self.build.itemsTab.items[slot.selItemId]
	if not item or not item.base then
		return
	end

	self.build.itemsTab:AddUndoState()

	-- save count of abyssal sockets
	local abyssalSocketCount = 0
	for _, socket in ipairs(item.sockets) do
		if socket.color == "A" then
			abyssalSocketCount = abyssalSocketCount + 1
		end
	end

	local groupCount = 0
	item.sockets = {}
	local maxSockets = (item.base.socketLimit or 0) - abyssalSocketCount
	for _, group in ipairs(self.socketGroupList) do
		local colours = { "R", "G", "B" }
		if group.slot == slotName and group.source == nil then
			for _, gem in ipairs(group.gemList) do
				local grantedEffect = gem.grantedEffect or (gem.gemData and gem.gemData.grantedEffect)
				if grantedEffect and maxSockets > 0 then
					local gemColour = grantedEffect.color and colours[grantedEffect.color] or "W"
					table.insert(item.sockets, { color = gemColour, group = groupCount })
					maxSockets = maxSockets - 1
				end
			end
			groupCount = groupCount + 1
		end
	end

	for _ = 0, abyssalSocketCount - 1 do
		groupCount = groupCount + 1
		table.insert(item.sockets, { color = "A", group = groupCount })
	end
	item:BuildAndParseRaw()
	self:UpdateSocketGroups()
	self.build.buildFlag = true
end


function SkillsTabClass:LoadSkill(node, skillSetId)
	if node.elem ~= "Skill" then
		return
	end

	local socketGroup = { }
	socketGroup.enabled = node.attrib.active == "true" or node.attrib.enabled == "true"
	socketGroup.includeInFullDPS = node.attrib.includeInFullDPS and node.attrib.includeInFullDPS == "true"
	socketGroup.groupCount = tonumber(node.attrib.groupCount)
	socketGroup.label = node.attrib.label
	socketGroup.slot = node.attrib.slot
	socketGroup.source = node.attrib.source
	socketGroup.mainActiveSkill = tonumber(node.attrib.mainActiveSkill) or 1
	socketGroup.mainActiveSkillCalcs = tonumber(node.attrib.mainActiveSkillCalcs) or 1
	socketGroup.gemList = { }
	if node.attrib.imbuedSupport and node.attrib.slot then
		socketGroup.imbuedSupport = node.attrib.imbuedSupport
		local gemId = data.gemForBaseName[socketGroup.imbuedSupport:lower().." support"]
		local gem = gemId and data.gems[gemId]
		if gem and gem.grantedEffect then
			self.imbuedSupportBySlot[socketGroup.slot] = gem.grantedEffect
		end
	end

	for _, child in ipairs(node) do
		local gemInstance = { }
		gemInstance.nameSpec = sanitiseText(child.attrib.nameSpec or "")
		if child.attrib.gemId then
			local gemData
			local possibleVariants = self.build.data.gemsByGameId[child.attrib.gemId]
			if possibleVariants then
				-- If it is a known gem, try to determine which variant is used
				if child.attrib.variantId then
					-- New save format from 3.23 that stores the specific variation (transfiguration)
					gemData = possibleVariants[child.attrib.variantId]
				elseif child.attrib.skillId then
					-- Old format relying on the uniqueness of the granted effects id
					for _, variant in pairs(possibleVariants) do
						if variant.grantedEffectId == child.attrib.skillId then
							gemData = variant
							break
						end
					end
				end
			end
			if gemData then
				gemInstance.gemId = gemData.id
				gemInstance.skillId = gemData.grantedEffectId
				gemInstance.nameSpec = gemData.nameSpec
			end
		elseif child.attrib.skillId then
			local grantedEffect = self.build.data.skills[child.attrib.skillId]
			if grantedEffect then
				gemInstance.gemId = self.build.data.gemForSkill[grantedEffect]
				gemInstance.skillId = grantedEffect.id
				gemInstance.nameSpec = grantedEffect.name
			end
		end
		gemInstance.level = tonumber(child.attrib.level)
		gemInstance.quality = tonumber(child.attrib.quality)
		gemInstance.nameSpec = sanitiseText(gemInstance.nameSpec)
		gemInstance.enabled = not child.attrib.enabled and true or child.attrib.enabled == "true"
		gemInstance.enableGlobal1 = not child.attrib.enableGlobal1 or child.attrib.enableGlobal1 == "true"
		gemInstance.enableGlobal2 = child.attrib.enableGlobal2 == "true"
		gemInstance.count = tonumber(child.attrib.count) or 1
		gemInstance.skillPart = tonumber(child.attrib.skillPart)
		gemInstance.skillPartCalcs = tonumber(child.attrib.skillPartCalcs)
		gemInstance.skillStageCount = tonumber(child.attrib.skillStageCount)
		gemInstance.skillStageCountCalcs = tonumber(child.attrib.skillStageCountCalcs)
		gemInstance.skillMineCount = tonumber(child.attrib.skillMineCount)
		gemInstance.skillMineCountCalcs = tonumber(child.attrib.skillMineCountCalcs)
		gemInstance.skillMinion = child.attrib.skillMinion
		gemInstance.skillMinionCalcs = child.attrib.skillMinionCalcs
		gemInstance.skillMinionItemSet = tonumber(child.attrib.skillMinionItemSet)
		gemInstance.skillMinionItemSetCalcs = tonumber(child.attrib.skillMinionItemSetCalcs)
		gemInstance.skillMinionSkill = tonumber(child.attrib.skillMinionSkill)
		gemInstance.skillMinionSkillCalcs = tonumber(child.attrib.skillMinionSkillCalcs)
		t_insert(socketGroup.gemList, gemInstance)
	end
	if node.attrib.skillPart and socketGroup.gemList[1] then
		socketGroup.gemList[1].skillPart = tonumber(node.attrib.skillPart)
	end
	self:ProcessSocketGroup(socketGroup)
	t_insert(self.skillSets[skillSetId].socketGroupList, socketGroup)
end

function SkillsTabClass:Load(xml, fileName)
	self.activeSkillSetId = 0
	self.skillSets = { }
	self.skillSetOrderList = { }
	-- Handle legacy configuration settings when loading `defaultGemLevel`
	if xml.attrib.matchGemLevelToCharacterLevel == "true" then
		self.defaultGemLevel = "characterLevel"
	elseif type(xml.attrib.defaultGemLevel) == "string" and tonumber(xml.attrib.defaultGemLevel) == nil then
		self.defaultGemLevel = xml.attrib.defaultGemLevel
	else
		self.defaultGemLevel = "normalMaximum"
	end
	self.defaultGemQuality = m_max(m_min(tonumber(xml.attrib.defaultGemQuality) or 0, 23), 0)
	if xml.attrib.sortGemsByDPS then
		self.sortGemsByDPS = xml.attrib.sortGemsByDPS == "true"
	end
	if xml.attrib.showLegacyGems then
		self.showLegacyGems = xml.attrib.showLegacyGems == "true"
	end
	self.showSupportGemTypes = xml.attrib.showSupportGemTypes or "ALL"
	self.sortGemsByDPSField = xml.attrib.sortGemsByDPSField or "CombinedDPS"
	for _, node in ipairs(xml) do
		if node.elem == "Skill" then
			-- Old format, initialize skill sets if needed
			if not self.skillSetOrderList[1] then
				self.skillSetOrderList[1] = 1
				self:NewSkillSet(1)
			end
			self:LoadSkill(node, 1)
		end

		if node.elem == "SkillSet" then
			local skillSet = self:NewSkillSet(tonumber(node.attrib.id))
			skillSet.title = node.attrib.title
			t_insert(self.skillSetOrderList, skillSet.id)
			for _, subNode in ipairs(node) do
				self:LoadSkill(subNode, skillSet.id)
			end
		end
	end
	self:SetActiveSkillSet(tonumber(xml.attrib.activeSkillSet) or 1)
	self:ResetUndo()
end

function SkillsTabClass:Save(xml)
	xml.attrib = {
		activeSkillSet = tostring(self.activeSkillSetId),
		defaultGemLevel = self.defaultGemLevel,
		defaultGemQuality = tostring(self.defaultGemQuality),
		sortGemsByDPS = tostring(self.sortGemsByDPS),
		showSupportGemTypes = self.showSupportGemTypes,
		sortGemsByDPSField = self.sortGemsByDPSField,
		showLegacyGems = tostring(self.showLegacyGems),
	}
	for _, skillSetId in ipairs(self.skillSetOrderList) do
		local skillSet = self.skillSets[skillSetId]
		local child = { elem = "SkillSet", attrib = { id = tostring(skillSetId), title = skillSet.title } }
		t_insert(xml, child)

		for _, socketGroup in ipairs(skillSet.socketGroupList) do
			local node = { elem = "Skill", attrib = {
				enabled = tostring(socketGroup.enabled),
				includeInFullDPS = tostring(socketGroup.includeInFullDPS),
				groupCount = socketGroup.groupCount ~= nil and tostring(socketGroup.groupCount),
				label = socketGroup.label,
				slot = socketGroup.slot,
				source = socketGroup.source,
				mainActiveSkill = tostring(socketGroup.mainActiveSkill),
				mainActiveSkillCalcs = tostring(socketGroup.mainActiveSkillCalcs),
				imbuedSupport = socketGroup.imbuedSupport and tostring(socketGroup.imbuedSupport),
			} }
			for _, gemInstance in ipairs(socketGroup.gemList) do
				t_insert(node, { elem = "Gem", attrib = {
					nameSpec = gemInstance.nameSpec,
					skillId = gemInstance.skillId,
					gemId = gemInstance.gemData and gemInstance.gemData.gameId,
					variantId = gemInstance.gemData and gemInstance.gemData.variantId,
					level = tostring(gemInstance.level),
					quality = tostring(gemInstance.quality),
					enabled = tostring(gemInstance.enabled),
					enableGlobal1 = tostring(gemInstance.enableGlobal1),
					enableGlobal2 = tostring(gemInstance.enableGlobal2),
					count = tostring(gemInstance.count),
					skillPart = gemInstance.skillPart and tostring(gemInstance.skillPart),
					skillPartCalcs = gemInstance.skillPartCalcs and tostring(gemInstance.skillPartCalcs),
					skillStageCount = gemInstance.skillStageCount and tostring(gemInstance.skillStageCount),
					skillStageCountCalcs = gemInstance.skillStageCountCalcs and tostring(gemInstance.skillStageCountCalcs),
					skillMineCount = gemInstance.skillMineCount and tostring(gemInstance.skillMineCount),
					skillMineCountCalcs = gemInstance.skillMineCountCalcs and tostring(gemInstance.skillMineCountCalcs),
					skillMinion = gemInstance.skillMinion,
					skillMinionCalcs = gemInstance.skillMinionCalcs,
					skillMinionItemSet = gemInstance.skillMinionItemSet and tostring(gemInstance.skillMinionItemSet),
					skillMinionItemSetCalcs = gemInstance.skillMinionItemSetCalcs and tostring(gemInstance.skillMinionItemSetCalcs),
					skillMinionSkill = gemInstance.skillMinionSkill and tostring(gemInstance.skillMinionSkill),
					skillMinionSkillCalcs = gemInstance.skillMinionSkillCalcs and tostring(gemInstance.skillMinionSkillCalcs),
				} })
			end
			t_insert(child, node)
		end
	end
end

function SkillsTabClass:CopySocketGroup(socketGroup)
	local skillText = ""
	if socketGroup.label and socketGroup.label:match("%S") then
		skillText = skillText .. "Label: " .. socketGroup.label .. "\r\n"
	end
	if socketGroup.slot then
		skillText = skillText .. "Slot: " .. socketGroup.slot .. "\r\n"
	end
	for _, gemInstance in ipairs(socketGroup.gemList) do
		skillText = skillText .. string.format("%s %d/%d %s %d\r\n", gemInstance.nameSpec, gemInstance.level, gemInstance.quality, gemInstance.enabled and "" or "DISABLED", gemInstance.count or 1)
	end
	Copy(skillText)
	return skillText
end

function SkillsTabClass:PasteSocketGroup(testInput)
	local skillText = sanitiseText(Paste() or testInput)
	if skillText then
		local newGroup = { label = "", enabled = true, gemList = { } }
		local label = skillText:match("Label: (%C+)")
		if label then
			newGroup.label = label
		end
		local slot = skillText:match("Slot: (%C+)")
		if slot then
			newGroup.slot = slot
		end
		for nameSpec, level, quality, state, count in skillText:gmatch("([ %a']+) (%d+)/(%d+) ?(%a*) (%d+)") do
			t_insert(newGroup.gemList, {
				nameSpec = nameSpec,
				level = tonumber(level) or 20,
				quality = tonumber(quality) or 0,
				enabled = state ~= "DISABLED",
				count = tonumber(count) or 1,
				enableGlobal1 = true,
				enableGlobal2 = true
			})
		end
		if #newGroup.gemList > 0 then
			t_insert(self.socketGroupList, newGroup)
			self:SetDisplayGroup(newGroup)
			self:AddUndoState()
			self.build.buildFlag = true
		end
	end
end

-- Find the skill gem matching the given specification
function SkillsTabClass:FindSkillGem(nameSpec)
	-- Search for gem name using increasingly broad search patterns
	local patternList = {
		"^ "..nameSpec:gsub("%a", function(a) return "["..a:upper()..a:lower().."]" end).."$", -- Exact match (case-insensitive)
		"^"..nameSpec:gsub("%a", " %0%%l+").."$", -- Simple abbreviation ("CtF" -> "Cold to Fire")
		"^ "..nameSpec:gsub(" ",""):gsub("%l", "%%l*%0").."%l+$", -- Abbreviated words ("CldFr" -> "Cold to Fire")
		"^"..nameSpec:gsub(" ",""):gsub("%a", ".*%0"), -- Global abbreviation ("CtoF" -> "Cold to Fire")
		"^"..nameSpec:gsub(" ",""):gsub("%a", function(a) return ".*".."["..a:upper()..a:lower().."]" end), -- Case insensitive global abbreviation ("ctof" -> "Cold to Fire")
	}
	for i, pattern in ipairs(patternList) do
		local foundGemData
		for gemId, gemData in pairs(self.build.data.gems) do
			if (" "..gemData.name):match(pattern) then
				if foundGemData then
					return "Ambiguous gem name '" .. nameSpec .. "': matches '" .. foundGemData.name .. "', '" .. gemData.name .. "'"
				end
				foundGemData = gemData
			end
		end
		if foundGemData then
			return nil, foundGemData
		end
	end
	return "Unrecognised gem name '" .. nameSpec .. "'"
end

function SkillsTabClass:ProcessGemLevel(gemData, imbued)
	local grantedEffect = gemData.grantedEffect
	local naturalMaxLevel = gemData.naturalMaxLevel
	if imbued or self.defaultGemLevel == "levelOne" then
		return 1
	elseif self.defaultGemLevel == "awakenedMaximum" then
		return naturalMaxLevel + 1
	elseif self.defaultGemLevel == "corruptedMaximum" then
		if grantedEffect.plusVersionOf then
			return naturalMaxLevel
		else
			return naturalMaxLevel + 1
		end
	elseif self.defaultGemLevel == "normalMaximum" then
		return naturalMaxLevel
	else -- self.defaultGemLevel == "characterLevel"
		local maxGemLevel = naturalMaxLevel
		if not grantedEffect.levels[maxGemLevel] then
			maxGemLevel = #grantedEffect.levels
		end
		local characterLevel = self.build and self.build.characterLevel or 1
		for gemLevel = maxGemLevel, 1, -1 do
			if grantedEffect.levels[gemLevel].levelRequirement <= characterLevel then
				return gemLevel
			end
		end
		return 1
	end
end

-- Processes the given socket group, filling in information that will be used for display or calculations
---@param socketGroup table
function SkillsTabClass:ProcessSocketGroup(socketGroup)
	-- Loop through the skill gem list
	local data = self.build.data
	for _, gemInstance in ipairs(socketGroup.gemList) do
		gemInstance.color = "^8"
		gemInstance.nameSpec = gemInstance.nameSpec or ""
		local prevDefaultLevel = gemInstance.gemData and gemInstance.gemData.naturalMaxLevel or (gemInstance.new and 20)
		gemInstance.gemData, gemInstance.grantedEffect = nil
		if gemInstance.gemId then
			-- Specified by gem ID
			-- Used for skills granted by skill gems
			gemInstance.errMsg = nil
			gemInstance.gemData = data.gems[gemInstance.gemId]
			if gemInstance.gemData then
				gemInstance.nameSpec = gemInstance.gemData.name
				gemInstance.skillId = gemInstance.gemData.grantedEffectId
			end
		elseif gemInstance.skillId then
			-- Specified by skill ID
			-- Used for skills granted by items
			gemInstance.errMsg = nil
			local gemId = data.gemForSkill[gemInstance.skillId]
			if gemId then
				gemInstance.gemData = data.gems[gemId]
			else
				gemInstance.grantedEffect = data.skills[gemInstance.skillId]
			end
			if gemInstance.triggered then
				if gemInstance.grantedEffect.levels[gemInstance.level] then
					gemInstance.grantedEffect.levels[gemInstance.level].cost = {}
				end
			end
		elseif gemInstance.nameSpec:match("%S") then
			-- Specified by gem/skill name, try to match it
			-- Used to migrate pre-1.4.20 builds
			gemInstance.errMsg, gemInstance.gemData = self:FindSkillGem(gemInstance.nameSpec)
			gemInstance.gemId = gemInstance.gemData and gemInstance.gemData.id
			gemInstance.skillId = gemInstance.gemData and gemInstance.gemData.grantedEffectId
			if gemInstance.gemData then
				gemInstance.nameSpec = gemInstance.gemData.name
			end
		else
			gemInstance.errMsg, gemInstance.gemData, gemInstance.skillId = nil
		end
		if gemInstance.gemData and gemInstance.gemData.grantedEffect.unsupported then
			gemInstance.errMsg = gemInstance.nameSpec .. " is not supported yet"
			gemInstance.gemData = nil
		end
		if gemInstance.gemData or gemInstance.grantedEffect then
			gemInstance.new = nil
			local grantedEffect = gemInstance.grantedEffect or gemInstance.gemData.grantedEffect
			if grantedEffect.color == 1 then
				gemInstance.color = colorCodes.STRENGTH
			elseif grantedEffect.color == 2 then
				gemInstance.color = colorCodes.DEXTERITY
			elseif grantedEffect.color == 3 then
				gemInstance.color = colorCodes.INTELLIGENCE
			else
				gemInstance.color = colorCodes.NORMAL
			end
			if prevDefaultLevel and gemInstance.gemData and gemInstance.gemData.naturalMaxLevel ~= prevDefaultLevel then
				gemInstance.level = gemInstance.gemData.naturalMaxLevel
				gemInstance.naturalMaxLevel = gemInstance.level
			end
			calcLib.validateGemLevel(gemInstance)
			if gemInstance.gemData then
				gemInstance.reqLevel = grantedEffect.levels[gemInstance.level].levelRequirement
				gemInstance.reqStr = calcLib.getGemStatRequirement(gemInstance.reqLevel, grantedEffect.support, gemInstance.gemData.reqStr)
				gemInstance.reqDex = calcLib.getGemStatRequirement(gemInstance.reqLevel, grantedEffect.support, gemInstance.gemData.reqDex)
				gemInstance.reqInt = calcLib.getGemStatRequirement(gemInstance.reqLevel, grantedEffect.support, gemInstance.gemData.reqInt)
			end
		end
	end
end

-- reprocess socket groups on rebuild
function SkillsTabClass:UpdateSocketGroups()
	local slotSocketedCounts = {}
	for _, socketGroup in ipairs(self.socketGroupList) do
		-- Clear stale matches when a group is no longer assigned to an item.
		for _, gemInstance in ipairs(socketGroup.gemList) do
			gemInstance.matchesSocket = false
		end
		if socketGroup.slot and socketGroup.source == nil then
			local gemOffset = (slotSocketedCounts[socketGroup.slot] or 0)
			for i, gemInstance in ipairs(socketGroup.gemList) do
				-- add quality for matching sockets by looking up linked item
				if (gemInstance.grantedEffect or gemInstance.gemData) then
					local grantedEffect = gemInstance.grantedEffect or gemInstance.gemData.grantedEffect
					local slot = self.build.itemsTab.slots[socketGroup.slot]
					-- since PoB processes split links on an item as separate
					-- groups, we can assume that we continue from where the last
					-- socket group with the slot ended at
					local colours = { "R", "G", "B" }
					local gemIdx = gemOffset + i
					if slot then
						local item = self.build.itemsTab.items[slot.selItemId]
						if item and item.sockets then
							-- e.g. dialla's malefaction
							if item.sockets.colourAlwaysMatches then
								gemInstance.matchesSocket = true
							else
								local gemColour = grantedEffect.color and colours[grantedEffect.color]
								gemInstance.matchesSocket = item.sockets[gemIdx] and (item.sockets[gemIdx].color == gemColour)
							end
						end
					end
				end
			end
			slotSocketedCounts[socketGroup.slot] = gemOffset + #socketGroup.gemList
		end
	end
end

-- Set the socket group in the process of being edited
function SkillsTabClass:SetDisplayGroup(socketGroup)
	self.displayGroup = socketGroup
	if socketGroup then
		self:ProcessSocketGroup(socketGroup)
	end
end

function SkillsTabClass:CreateUndoState()
	local state = { }
	state.activeSkillSetId = self.activeSkillSetId
	state.skillSets = { }
	for skillSetIndex, skillSet in pairs(self.skillSets) do
		local newSkillSet = copyTable(skillSet, true)
		newSkillSet.socketGroupList = { }
		for socketGroupIndex, socketGroup in pairs(skillSet.socketGroupList) do
			local newGroup = copyTable(socketGroup, true)
			newGroup.gemList = { }
			for gemIndex, gem in pairs(socketGroup.gemList) do
				newGroup.gemList[gemIndex] = copyTable(gem, true)
			end
			newSkillSet.socketGroupList[socketGroupIndex] = newGroup
		end
		state.skillSets[skillSetIndex] = newSkillSet
	end
	state.skillSetOrderList = copyTable(self.skillSetOrderList)
	-- Save active socket group for both skillsTab and calcsTab to UndoState
	state.activeSocketGroup = self.build.mainSocketGroup
	state.activeSocketGroup2 = self.build.calcsTab.input.skill_number
	return state
end

function SkillsTabClass:RestoreUndoState(state)
	local displayId = isValueInArray(self.socketGroupList, self.displayGroup)
	wipeTable(self.skillSets)
	for k, v in pairs(state.skillSets) do
		self.skillSets[k] = v
	end
	wipeTable(self.skillSetOrderList)
	for k, v in ipairs(state.skillSetOrderList) do
		self.skillSetOrderList[k] = v
	end
	self:SetActiveSkillSet(state.activeSkillSetId)
	self:SetDisplayGroup(displayId and self.socketGroupList[displayId])
	-- Load active socket group for both skillsTab and calcsTab from UndoState
	self.build.mainSocketGroup = state.activeSocketGroup
	self.build.calcsTab.input.skill_number = state.activeSocketGroup2
end

-- Creates a new skill set
function SkillsTabClass:NewSkillSet(skillSetId)
	local skillSet = { id = skillSetId, socketGroupList = {} }
	if not skillSetId then
		skillSet.id = 1
		while self.skillSets[skillSet.id] do
			skillSet.id = skillSet.id + 1
		end
	end
	self.skillSets[skillSet.id] = skillSet
	return skillSet
end

function SkillsTabClass:RebuildImbuedSupportBySlot()
	wipeTable(self.imbuedSupportBySlot)
	for _, socketGroup in ipairs(self.socketGroupList) do
		if socketGroup.slot and socketGroup.imbuedSupport then
			local gemId = data.gemForBaseName[socketGroup.imbuedSupport:lower().." support"]
			local gem = gemId and data.gems[gemId]
			if gem and gem.grantedEffect then
				self.imbuedSupportBySlot[socketGroup.slot] = gem.grantedEffect
			end
		end
	end
end

-- Changes the active skill set
function SkillsTabClass:SetActiveSkillSet(skillSetId)
	-- Initialize skill sets if needed
	if not self.skillSetOrderList[1] then
		self.skillSetOrderList[1] = 1
		self:NewSkillSet(1)
	end

	if not skillSetId then
		skillSetId = self.activeSkillSetId
	end

	if not self.skillSets[skillSetId] then
		skillSetId = self.skillSetOrderList[1]
	end

	self.socketGroupList = self.skillSets[skillSetId].socketGroupList
	self:RebuildImbuedSupportBySlot()
	self.activeSkillSetId = skillSetId
	self.build.buildFlag = true

	self:SetDisplayGroup(self.socketGroupList[1])
	self.build:SyncLoadouts()
end
