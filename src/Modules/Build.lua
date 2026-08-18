-- Path of Building
--
-- Module: Build
-- Loads and manages the current build.
--
local pairs = pairs
local ipairs = ipairs
local next = next
local t_insert = table.insert
local m_min = math.min
local m_max = math.max
local m_huge = math.huge
local m_abs = math.abs
local s_format = string.format

local buildMode = { }

local function InsertIfNew(t, val)
	if (not t) then return end
	for i,v in ipairs(t) do
		if v == val then return end
	end
	table.insert(t, val)
end

---matchFlags
---  Compares the skill flags table against the line flag settings
---  Required enabling flags check takes precedence over disabling flags check
---@param reqFlags table containing the required flags
---@param notFlags table containing the disabling flags
---@param flags table containing the flags to match against
local function matchFlags(reqFlags, notFlags, flags)
	if type(reqFlags) == "string" then
		reqFlags = { reqFlags }
	end
	if reqFlags then
		for _, flag in ipairs(reqFlags) do
			if not flags[flag] then
				return
			end
		end
	end

	if type(notFlags) == "string" then
		notFlags = { notFlags }
	end
	if notFlags then
		for _, flag in ipairs(notFlags) do
			if flags[flag] then
				return
			end
		end
	end
	-- Both flag checks passed, default true
	return true
end

function buildMode:Init(dbFileName, buildName, buildXML, convertBuild, importLink)
	self.dbFileName = dbFileName
	self.buildName = buildName
	self.importLink = importLink
	if dbFileName then
		self.dbFileSubPath = self.dbFileName:sub(#main.buildPath + 1, -#self.buildName - 5)
	else
		self.dbFileSubPath = main.modes.LIST.subPath or ""
	end
	if not buildName then
		main:SetMode("LIST")
	end
	self.saveAsSortMode = "NAME"

	-- Load build file
	self.xmlSectionList = { }
	self.spectreList = { }
	self.timelessData = { jewelType = { }, conquerorType = { }, devotionVariant1 = 1, devotionVariant2 = 1, jewelSocket = { }, fallbackWeightMode = { }, searchList = "", searchListFallback = "", searchResults = { }, sharedResults = { } }
	self.viewMode = "TREE"
	self.characterLevel = m_min(m_max(main.defaultCharLevel or 1, 1), 100)
	self.targetVersion = liveTargetVersion
	self.bandit = "None"
	self.pantheonMajorGod = "None"
	self.pantheonMinorGod = "None"
	self.characterLevelAutoMode = main.defaultCharLevel == 1 or main.defaultCharLevel == nil
	if buildXML then
		if self:LoadDB(buildXML, "Unnamed build") then
			self:CloseBuild()
			return
		end
		self.modFlag = true
	else
		if self:LoadDBFile() then
			self:CloseBuild()
			return
		end
		self.modFlag = false
	end

	if convertBuild then
		self.targetVersion = liveTargetVersion
	end
	if self.targetVersion ~= liveTargetVersion then
		self.targetVersion = nil
		ConPrintf("Build '%s' was created for an unsupported game version and must be converted before it can be used", self.buildName or "?")
		return
	end

	self.abortSave = true

	self.warningLines = { }
	self.statBoxList = { }

	if buildName == "~~temp~~" then
		-- Remove temporary build file
		os.remove(self.dbFileName)
		self.buildName = "Unnamed build"
		self.dbFileName = false
		self.dbFileSubPath = nil
		self.modFlag = true
	end

	-- List of display stats
	self.displayStats, self.minionDisplayStats, self.extraSaveStats = LoadModule("Modules/BuildDisplayStats")

	-- Initialise build components
	self.latestTree = main.tree[latestTreeVersion]
	data.setJewelRadiiGlobally(latestTreeVersion)
	self.data = data
	self.importTab = new("ImportTab", self)
	self.notesTab = new("NotesTab", self)
	self.partyTab = new("PartyTab", self)
	self.configTab = new("ConfigTab", self)
	self.itemsTab = new("ItemsTab", self)
	self.treeTab = new("TreeTab", self)
	self.skillsTab = new("SkillsTab", self)
	self.calcsTab = new("CalcsTab", self)

	-- Load sections from the build file
	self.savers = {
		["Config"] = self.configTab,
		["Notes"] = self.notesTab,
		["Party"] = self.partyTab,
		["Tree"] = self.treeTab,
		["TreeView"] = self.treeTab.viewer,
		["Items"] = self.itemsTab,
		["Skills"] = self.skillsTab,
		["Calcs"] = self.calcsTab,
		["Import"] = self.importTab,
	}
	self.legacyLoaders = { -- Special loaders for legacy sections
		["Spec"] = self.treeTab,
	}

	--special rebuild to properly initialise boss placeholders
	self.configTab:BuildModList()

	-- Load legacy bandit and pantheon choices from build section
	for _, control in ipairs({ "bandit", "pantheonMajorGod", "pantheonMinorGod" }) do
		self.configTab.input[control] = self[control]
	end

	-- so we ran into problems with converted trees, trying to check passive tree routes and also consider thread jewels
	-- but we can't check jewel info because items have not been loaded yet, and they come after passives in the xml.
	-- the simplest solution seems to be making sure passive trees (which contain jewel sockets) are loaded last.
	local deferredPassiveTrees = { }
	for _, node in ipairs(self.xmlSectionList) do
		-- Check if there is a saver that can load this section
		local saver = self.savers[node.elem] or self.legacyLoaders[node.elem]
		if saver then
			-- if the saver is treeTab, defer it until everything is loaded
			if saver == self.treeTab  then
				t_insert(deferredPassiveTrees, node)
			else
				if saver:Load(node, self.dbFileName) then
					self:CloseBuild()
					return
				end
			end
		end
	end
	for _, node in ipairs(deferredPassiveTrees) do
		-- Check if there is a saver that can load this section
		if self.treeTab:Load(node, self.dbFileName) then
			self:CloseBuild()
			return
		end
	end
	for _, saver in pairs(self.savers) do
		if saver.PostLoad then
			saver:PostLoad()
		end
	end

	if next(self.configTab.input) == nil then
		-- Check for old calcs tab settings
		self.configTab:ImportCalcSettings()
	end

	-- reprocess socket groups as they might depend on items which don't necessarily load first.
	self.skillsTab:UpdateSocketGroups()
	-- Build calculation output tables
	wipeGlobalCache()
	self.outputRevision = 1
	self.calcsTab:BuildOutput()
	self:RefreshStatList()
	self.buildFlag = false

	self.abortSave = false
	self:SyncLoadouts()
end

local acts = {
	-- https://www.poewiki.net/wiki/Passive_skill
	[1] = { level = 1, questPoints = 0 },
	-- Act 1   : The Dweller of the Deep
	-- Act 1   : The Marooned Mariner
	[2] = { level = 12, questPoints = 2 },
	-- Act 1,2 : The Way Forward (Reward after reaching Act 2)
	-- Act 2   : Through Sacred Ground (Fellshrine Reward 3.25)
	[3] = { level = 22, questPoints = 4 },
	-- Act 3   : Victario's Secrets
	-- Act 3   : Piety's Pets
	[4] = { level = 32, questPoints = 6 },
	-- Act 4   : An Indomitable Spirit
	[5] = { level = 40, questPoints = 7 },
	-- Act 5   : In Service to Science
	-- Act 5   : Kitava's Torments
	[6] = { level = 44, questPoints = 9 },
	-- Act 6   : The Father of War
	-- Act 6   : The Puppet Mistress
	-- Act 6   : The Cloven One
	[7] = { level = 50, questPoints = 12 },
	-- Act 7   : The Master of a Million Faces
	-- Act 7   : Queen of Despair
	-- Act 7   : Kishara's Star
	[8] = { level = 54, questPoints = 15 },
	-- Act 8   : Love is Dead
	-- Act 8   : Reflection of Terror
	-- Act 8   : The Gemling Legion
	[9] = { level = 60, questPoints = 18 },
	-- Act 9   : Queen of the Sands
	-- Act 9   : The Ruler of Highgate
	[10] = { level = 64, questPoints = 20 },
	-- Act 10  : Vilenta's Vengeance
	-- Act 10  : An End to Hunger (+2)
	[11] = { level = 67, questPoints = 23 },
}

local function actExtra(act, extra)
	-- Act 2 : Deal With The Bandits (+1 if the player kills all bandits)
	return act > 2 and extra or 0
end

-- Build the loadout link tables from the tree/item/skill/config set names
function buildMode:SyncLoadouts()
	local filteredList = { }
	local treeList = {}
	local itemList = {}
	local skillList = {}
	local configList = {}
	-- used when selecting a loadout to set the correct setId for each SetActiveSet()
	self.treeListSpecialLinks, self.itemListSpecialLinks, self.skillListSpecialLinks, self.configListSpecialLinks = {}, {}, {}, {}

	local oneSkill = self.skillsTab and #self.skillsTab.skillSetOrderList == 1
	local oneItem = self.itemsTab and #self.itemsTab.itemSetOrderList == 1
	local oneConfig = self.configTab and #self.configTab.configSetOrderList == 1

	if self.treeTab ~= nil and self.itemsTab ~= nil and self.skillsTab ~= nil and self.configTab ~= nil then
		local transferTable = {}
		local sortedTreeListSpecialLinks = {}
		for id, spec in ipairs(self.treeTab.specList) do
			local specTitle = spec.title or "Default"
			-- only alphanumeric and comma are allowed in the braces { }
			local linkIdentifier = string.match(specTitle, "%{([%w,]+)%}")

			if linkIdentifier then
				local setName = specTitle:gsub("%{" .. linkIdentifier .. "%}", ""):gsub("^%s*", ""):gsub("%s*$", "")
				if not setName or setName == "" then
					setName = "Default"
				end

				-- iterate over each identifier, delimited by comma, and set the index so we can grab it later
				-- setId index is the id of the set in the global list needed for SetActiveSet
				-- setName is only used for Tree currently and we strip the braces to get the plain name of the set, this is used as the name of the loadout
				for linkId in string.gmatch(linkIdentifier, "[^%,]+") do
					transferTable["setId"] = id
					transferTable["setName"] = setName
					transferTable["linkId"] = linkId
					self.treeListSpecialLinks[linkId] = transferTable
					t_insert(sortedTreeListSpecialLinks, transferTable)
					transferTable = {}
				end
			else
				t_insert(treeList, (spec.treeVersion ~= latestTreeVersion and ("["..treeVersions[spec.treeVersion].display.."] ") or "")..(specTitle))
			end
		end

		-- item, skill, and config sets have identical structure
		local function identifyLinks(setOrderList, tabSets, setList, specialLinks, treeLinks)
			for id, set in ipairs(setOrderList) do
				local setTitle = tabSets[set].title or "Default"
				local linkIdentifier = string.match(setTitle, "%{([%w,]+)%}")

				-- this if/else prioritizes group identifier in case the user creates sets with same name AND same identifiers
				-- result is only the group is recognized and one loadout is created rather than a duplicate from each condition met
				if linkIdentifier then
					local setName = setTitle:gsub("%{" .. linkIdentifier .. "%}", ""):gsub("^%s*", ""):gsub("%s*$", "")
					if not setName or setName == "" then
						setName = "Default"
					end

					for linkId in string.gmatch(linkIdentifier, "[^%,]+") do
						transferTable["setId"] = set
						transferTable["setName"] = setName
						specialLinks[linkId] = transferTable
						transferTable = {}
					end
				else
					setList[setTitle] = true
				end
			end
		end
		identifyLinks(self.itemsTab.itemSetOrderList, self.itemsTab.itemSets, itemList, self.itemListSpecialLinks, self.treeListSpecialLinks)
		identifyLinks(self.skillsTab.skillSetOrderList, self.skillsTab.skillSets, skillList, self.skillListSpecialLinks, self.treeListSpecialLinks)
		identifyLinks(self.configTab.configSetOrderList, self.configTab.configSets, configList, self.configListSpecialLinks, self.treeListSpecialLinks)

		-- loop over all for exact match loadouts
		for id, tree in ipairs(treeList) do
			if (oneItem or itemList[tree]) and (oneSkill or skillList[tree]) and (oneConfig or configList[tree]) then
				t_insert(filteredList, tree)
			end
		end
		-- loop over the identifiers found within braces and set the loadout name to the TreeSet
		for _, tree in ipairs(sortedTreeListSpecialLinks) do
			local treeLinkId = tree.linkId
			if ((oneItem or self.itemListSpecialLinks[treeLinkId]) and (oneSkill or self.skillListSpecialLinks[treeLinkId]) and (oneConfig or self.configListSpecialLinks[treeLinkId])) then
				t_insert(filteredList, tree.setName .." {"..treeLinkId.."}")
			end
		end
	end

	self.loadoutList = filteredList
	return treeList, itemList, skillList, configList
end

-- Activate the tree/item/skill/config sets belonging to the named loadout
function buildMode:SelectLoadout(value)
	-- item, skill, and config sets have identical structure
	-- return id as soon as it's found
	local function findSetId(setOrderList, value, sets, setSpecialLinks)
		for _, setOrder in ipairs(setOrderList) do
			if value == (sets[setOrder].title or "Default") then
				return setOrder
			else
				local linkMatch = string.match(value, "%{(%w+)%}")
				if linkMatch then
					return setSpecialLinks[linkMatch]["setId"]
				end
			end
		end
		return nil
	end

	-- trees have a different structure with id/name pairs
	-- return id as soon as it's found
	local function findNamedSetId(treeList, value, setSpecialLinks)
		for id, spec in ipairs(treeList) do
			if value == spec then
				return id
			else
				local linkMatch = string.match(value, "%{(%w+)%}")
				if linkMatch then
					return setSpecialLinks[linkMatch]["setId"]
				end
			end
		end
		return nil
	end

	local oneSkill = self.skillsTab and #self.skillsTab.skillSetOrderList == 1
	local oneItem = self.itemsTab and #self.itemsTab.itemSetOrderList == 1
	local oneConfig = self.configTab and #self.configTab.configSetOrderList == 1

	local newSpecId = findNamedSetId(self.treeTab:GetSpecList(), value, self.treeListSpecialLinks)
	local newItemId = oneItem and 1 or findSetId(self.itemsTab.itemSetOrderList, value, self.itemsTab.itemSets, self.itemListSpecialLinks)
	local newSkillId = oneSkill and 1 or findSetId(self.skillsTab.skillSetOrderList, value, self.skillsTab.skillSets, self.skillListSpecialLinks)
	local newConfigId = oneConfig and 1 or findSetId(self.configTab.configSetOrderList, value, self.configTab.configSets, self.configListSpecialLinks)

	-- if exact match nor special grouping cannot find setIds, bail
	if newSpecId == nil or newItemId == nil or newSkillId == nil or newConfigId == nil then
		return
	end

	if newSpecId ~= self.treeTab.activeSpec then
		self.treeTab:SetActiveSpec(newSpecId)
	end
	if newItemId ~= self.itemsTab.activeItemSetId then
		self.itemsTab:SetActiveItemSet(newItemId)
	end
	if newSkillId ~= self.skillsTab.activeSkillSetId then
		self.skillsTab:SetActiveSkillSet(newSkillId)
	end
	if newConfigId ~= self.configTab.activeConfigSetId then
		self.configTab:SetActiveConfigSet(newConfigId)
	end
	return true
end

function buildMode:EstimatePlayerProgress()
	if self.spec then
		local PointsUsed, AscUsed, SecondaryAscUsed = self.spec:CountAllocNodes()
		local extra = self.calcsTab.mainOutput and self.calcsTab.mainOutput.ExtraPoints or 0
		local usedMax, ascMax, secondaryAscMax, level, act = 99 + 23 + extra, 8, 8, 1, 0

		-- Find estimated act and level based on points used
		repeat
			act = act + 1
			level = m_min(m_max(PointsUsed + 1 - acts[act].questPoints - actExtra(act, extra), acts[act].level), 100)
		until act == 11 or level <= acts[act + 1].level

		if self.characterLevelAutoMode and self.characterLevel ~= level then
			self.characterLevel = level
			self.configTab:BuildModList()
		end

		if PointsUsed > usedMax then InsertIfNew(self.warningLines, "You have too many passive points allocated") end
		if AscUsed > ascMax then InsertIfNew(self.warningLines, "You have too many ascendancy points allocated") end
		if SecondaryAscUsed > secondaryAscMax then InsertIfNew(self.warningLines, "You have too many secondary ascendancy points allocated") end
		self.Act = level < 90 and act <= 10 and act or "Endgame"
		self.pointsUsed, self.pointsMax = PointsUsed, usedMax
		self.ascPointsUsed, self.ascPointsMax = AscUsed, ascMax
		self.requiredLevel = level
	end
end

function buildMode:CanExit(mode)
	return true
end

function buildMode:Shutdown()
	if launch.devMode and (not main.disableDevAutoSave) and self.targetVersion and not self.abortSave then
		if self.dbFileName then
			self:SaveDBFile()
		elseif self.unsaved then
			self.dbFileName = main.buildPath.."~~temp~~.xml"
			self.buildName = "~~temp~~"
			self.dbFileSubPath = ""
			self:SaveDBFile()
		end
	end
	self.abortSave = nil

	self.savers = nil
end

function buildMode:GetArgs()
	return self.dbFileName, self.buildName
end

function buildMode:CloseBuild()
	main:SetMode("LIST", self.dbFileName and self.buildName, self.dbFileSubPath)
end

-- Re-initialise the build, converting it to the latest game version
function buildMode:ConvertToLatestVersion()
	self:Shutdown()
	self:Init(self.dbFileName, self.buildName, nil, true)
end

function buildMode:Load(xml, fileName)
	self.targetVersion = xml.attrib.targetVersion or legacyTargetVersion
	if xml.attrib.viewMode then
		self.viewMode = xml.attrib.viewMode
	end
	self.characterLevel = tonumber(xml.attrib.level) or 1
	self.characterLevelAutoMode = xml.attrib.characterLevelAutoMode == "true"
	for _, diff in pairs({ "bandit", "pantheonMajorGod", "pantheonMinorGod" }) do
		self[diff] = xml.attrib[diff] or "None"
	end
	self.mainSocketGroup = tonumber(xml.attrib.mainSkillIndex) or tonumber(xml.attrib.mainSocketGroup) or 1
	wipeTable(self.spectreList)
	for _, child in ipairs(xml) do
		if child.elem == "Spectre" then
			if child.attrib.id and data.minions[child.attrib.id] then
				t_insert(self.spectreList, child.attrib.id)
			end
		elseif child.elem == "TimelessData" then
			self.timelessData.jewelType = {
				id = tonumber(child.attrib.jewelTypeId)
			}
			self.timelessData.conquerorType = {
				id = tonumber(child.attrib.conquerorTypeId)
			}
			self.timelessData.devotionVariant1 = tonumber(child.attrib.devotionVariant1) or 1
			self.timelessData.devotionVariant2 = tonumber(child.attrib.devotionVariant2) or 1
			self.timelessData.jewelSocket = {
				id = tonumber(child.attrib.jewelSocketId)
			}
			self.timelessData.fallbackWeightMode = {
				idx = tonumber(child.attrib.fallbackWeightModeIdx)
			}
			self.timelessData.socketFilter = child.attrib.socketFilter == "true"
			self.timelessData.socketFilterDistance = tonumber(child.attrib.socketFilterDistance) or 0
			self.timelessData.searchList = child.attrib.searchList
			self.timelessData.searchListFallback = child.attrib.searchListFallback
		end
	end
end

function buildMode:Save(xml)
	xml.attrib = {
		targetVersion = self.targetVersion,
		viewMode = self.viewMode,
		level = tostring(self.characterLevel),
		className = self.spec.curClassName,
		ascendClassName = self.spec.curAscendClassName,
		bandit = self.configTab.input.bandit,
		pantheonMajorGod = self.configTab.input.pantheonMajorGod,
		pantheonMinorGod = self.configTab.input.pantheonMinorGod,
		mainSocketGroup = tostring(self.mainSocketGroup),
		characterLevelAutoMode = tostring(self.characterLevelAutoMode)
	}
	for _, id in ipairs(self.spectreList) do
		t_insert(xml, { elem = "Spectre", attrib = { id = id } })
	end
	local addedStatNames = { }
	for index, statData in ipairs(self.displayStats) do
		if matchFlags(statData.flag, statData.notFlag, self.calcsTab.mainEnv.player.mainSkill.skillFlags) then
			local statName = statData.stat and statData.stat..(statData.childStat or "")
			if statName and not addedStatNames[statName] then
				if statData.stat == "SkillDPS" then
					local statVal = self.calcsTab.mainOutput[statData.stat]
					for _, skillData in ipairs(statVal) do
						local triggerStr = ""
						if skillData.trigger and skillData.trigger ~= "" then
							triggerStr = skillData.trigger
						end
						local lhsString = skillData.name
						if skillData.count >= 2 then
							lhsString = tostring(skillData.count).."x "..skillData.name
						end
						t_insert(xml, { elem = "FullDPSSkill", attrib = { stat = lhsString, value = tostring(skillData.dps * skillData.count), skillPart = skillData.skillPart or "", source = skillData.source or skillData.trigger or "" } })
					end
					addedStatNames[statName] = true
				else
					local statVal = self.calcsTab.mainOutput[statData.stat]
					if statVal and statData.childStat then
						statVal = statVal[statData.childStat]
					end
					if statVal and (statData.condFunc and statData.condFunc(statVal, self.calcsTab.mainOutput) or true) then
						t_insert(xml, { elem = "PlayerStat", attrib = { stat = statName, value = tostring(statVal) } })
						addedStatNames[statName] = true
					end
				end
			end
		end
	end
	for index, stat in ipairs(self.extraSaveStats) do
		local statVal = self.calcsTab.mainOutput[stat]
		if statVal then
			t_insert(xml, { elem = "PlayerStat", attrib = { stat = stat, value = tostring(statVal) } })
		end
	end
	if self.calcsTab.mainEnv.minion then
		for index, statData in ipairs(self.minionDisplayStats) do
			if statData.stat then
				local statVal = self.calcsTab.mainOutput.Minion[statData.stat]
				if statVal then
					t_insert(xml, { elem = "MinionStat", attrib = { stat = statData.stat, value = tostring(statVal) } })
				end
			end
		end
	end
	local timelessData = {
		elem = "TimelessData",
		attrib = {
			jewelTypeId = next(self.timelessData.jewelType) and tostring(self.timelessData.jewelType.id),
			conquerorTypeId = next(self.timelessData.conquerorType) and tostring(self.timelessData.conquerorType.id),
			devotionVariant1 = tostring(self.timelessData.devotionVariant1),
			devotionVariant2 = tostring(self.timelessData.devotionVariant2),
			jewelSocketId = next(self.timelessData.jewelSocket) and tostring(self.timelessData.jewelSocket.id),
			fallbackWeightModeIdx = next(self.timelessData.fallbackWeightMode) and tostring(self.timelessData.fallbackWeightMode.idx),
			socketFilter = self.timelessData.socketFilter and "true",
			socketFilterDistance = self.timelessData.socketFilterDistance and tostring(self.timelessData.socketFilterDistance),
			searchList = self.timelessData.searchList and tostring(self.timelessData.searchList),
			searchListFallback = self.timelessData.searchListFallback and tostring(self.timelessData.searchListFallback)
		}
	}
	t_insert(xml, timelessData)
end

function buildMode:ResetModFlags()
	self.modFlag = false
	self.notesTab.modFlag = false
	self.partyTab.modFlag = false
	self.configTab.modFlag = false
	self.treeTab.modFlag = false
	self.treeTab.searchFlag = false
	self.spec.modFlag = false
	self.skillsTab.modFlag = false
	self.itemsTab.modFlag = false
	self.calcsTab.modFlag = false
end

function buildMode:OnFrame()
	-- Stop here if the loaded build needs to be converted
	if not self.targetVersion then
		return
	end

	if self.abortSave and not launch.devMode then
		self:CloseBuild()
	end

	if self.buildFlag then
		-- Wipe Global Cache
		wipeGlobalCache()

		-- Rebuild calculation output tables
		self.outputRevision = self.outputRevision + 1
		self.buildFlag = false
		self.skillsTab:UpdateSocketGroups()
		self.calcsTab:BuildOutput()
		self:RefreshStatList()
		self.configTab.calcFunc, self.configTab.calcBase = self.calcsTab:GetMiscCalculator(self)
	end

	self.unsaved = self.modFlag or self.notesTab.modFlag or self.partyTab.modFlag or self.configTab.modFlag or self.treeTab.modFlag or self.treeTab.searchFlag or self.spec.modFlag or self.skillsTab.modFlag or self.itemsTab.modFlag or self.calcsTab.modFlag
end

function buildMode:FormatStat(statData, statVal, overCapStatVal, colorOverride)
	if type(statVal) == "table" then return "" end
	local val = statVal * ((statData.pc or statData.mod) and 100 or 1) - (statData.mod and 100 or 0)
	local color = colorOverride or (statVal >= 0 and "^7" or statData.chaosInoc and "^8" or colorCodes.NEGATIVE)
	if statData.label == "Unreserved Life" and statVal == 0 then
		color = colorCodes.NEGATIVE
	end

	local valStr
	if statData.compactValue and main.useCompactValues and val ~= m_huge and val ~= -m_huge then
		local absVal = m_abs(val)
		if absVal >= 1000000000 then
			valStr = s_format("%.1fB", val / 1000000000)
		elseif absVal >= 1000000 then
			valStr = s_format("%.1fM", val / 1000000)
		elseif absVal >= 10000 then
			valStr = s_format("%.1fK", val / 1000)
		end
	end
	if not valStr then
		valStr = s_format("%"..statData.fmt, val)
		local number, suffix = valStr:match("^([%+%-]?%d+%.%d+)(%D*)$")
		if number then
			valStr = number:gsub("0+$", ""):gsub("%.$", "") .. suffix
		end
	end
	valStr = color .. formatNumSep(valStr)

	if overCapStatVal and overCapStatVal > 0 then
		valStr = valStr .. "^x808080" .. " (+" .. s_format("%d", overCapStatVal) .. "%)"
	end
	return valStr
end

-- Add stat list for given actor
function buildMode:AddDisplayStatList(statList, actor)
	local statBoxList = self.statBoxList
	for index, statData in ipairs(statList) do
		if matchFlags(statData.flag, statData.notFlag, actor.mainSkill.skillFlags) then
			local labelColor = "^7"
			if statData.color then
				labelColor = statData.color
			end
			if statData.stat then
				local statVal = actor.output[statData.stat]
				-- access output values that are one node deeper (statData.stat is a table e.g. output.MainHand.Accuracy vs output.Life)
				if statVal and statData.childStat then
					statVal = statVal[statData.childStat]
				end
				if statVal and ((statData.condFunc and statData.condFunc(statVal,actor.output)) or (not statData.condFunc and statVal ~= 0)) then
					local overCapStatVal = actor.output[statData.overCapStat] or nil
					if overCapStatVal and statData.overCapStatCondFunc and not statData.overCapStatCondFunc(statVal, actor.output) then
						overCapStatVal = nil
					end
					if statData.stat == "SkillDPS" then
						labelColor = colorCodes.CUSTOM
						table.sort(actor.output.SkillDPS, function(a,b) return (a.dps * a.count) > (b.dps * b.count) end)
						for _, skillData in ipairs(actor.output.SkillDPS) do
							local triggerStr = ""
							if skillData.trigger and skillData.trigger ~= "" then
								triggerStr = colorCodes.WARNING.." ("..skillData.trigger..")"..labelColor
							end
							local lhsString = labelColor..skillData.name..triggerStr..":"
							if skillData.count >= 2 then
								lhsString = labelColor..tostring(skillData.count).."x "..skillData.name..triggerStr..":"
							end
							t_insert(statBoxList, {
								height = 16,
								lhsString,
								self:FormatStat({ fmt = ".1f", compactValue = statData.compactValue }, skillData.dps * skillData.count, overCapStatVal),
							})
							if skillData.skillPart then
								t_insert(statBoxList, {
									height = 14,
									align = "CENTER_X", x = 140,
									"^8"..skillData.skillPart,
								})
							end
							if skillData.source then
								t_insert(statBoxList, {
									height = 14,
									align = "CENTER_X", x = 140,
									colorCodes.WARNING.."from " ..skillData.source,
								})
							end
						end
					elseif not (statData.hideStat) then
						-- Change the color of the stat label to red if cost exceeds pool
						local colorOverride = nil
						if actor.output[statData.stat.."Warning"] or (statData.warnFunc and statData.warnFunc(statVal, actor.output) and statData.warnColor) then
							colorOverride = colorCodes.NEGATIVE
						end
						local formattedStat = self:FormatStat(statData, statVal, overCapStatVal, colorOverride)
						if statData.suffix and (not statData.suffixCondFunc or statData.suffixCondFunc(statVal, actor.output)) then
							local suffix = statData.suffix
							if type(suffix) == "function" then
								suffix = suffix(statVal, actor.output)
							end
							if suffix then
								formattedStat = formattedStat .. "^x808080 (" .. suffix .. ")"
							end
						end
						t_insert(statBoxList, {
							height = 16,
							labelColor..statData.label..":",
							formattedStat,
						})
					end
				end
				if statData.warnFunc and statVal and ((statData.condFunc and statData.condFunc(statVal, actor.output)) or not statData.condFunc) then
					local v = statData.warnFunc(statVal, actor.output)
					if v then
						InsertIfNew(self.warningLines, v)
					end
				end
			elseif statData.label and statData.condFunc and statData.condFunc(actor.output) then
				t_insert(statBoxList, {
					height = 16, labelColor..statData.label..":",
					"^7"..actor.output[statData.labelStat].."%^x808080" .. " (" .. statData.val  .. ")",})
			elseif not statBoxList[#statBoxList] or statBoxList[#statBoxList][1] then
				t_insert(statBoxList, { height = 6 })
			end
		end
	end
	for pool, warningFlag in pairs({["Life"] = "LifeCostWarningList", ["Mana"] = "ManaCostWarningList", ["Rage"] = "RageCostWarningList", ["Energy Shield"] = "ESCostWarningList"}) do
		if actor.output[warningFlag] then
			local line = "You do not have enough "..(actor.output.EnergyShieldProtectsMana and pool == "Mana" and "Energy Shield and Mana" or pool).." to use: "
			for _, skill in ipairs(actor.output[warningFlag]) do
				line = line..skill..", "
			end
			line = line:sub(1, -3)
			InsertIfNew(self.warningLines, line)
		end
	end
	for pool, warningFlag in pairs({["Unreserved life"] = "LifePercentCostPercentCostWarningList", ["Unreserved Mana"] = "ManaPercentCostPercentCostWarningList"}) do
		if actor.output[warningFlag] then
			local line = "You do not have enough ".. pool .."% to use: "
			for _, skill in ipairs(actor.output[warningFlag]) do
				line = line..skill..", "
			end
			line = line:sub(1, -3)
			InsertIfNew(self.warningLines, line)
		end
	end
	if actor.output.VixensTooMuchCastSpeedWarn then
		InsertIfNew(self.warningLines, "You may have too much cast speed or too little cooldown reduction to effectively use Vixen's Curse replacement")
	end
	if actor.output.VixenModeNoVixenGlovesWarn then
		InsertIfNew(self.warningLines, "Vixen's calculation mode for Doom Blast is selected but you do not have Vixen's Entrapment Embroidered Gloves equipped")
	end

	do
		local aspectCount = 0
		aspectCount = aspectCount + (actor.output.CrabBarriersMax > 0 and actor.output.CrabBarriers > 0 and 1 or 0)
		aspectCount = aspectCount + (aspectCount < 2 and actor.modDB:Flag(nil, "Condition:AspectOfTheSpiderActive") and 1 or 0)
		aspectCount = aspectCount + (aspectCount < 2 and (actor.modDB:Flag(nil, "Condition:CatsAgilityActive") or actor.modDB:Flag(nil, "Condition:CatsStealthActive")) and 1 or 0)
		aspectCount = aspectCount + (aspectCount < 2 and (actor.modDB:Flag(nil, "Condition:AviansFlightActive") or actor.modDB:Flag(nil, "Condition:AviansMightActive")) and 1 or 0)
		if aspectCount > 1 then
			InsertIfNew(self.warningLines, "You have more than one Aspect skill active")
		end
	end
end

function buildMode:InsertItemWarnings()
	if self.calcsTab.mainEnv.itemWarnings.jewelLimitWarning then
		for _, warning in ipairs(self.calcsTab.mainEnv.itemWarnings.jewelLimitWarning) do
			InsertIfNew(self.warningLines, "You are exceeding jewel limit with the jewel "..warning)
		end
	end
	if self.calcsTab.mainEnv.itemWarnings.socketLimitWarning then
		for _, warning in ipairs(self.calcsTab.mainEnv.itemWarnings.socketLimitWarning) do
			InsertIfNew(self.warningLines, "You have too many gems in your "..warning.." slot")
		end
	end
	if self.calcsTab.mainEnv.itemWarnings.missingAnointWarning then
		InsertIfNew(self.warningLines, "You have eligible items missing an anoint: "..table.concat(self.calcsTab.mainEnv.itemWarnings.missingAnointWarning, ", "))
	end
end

-- Build list of side bar stats
function buildMode:RefreshStatList()
	self.warningLines = {}
	local statBoxList = wipeTable(self.statBoxList)
	if self.calcsTab.mainEnv.player.mainSkill.infoMessage then
			if #self.calcsTab.mainEnv.player.mainSkill.infoMessage > 40 then
				for line in string.gmatch(self.calcsTab.mainEnv.player.mainSkill.infoMessage, "([^:]+)") do
					t_insert(statBoxList, { height = 14, align = "CENTER_X", x = 140, colorCodes.CUSTOM .. line})
				end
			else
				t_insert(statBoxList, { height = 14, align = "CENTER_X", x = 140, colorCodes.CUSTOM .. self.calcsTab.mainEnv.player.mainSkill.infoMessage})
			end
		if self.calcsTab.mainEnv.player.mainSkill.infoMessage2 then
			t_insert(statBoxList, { height = 14, align = "CENTER_X", x = 140, "^8" .. self.calcsTab.mainEnv.player.mainSkill.infoMessage2})
		end
	end
	if self.calcsTab.mainEnv.minion then
		t_insert(statBoxList, { height = 18, "^7Minion:" })
		if self.calcsTab.mainEnv.minion.mainSkill.infoMessage then
			-- Split the line if too long
			if #self.calcsTab.mainEnv.minion.mainSkill.infoMessage > 40 then
				for line in string.gmatch(self.calcsTab.mainEnv.minion.mainSkill.infoMessage, "([^:]+)") do
					t_insert(statBoxList, { height = 14, align = "CENTER_X", x = 140, colorCodes.CUSTOM .. line})
				end
			else
				t_insert(statBoxList, { height = 14, align = "CENTER_X", x = 140, colorCodes.CUSTOM .. self.calcsTab.mainEnv.minion.mainSkill.infoMessage})
			end
			if self.calcsTab.mainEnv.minion.mainSkill.infoMessage2 then
				t_insert(statBoxList, { height = 14, align = "CENTER_X", x = 140, "^8" .. self.calcsTab.mainEnv.minion.mainSkill.infoMessage2})
			end
		end
		self:AddDisplayStatList(self.minionDisplayStats, self.calcsTab.mainEnv.minion)
		t_insert(statBoxList, { height = 10 })
		t_insert(statBoxList, { height = 18, "^7Player:" })
	end
	if self.calcsTab.mainEnv.player.mainSkill.skillFlags.disable then
		t_insert(statBoxList, { height = 16, "^7Skill disabled:" })
		t_insert(statBoxList, { height = 14, align = "CENTER_X", x = 140, self.calcsTab.mainEnv.player.mainSkill.disableReason })
	end
	self:AddDisplayStatList(self.displayStats, self.calcsTab.mainEnv.player)
	self:InsertItemWarnings()
	self:EstimatePlayerProgress()
end


function buildMode:CompareStatList(tooltip, statList, actor, baseOutput, compareOutput, header, nodeCount)
	local count = 0
	for _, statData in ipairs(statList) do
		if statData.stat and matchFlags(statData.flag, statData.notFlag, actor.mainSkill.skillFlags) and not statData.childStat and statData.stat ~= "SkillDPS" then
			local statVal1 = compareOutput[statData.stat] or 0
			local statVal2 = baseOutput[statData.stat] or 0
			local diff = statVal1 - statVal2
			if statData.stat == "FullDPS" and not compareOutput[statData.stat] then
				diff = 0
			end
			if (diff > 0.001 or diff < -0.001) and (not statData.condFunc or statData.condFunc(statVal1,compareOutput) or statData.condFunc(statVal2,baseOutput)) then
				if count == 0 then
					tooltip:AddLine(14, header)
				end
				local color = ((statData.lowerIsBetter and diff < 0) or (not statData.lowerIsBetter and diff > 0)) and colorCodes.POSITIVE or colorCodes.NEGATIVE
				local val = diff * ((statData.pc or statData.mod) and 100 or 1)
				local valStr = s_format("%+"..statData.fmt, val) -- Can't use self:FormatStat, because it doesn't have %+. Adding that would have complicated a simple function
				local number, suffix = valStr:match("^([%+%-]?%d+%.%d+)(%D*)$")
				if number then
					valStr = number:gsub("0+$", ""):gsub("%.$", "") .. suffix
				end

				valStr = formatNumSep(valStr)

				local line = s_format("%s%s %s", color, valStr, statData.label)
				local pcPerPt = ""
				if statData.compPercent and statVal1 ~= 0 and statVal2 ~= 0 then
					local pc = statVal1 / statVal2 * 100 - 100
					line = line .. s_format(" (%+.1f%%)", pc)
					if nodeCount then
						pcPerPt = s_format(" (%+.1f%%)", pc / nodeCount)
					end
				end
				if nodeCount then
					line = line .. s_format(" ^8[%+"..statData.fmt.."%s per point]", diff * ((statData.pc or statData.mod) and 100 or 1) / nodeCount, pcPerPt)
				end
				tooltip:AddLine(14, line)
				count = count + 1
			end
		end
	end
	return count
end

-- Compare values of all display stats between the two output tables, and add any changed stats to the tooltip
-- Adds the provided header line before the first stat line, if any are added
-- Returns the number of stat lines added
function buildMode:AddStatComparesToTooltip(tooltip, baseOutput, compareOutput, header, nodeCount)
	local count = 0
	if self.calcsTab.mainEnv.player.mainSkill.minion and baseOutput.Minion and compareOutput.Minion then
		count = count + self:CompareStatList(tooltip, self.minionDisplayStats, self.calcsTab.mainEnv.minion, baseOutput.Minion, compareOutput.Minion, header.."\n^7Minion:", nodeCount)
		if count > 0 then
			header = "^7Player:"
		else
			header = header.."\n^7Player:"
		end
	end
	count = count + self:CompareStatList(tooltip, self.displayStats, self.calcsTab.mainEnv.player, baseOutput, compareOutput, header, nodeCount)
	return count
end

-- Add requirements to tooltip
do
	local req = { }
	function buildMode:AddRequirementsToTooltip(tooltip, level, str, dex, int, strBase, dexBase, intBase)
		if level and level > 0 then
			t_insert(req, s_format("^x7F7F7FLevel %s%d", main:StatColor(level, nil, self.characterLevel), level))
		end
		-- Convert normal attributes to Omni attributes
		if self.calcsTab.mainEnv.modDB:Flag(nil, "OmniscienceRequirements") then
			local omniSatisfy = self.calcsTab.mainEnv.modDB:Sum("INC", nil, "OmniAttributeRequirements")
			local highestAttribute = 0
			for i, stat in ipairs({str, dex, int}) do
				if((stat or 0) > highestAttribute) then
					highestAttribute = stat
				end
			end
			local omni = math.floor(highestAttribute * (100/omniSatisfy))
			if omni and (omni > 0 or omni > self.calcsTab.mainOutput.Omni) then
				t_insert(req, s_format("%s%d ^x7F7F7FOmni", main:StatColor(omni, 0, self.calcsTab.mainOutput.Omni), omni))
			end
		else 
			if str and (str > 14 or str > self.calcsTab.mainOutput.Str) then
				t_insert(req, s_format("%s%d ^x7F7F7FStr", main:StatColor(str, strBase, self.calcsTab.mainOutput.Str), str))
			end
			if dex and (dex > 14 or dex > self.calcsTab.mainOutput.Dex) then
				t_insert(req, s_format("%s%d ^x7F7F7FDex", main:StatColor(dex, dexBase, self.calcsTab.mainOutput.Dex), dex))
			end
			if int and (int > 14 or int > self.calcsTab.mainOutput.Int) then
				t_insert(req, s_format("%s%d ^x7F7F7FInt", main:StatColor(int, intBase, self.calcsTab.mainOutput.Int), int))
			end
		end
		if req[1] then
			local fontSizeBig = main.showFlavourText and 18 or 16
			tooltip:AddLine(fontSizeBig, "^x7F7F7FRequires "..table.concat(req, "^x7F7F7F, "), "FONTIN SC")
			tooltip:AddSeparator(10)
		end
		wipeTable(req)
	end
end
function buildMode:LoadDB(xmlText, fileName)
	-- Parse the XML
	local dbXML, errMsg = common.xml.ParseXML(xmlText)
	if errMsg and errMsg:match(".*file returns nil") then
		main:OpenCloudErrorPopup(fileName)
		return true
	elseif errMsg then
		launch:ShowErrMsg("^1"..errMsg)
		return true
	elseif dbXML[1].elem ~= "PathOfBuilding" then
		launch:ShowErrMsg("^1Error parsing '%s': 'PathOfBuilding' root element missing", fileName)
		return true
	end

	-- Load Build section first
	for _, node in ipairs(dbXML[1]) do
		if type(node) == "table" and node.elem == "Build" then
			self:Load(node, self.dbFileName)
			break
		end
	end

	-- Check if xml has an import link
	for _, node in ipairs(dbXML[1]) do
		if type(node) == "table" and node.elem == "Import" then
			if node.attrib.importLink and not self.importLink then
				self.importLink = node.attrib.importLink
			end
			break
		end
	end

	-- Store other sections for later processing
	for _, node in ipairs(dbXML[1]) do
		if type(node) == "table" then
			t_insert(self.xmlSectionList, node)
		end
	end
end

function buildMode:LoadDBFile()
	if not self.dbFileName then
		return
	end
	ConPrintf("Loading '%s'...", self.dbFileName)
	local file = io.open(self.dbFileName, "r")
	if not file then
		self.dbFileName = nil
		return true
	end
	local xmlText = file:read("*a")
	file:close()
	return self:LoadDB(xmlText, self.dbFileName)
end

function buildMode:SaveDB(fileName)
	local dbXML = { elem = "PathOfBuilding" }

	-- Save Build section first
	do
		local node = { elem = "Build" }
		self:Save(node)
		t_insert(dbXML, node)
	end

	-- Call on all savers to save their data in their respective sections
	for elem, saver in pairs(self.savers) do
		local node = { elem = elem }
		saver:Save(node)
		t_insert(dbXML, node)
	end

	-- Compose the XML
	local xmlText, errMsg = common.xml.ComposeXML(dbXML)
	if not xmlText then
		launch:ShowErrMsg("Error saving '%s': %s", fileName, errMsg)
	else
		return xmlText
	end
end


function buildMode:SaveDBFile()
	if not self.dbFileName then
		return true
	end
	local xmlText = self:SaveDB(self.dbFileName)
	if not xmlText then
		return true
	end
	local file = io.open(self.dbFileName, "w+")
	if not file then
		main:OpenMessagePopup("Error", "Couldn't save the build file:\n"..self.dbFileName.."\nMake sure the save folder exists and is writable.")
		return true
	end
	file:write(xmlText)
	file:close()
	self.actionOnSave = nil

	-- Reset all modFlags
	self:ResetModFlags()
end

return buildMode
