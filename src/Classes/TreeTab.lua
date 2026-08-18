-- Path of Building
--
-- Module: Tree Tab
-- Passive skill tree state for the current build.
--
local ipairs = ipairs
local pairs = pairs
local next = next
local t_insert = table.insert
local t_remove = table.remove
local t_sort = table.sort
local t_concat = table.concat
local m_max = math.max
local m_min = math.min
local m_floor = math.floor
local s_format = string.format

local TreeTabClass = newClass("TreeTab", function(self, build)
	self.build = build
	self.isComparing = false

	-- Passive tree view state, persisted in the build file's TreeView section
	self.viewer = {
		zoomLevel = 3,
		zoom = 1.2 ^ 3,
		zoomX = 0,
		zoomY = 0,
		searchStr = "",
		searchStrSaved = "",
		showStatDifferences = true,
	}
	function self.viewer:Load(xml, fileName)
		if xml.attrib.zoomLevel then
			self.zoomLevel = tonumber(xml.attrib.zoomLevel)
			self.zoom = 1.2 ^ self.zoomLevel
		end
		if xml.attrib.zoomX and xml.attrib.zoomY then
			self.zoomX = tonumber(xml.attrib.zoomX)
			self.zoomY = tonumber(xml.attrib.zoomY)
		end
		if xml.attrib.searchStr then
			self.searchStr = xml.attrib.searchStr
			self.searchStrSaved = xml.attrib.searchStr
		end
		if xml.attrib.showStatDifferences then
			self.showStatDifferences = xml.attrib.showStatDifferences == "true"
		end
	end
	function self.viewer:Save(xml)
		self.searchStrSaved = self.searchStr
		xml.attrib = {
			zoomLevel = tostring(self.zoomLevel),
			zoomX = tostring(self.zoomX),
			zoomY = tostring(self.zoomY),
			searchStr = self.searchStr,
			showStatDifferences = tostring(self.showStatDifferences),
		}
	end

	self.specList = { }
	self.specList[1] = new("PassiveSpec", build, latestTreeVersion)
	self:SetActiveSpec(1)
	self:SetCompareSpec(1)

	-- table holding all realm/league pairs. (allLeagues[realm] = [league.id,...])
	self.tradeLeaguesList = {}

	--Default index for Tattoos
	self.defaultTattoo = { }

	self.powerStatList = { }
	for _, stat in ipairs(data.powerStatList) do
		if not stat.ignoreForNodes then
			t_insert(self.powerStatList, stat)
		end
	end

	-- Completion callback from the CalcsTab power builder coroutine
	self.build.powerBuilderCallback = function()
		local powerStat = self.build.calcsTab.powerStat or data.powerStatList[1]
		self.powerReport = self:BuildPowerReportList(powerStat)
		self.powerReportStat = powerStat
	end
end)

function TreeTabClass:RemoveTattooFromNode(node)
	self.build.spec.tree.nodes[node.id].isTattoo = false
	self.build.spec.hashOverrides[node.id] = nil
	self.build.spec:ReplaceNode(node, self.build.spec.tree.nodes[node.id])
	self.build.spec:BuildAllDependsAndPaths()
end

-- Replace the given node with the given tattoo node
function TreeTabClass:ApplyTattooToNode(node, tattooId)
	local newTattooNode = self.build.spec.tree.tattoo.nodes[tattooId]
	if not newTattooNode then
		return
	end
	newTattooNode.id = node.id
	self.build.spec.hashOverrides[node.id] = newTattooNode
	self.build.spec:ReplaceNode(node, newTattooNode)
	self.build.spec:BuildAllDependsAndPaths()
end

-- Select a mastery effect for the given mastery node, allocating the node if necessary
function TreeTabClass:SelectMasteryEffect(node, effectId)
	local effect = self.build.spec.tree.masteryEffects[effectId]
	if not effect then
		return
	end
	node.sd = effect.sd
	node.allMasteryOptions = false
	node.reminderText = { "Tip: Right click to select a different effect" }
	self.build.spec.tree:ProcessStats(node)
	self.build.spec.masterySelections[node.id] = effect.id
	if not node.alloc then
		self.build.spec:AllocNode(node)
	end
	self.build.spec:AddUndoState()
	self.modFlag = true
	self.build.buildFlag = true
end

function TreeTabClass:GetSpecList()
	local newSpecList = { }
	for _, spec in ipairs(self.specList) do
		t_insert(newSpecList, (spec.treeVersion ~= latestTreeVersion and ("["..treeVersions[spec.treeVersion].display.."] ") or "")..(spec.title or "Default"))
	end
	return newSpecList
end

function TreeTabClass:Load(xml, dbFileName)
	self.specList = { }
	if xml.elem == "Spec" then
		-- Import single spec from old build
		self.specList[1] = new("PassiveSpec", self.build, defaultTreeVersion)
		self.specList[1]:Load(xml, dbFileName)
		self.activeSpec = 1
		self.build.spec = self.specList[1]
		return
	end
	for _, node in pairs(xml) do
		if type(node) == "table" then
			if node.elem == "Spec" then
				if node.attrib.treeVersion and not treeVersions[node.attrib.treeVersion] then
					main:OpenMessagePopup("Unknown Passive Tree Version", "The build you are trying to load uses an unrecognised version of the passive skill tree.\nYou may need to update the program before loading this build.")
					return true
				end
				local newSpec = new("PassiveSpec", self.build, node.attrib.treeVersion or defaultTreeVersion)
				newSpec:Load(node, dbFileName)
				t_insert(self.specList, newSpec)
			end
		end
	end
	if not self.specList[1] then
		self.specList[1] = new("PassiveSpec", self.build, latestTreeVersion)
	end
	self:SetActiveSpec(tonumber(xml.attrib.activeSpec) or 1)
end

function TreeTabClass:PostLoad()
	for _, spec in ipairs(self.specList) do
		spec:PostLoad()
	end
	self.build.itemsTab:PopulateSlots()
end

function TreeTabClass:Save(xml)
	xml.attrib = {
		activeSpec = tostring(self.activeSpec)
	}
	for specId, spec in ipairs(self.specList) do
		local child = {
			elem = "Spec"
		}
		spec:Save(child)
		t_insert(xml, child)
	end
end

function TreeTabClass:SetActiveSpec(specId)
	local prevSpec = self.build.spec
	self.activeSpec = m_min(specId, #self.specList)
	local curSpec = self.specList[self.activeSpec]
	data.setJewelRadiiGlobally(curSpec.treeVersion)
	self.build.spec = curSpec
	self.build.buildFlag = true
	for _, slot in pairs(self.build.itemsTab.slots) do
		if slot.nodeId then
			if prevSpec then
				-- Update the previous spec's jewel for this slot
				prevSpec.jewels[slot.nodeId] = slot.selItemId
			end
			if curSpec.jewels[slot.nodeId] then
				-- Socket the jewel for the new spec
				slot.selItemId = curSpec.jewels[slot.nodeId]
			else
				-- Unsocket the old jewel from the previous spec
				slot.selItemId = 0
			end
		end
	end
	self.showConvert = not curSpec.treeVersion:match("^" .. latestTreeVersion)
	if self.build.itemsTab.itemOrderList[1] then
		-- Update item slots if items have been loaded already
		self.build.itemsTab:PopulateSlots()
	end
	self.build:SyncLoadouts()
end

function TreeTabClass:SetCompareSpec(specId)
	self.activeCompareSpec = m_min(specId, #self.specList)
	local curSpec = self.specList[self.activeCompareSpec]

	self.compareSpec = curSpec
end

function TreeTabClass:ConvertToVersion(version, remove, success, ignoreTreeSubType)
	local treeSubTypeCapture = self.build.spec.treeVersion:match("(_%l+_?%l*)")
	if not ignoreTreeSubType and treeSubTypeCapture and not version:match(treeSubTypeCapture) then
		if isValueInTable(treeVersionList, version..treeSubTypeCapture) then
			version = version..treeSubTypeCapture
		end
	end
	local newSpec = new("PassiveSpec", self.build, version)
	newSpec.title = self.build.spec.title
	newSpec.jewels = copyTable(self.build.spec.jewels)
	newSpec:RestoreUndoState(self.build.spec:CreateUndoState(), version)
	newSpec:BuildClusterJewelGraphs()
	t_insert(self.specList, self.activeSpec + 1, newSpec)
	if remove then
		t_remove(self.specList, self.activeSpec)
		-- activeSpec + 1 is shifted down one on remove, otherwise we would set the spec below it if it exists
		self:SetActiveSpec(self.activeSpec)
	else
		self:SetActiveSpec(self.activeSpec + 1)
	end
	self.modFlag = true
	if success then
		main:OpenMessagePopup("Tree Converted", "The tree has been converted to "..treeVersions[version].display..".")
	end
end

function TreeTabClass:ConvertAllToVersion(version)
	local currActiveSpec = self.activeSpec
	local specVersionList = { }
	for _, spec in ipairs(self.specList) do
		t_insert(specVersionList, spec.treeVersion)
	end
	for index, specVersion in ipairs(specVersionList) do
		if specVersion ~= version then
			self:SetActiveSpec(index)
			self:ConvertToVersion(version, true, false)
		end
	end
	self:SetActiveSpec(currActiveSpec)
end

-- Import a passive tree from a tree link, adding it as a new spec
function TreeTabClass:ImportTree(treeLink, name)
	local versionLookup = "tree/([0-9]+)%.([0-9]+)%.([0-9]+)/"
	local function validateTreeVersion(alternateType, major, minor)
		-- Take the Major and Minor version numbers and confirm it is a valid tree version. The point release is also passed in but it is not used
		-- Return: the passed in tree version as text or latestTreeVersion
		if major and minor then
			--need leading 0 here
			local newTreeVersionNum = tonumber(string.format("%d.%02d", major, minor))
			if newTreeVersionNum >= treeVersions[defaultTreeVersion].num and newTreeVersionNum <= treeVersions[latestTreeVersion].num then
				-- no leading 0 here
				return string.format("%s_%s", major, minor) .. (alternateType and ("_" .. alternateType:gsub("-", "_")) or "")
			else
				ConPrintf("Version '%d_%02d' is out of bounds", major, minor)
			end
		end
		return latestTreeVersion .. (alternateType and ("_" .. alternateType:gsub("-", "_")) or "")
	end
	if treeLink:match("poeplanner.com/") then
		treeLink = treeLink:gsub("/%?v=.+#","/")
		-- treeVersion is not known at this point. We need to decode the URL to get it.
		local tmpSpec = new("PassiveSpec", self.build, latestTreeVersion)
		local newTreeVersion_or_errMsg = tmpSpec:DecodePoePlannerURL(treeLink, true)
		-- Check for an error message
		if string.find(newTreeVersion_or_errMsg, "Invalid") then
			return newTreeVersion_or_errMsg
		end
		local newSpec = new("PassiveSpec", self.build, newTreeVersion_or_errMsg)
		newSpec.title = name
		newSpec:DecodePoePlannerURL(treeLink, false)
		t_insert(self.specList, newSpec)
		self:SetActiveSpec(#self.specList)
		self.modFlag = true
		self.build.spec:AddUndoState()
		self.build.buildFlag = true
		return
	end
	if treeLink:match("poeskilltree.com/") then
		local oldStyleVersionLookup = "/%?v=([0-9]+)%.([0-9]+)%.([0-9]+)%-?%w?%-?%w?#"
		-- Strip the version from the tree : https://poeskilltree.com/?v=3.6.0#AAAABAMAABEtfIOFMo6-ksHfsOvu -> https://poeskilltree.com/AAAABAMAABEtfIOFMo6-ksHfsOvu
		local versionedTreeLink = treeLink
		treeLink = treeLink:gsub("/%?v=.+#","/")
		local newTreeVersion = validateTreeVersion(versionedTreeLink:match("%-(%l+%-?%l*)#"), versionedTreeLink:match(oldStyleVersionLookup))
		local newSpec = new("PassiveSpec", self.build, newTreeVersion)
		newSpec.title = name
		local errMsg = newSpec:DecodeURL(treeLink)
		if errMsg then
			return errMsg
		end
		t_insert(self.specList, newSpec)
		self:SetActiveSpec(#self.specList)
		self.modFlag = true
		self.build.spec:AddUndoState()
		self.build.buildFlag = true
		return
	end
	-- EG: https://www.pathofexile.com/passive-skill-tree/3.15.0/AAAABgMADI6-HwKSwQQHLJwtH9-wTLNfKoP3ES3r5AAA
	local newTreeVersion = validateTreeVersion(treeLink:match("tree/(%l+%-?%l*)"), treeLink:match(versionLookup))
	local newSpec = new("PassiveSpec", self.build, newTreeVersion)
	newSpec.title = name
	local errMsg = newSpec:DecodeURL(treeLink)
	if errMsg then
		return errMsg
	end
	t_insert(self.specList, newSpec)
	self:SetActiveSpec(#self.specList)
	self.modFlag = true
	self.build.spec:AddUndoState()
	self.build.buildFlag = true
end

-- Export the active passive tree as a tree link
function TreeTabClass:ExportTree()
	return self.build.spec:EncodeURL(treeVersions[self.build.spec.treeVersion].url)
end

function TreeTabClass:SetPowerCalc(powerStat)
	self.build.buildFlag = true
	self.build.calcsTab.powerBuildFlag = true
	self.build.calcsTab.powerStat = powerStat
	self.powerReport = nil
	self.powerReportStat = powerStat
end

function TreeTabClass:BuildPowerReportList(currentStat)
	local report = {}

	if not (currentStat and currentStat.stat) then
		return report
	end

	-- locate formatting information for the type of heat map being used.
	-- maybe a better place to find this? At the moment, it is the only place
	-- in the code that has this information in a tidy place.
	local displayStat = nil

	for index, ds in ipairs(self.build.displayStats) do
		if ds.stat == currentStat.stat then
			displayStat = ds
			break
		end
	end

	-- not every heat map has an associated "stat" in the displayStats table
	-- this is due to not every stat being displayed in the sidebar, I believe.
	-- But, we do want to use the formatting knowledge stored in that table rather than duplicating it here.
	-- If no corresponding stat is found, just default to a generic stat display (>0=good, one digit of precision).
	if not displayStat then
		displayStat = {
			fmt = ".1f"
		}
	end
	local powerMultiplier = (displayStat.pc or displayStat.mod) and 100 or 1
	local function formatPower(power)
		local powerStr = formatNumSep(s_format("%"..displayStat.fmt, power))
		if (power > 0 and not displayStat.lowerIsBetter) or (power < 0 and displayStat.lowerIsBetter) then
			return colorCodes.POSITIVE .. powerStr
		elseif (power < 0 and not displayStat.lowerIsBetter) or (power > 0 and displayStat.lowerIsBetter) then
			return colorCodes.NEGATIVE .. powerStr
		end
		return powerStr
	end
	local function getNodePathDist(node, isAlloc)
		if isAlloc then
			return #(node.depends or { }) == 0 and 1 or #node.depends
		end
		return node.power.distance or #(node.path or {}) == 0 and 1 or #node.path
	end
	local function addReportEntry(node, name, nodePower, pathPower, pathDist, isAlloc, pathPowerStr)
		t_insert(report, {
			name = name,
			power = nodePower,
			powerStr = formatPower(nodePower),
			pathPower = pathPower,
			pathPowerStr = pathPowerStr or formatPower(pathPower),
			allocated = isAlloc,
			id = node.id,
			x = node.x,
			y = node.y,
			type = node.type,
			sd = node.sd,
			pathDist = pathDist
		})
	end

	-- search all nodes, ignoring ascendancies, sockets, etc.
	for nodeId, node in pairs(self.build.spec.nodes) do
		local isAlloc = node.alloc or self.build.calcsTab.mainEnv.grantedPassives[nodeId]
		if (node.type == "Normal" or node.type == "Keystone" or node.type == "Notable") and not node.ascendancyName then
			local pathDist = getNodePathDist(node, isAlloc)
			local nodePower = (node.power.singleStat or 0) * powerMultiplier
			local pathPower = (node.power.pathPower or 0) / pathDist * powerMultiplier
			addReportEntry(node, node.dn, nodePower, pathPower, pathDist, isAlloc)
		elseif node.type == "Mastery" and node.power.masteryEffects and not node.ascendancyName then
			local pathDist = getNodePathDist(node, isAlloc)

			for _, masteryEffect in ipairs(node.masteryEffects or { }) do
				local effect = self.build.spec.tree.masteryEffects[masteryEffect.effect]
				local effectPower = node.power.masteryEffects[masteryEffect.effect]
				if effect and effectPower then
					local effectLabelParts = isAlloc and not node.allMasteryOptions and node.sd or effect.stats or effect.sd
					local name = effectLabelParts and node.dn..": "..t_concat(effectLabelParts, " / ") or node.dn
					local nodePower = (effectPower.singleStat or 0) * powerMultiplier
					local pathPower = ((effectPower.pathPower or effectPower.singleStat or 0) / pathDist) * powerMultiplier
					addReportEntry(node, name, nodePower, pathPower, pathDist, isAlloc)
				end
			end
		end
	end

	-- search all cluster notables and add to the list
	for nodeName, node in pairs(self.build.spec.tree.clusterNodeMap) do
		local isAlloc = node.alloc
		if not isAlloc then
			local nodePower = (node.power and node.power.singleStat or 0) * powerMultiplier
			addReportEntry(node, node.dn, nodePower, 0, "Cluster", isAlloc, "--")
		end
	end

	-- sort it
	if displayStat.lowerIsBetter then
		t_sort(report, function (a,b)
			return a.power < b.power
		end)
	else
		t_sort(report, function (a,b)
			return a.power > b.power
		end)
	end

	return report
end

-- Generates the jewel item for a timeless search result and adds it to the build
local function timelessJewelResultClick(build, timelessData, index, data, doubleClick)
	if doubleClick and timelessData.searchResults[index].label:match("B2B2B2") == nil then
		local socketInfo = data.socketLabel or (timelessData.sharedResults.socket and timelessData.sharedResults.socket.keystone) or "Unknown"
		local label = "[" .. data.seed .. "; " .. data.total.. "; " .. socketInfo .. "]\n"
		local variant = timelessData.sharedResults.conqueror.id == 1 and 1 or (timelessData.sharedResults.conqueror.id - 1) .. "\n"
		local itemData
		if timelessData.sharedResults.type.id == 1 then
			itemData = [[
Glorious Vanity ]] .. label .. [[
Timeless Jewel
League: Legion
Limited to: 1 Historic
Variant: Doryani (Corrupted Soul)
Variant: Xibaqua (Divine Flesh)
Variant: Ahuana (Immortal Ambition)
Selected Variant: ]] .. variant .. "\n" ..[[
Radius: Large
Implicits: 0
{variant:1}Bathed in the blood of ]] .. data.seed .. [[ sacrificed in the name of Doryani
{variant:2}Bathed in the blood of ]] .. data.seed .. [[ sacrificed in the name of Xibaqua
{variant:3}Bathed in the blood of ]] .. data.seed .. [[ sacrificed in the name of Ahuana
Passives in radius are Conquered by the Vaal
Historic
]]
		elseif timelessData.sharedResults.type.id == 2 then
			itemData = [[
Lethal Pride ]] .. label .. [[
Timeless Jewel
League: Legion
Limited to: 1 Historic
Variant: Kaom (Strength of Blood)
Variant: Rakiata (Tempered by War)
Variant: Akoya (Chainbreaker)
Selected Variant: ]] .. variant .. "\n" .. [[
Radius: Large
Implicits: 0
{variant:1}Commanded leadership over ]] .. data.seed .. [[ warriors under Kaom
{variant:2}Commanded leadership over ]] .. data.seed .. [[ warriors under Rakiata
{variant:3}Commanded leadership over ]] .. data.seed .. [[ warriors under Akoya
Passives in radius are Conquered by the Karui
Historic
]]
		elseif timelessData.sharedResults.type.id == 3 then
			itemData = [[
Brutal Restraint ]] .. label .. [[
Timeless Jewel
League: Legion
Limited to: 1 Historic
Variant: Asenath (Dance with Death)
Variant: Nasima (Second Sight)
Variant: Balbala (The Traitor)
Selected Variant: ]] .. variant .. "\n" .. [[
Radius: Large
Implicits: 0
{variant:1}Denoted service of ]] .. data.seed .. [[ dekhara in the akhara of Asenath
{variant:2}Denoted service of ]] .. data.seed .. [[ dekhara in the akhara of Nasima
{variant:3}Denoted service of ]] .. data.seed .. [[ dekhara in the akhara of Balbala
Passives in radius are Conquered by the Maraketh
Historic
]]
		elseif timelessData.sharedResults.type.id == 4 then
			local altVariant = timelessData.sharedResults.devotionVariant1.id ~= 1 and timelessData.sharedResults.devotionVariant1.id or math.random(2, 16)
			local altVariant2 = timelessData.sharedResults.devotionVariant2.id ~= 1 and timelessData.sharedResults.devotionVariant2.id or math.random(2, 16)
			if altVariant == altVariant2 then
				altVariant = altVariant % 15 + 2
			end
			itemData = [[
Militant Faith ]] .. label .. [[
Timeless Jewel
League: Legion
Limited to: 1 Historic
Has Alt Variant: true
Has Alt Variant Two: true
Variant: Avarius (Power of Purpose)
Variant: Dominus (Inner Conviction)
Variant: Maxarius (Transcendence)
Variant: Totem Damage
Variant: Brand Damage
Variant: Channelling Damage
Variant: Area Damage
Variant: Elemental Damage
Variant: Elemental Resistances
Variant: Effect of non-Damaging Ailments
Variant: Elemental Ailment Duration
Variant: Duration of Curses
Variant: Minion Attack and Cast Speed
Variant: Minions Accuracy Rating
Variant: Mana Regen
Variant: Skill Cost
Variant: Non-Curse Aura Effect
Variant: Defences from Shield
Selected Variant: ]] .. variant .. "\n" .. [[
Selected Alt Variant: ]] .. altVariant + 2 .. "\n" .. [[
Selected Alt Variant Two: ]] .. altVariant2 + 2 .. "\n" .. [[
Radius: Large
Implicits: 0
{variant:1}Carved to glorify ]] .. data.seed .. [[ new faithful converted by High Templar Avarius
{variant:2}Carved to glorify ]] .. data.seed .. [[ new faithful converted by High Templar Dominus
{variant:3}Carved to glorify ]] .. data.seed .. [[ new faithful converted by High Templar Maxarius
{variant:4}4% increased Totem Damage per 10 Devotion
{variant:5}4% increased Brand Damage per 10 Devotion
{variant:6}Channelling Skills deal 4% increased Damage per 10 Devotion
{variant:7}4% increased Area Damage per 10 Devotion
{variant:8}4% increased Elemental Damage per 10 Devotion
{variant:9}+2% to all Elemental Resistances per 10 Devotion
{variant:10}3% increased Effect of non-Damaging Ailments on Enemies per 10 Devotion
{variant:11}4% reduced Elemental Ailment Duration on you per 10 Devotion
{variant:12}4% reduced Duration of Curses on you per 10 Devotion
{variant:13}1% increased Minion Attack and Cast Speed per 10 Devotion
{variant:14}Minions have +60 to Accuracy Rating per 10 Devotion
{variant:15}Regenerate 0.6 Mana per Second per 10 Devotion
{variant:16}1% reduced Mana Cost of Skills per 10 Devotion
{variant:17}1% increased effect of Non-Curse Auras per 10 Devotion
{variant:18}3% increased Defences from Equipped Shield per 10 Devotion
Passives in radius are Conquered by the Templars
Historic
]]
		elseif timelessData.sharedResults.type.id == 5 then
			itemData = [[
Elegant Hubris ]] .. label .. [[
Timeless Jewel
League: Legion
Limited to: 1 Historic
Variant: Cadiro (Supreme Decadence)
Variant: Victario (Supreme Grandstanding)
Variant: Caspiro (Supreme Ostentation)
Selected Variant: ]] .. variant .. "\n" .. [[
Radius: Large
Implicits: 0
{variant:1}Commissioned ]] .. data.seed .. [[ coins to commemorate Cadiro
{variant:2}Commissioned ]] .. data.seed .. [[ coins to commemorate Victario
{variant:3}Commissioned ]] .. data.seed .. [[ coins to commemorate Caspiro
Passives in radius are Conquered by the Eternal Empire
Historic
]]
		elseif timelessData.sharedResults.type.id == 6 then
			itemData = [[
Heroic Tragedy ]] .. label .. [[
Timeless Jewel
League: Legion
Limited to: 1 Historic
Variant: Vorana (Black Scythe Training)
Variant: Uhtred (Celestial Mathematics)
Variant: Medved (The Unbreaking Circle)
Selected Variant: ]] .. variant .. "\n" .. [[
Radius: Large
Implicits: 0
{variant:1}Remembrancing ]] .. data.seed .. [[ songworthy deeds by the line of Vorana
{variant:2}Remembrancing ]] .. data.seed .. [[ songworthy deeds by the line of Uhtred
{variant:3}Remembrancing ]] .. data.seed .. [[ songworthy deeds by the line of Medved
Passives in radius are Conquered by the Kalguur
Historic
]]
		elseif timelessData.sharedResults.type.id == 7 then
			itemData = [[
Festering Vengeance ]] .. label .. [[
Murderous Eye Jewel
League: Allflame
Limited to: 1 Historic
Implicits: 0
Subjugating ]] .. data.seed .. [[ souls in the thrall of Tecrod
Passives affected are Conquered by the Abyssal
Historic
]]
		elseif timelessData.sharedResults.type.id == 8 then
			itemData = [[
Extinguishing Grasp ]] .. label .. [[
Searching Eye Jewel
League: Allflame
Limited to: 1 Historic
Implicits: 0
Subjugating ]] .. data.seed .. [[ souls in the thrall of Ulaman
Passives affected are Conquered by the Abyssal
Historic
]]
		elseif timelessData.sharedResults.type.id == 9 then
			itemData = [[
Baleful Dominion ]] .. label .. [[
Hypnotic Eye Jewel
League: Allflame
Limited to: 1 Historic
Implicits: 0
Subjugating ]] .. data.seed .. [[ souls in the thrall of Kurgal
Passives affected are Conquered by the Abyssal
Historic
]]
		elseif timelessData.sharedResults.type.id == 10 then
			itemData = [[
Destructive Aspiration ]] .. label .. [[
Ghastly Eye Jewel
League: Allflame
Limited to: 1 Historic
Implicits: 0
Subjugating ]] .. data.seed .. [[ souls in the thrall of Amanamu
Passives affected are Conquered by the Abyssal
Historic
]]
		elseif timelessData.sharedResults.type.id == 11 then
			itemData = [[
Reclaimed Malevolence ]] .. label .. [[
Assembled Eye Jewel
League: Allflame
Limited to: 1 Historic
Implicits: 0
Binding ]] .. data.seed .. [[ souls to phylacteries to sustain Zorath
Passives affected are Conquered by the Abyssal
Historic
]]
		end
		local item = new("Item", itemData)
		build.itemsTab:AddItem(item, true)
		build.itemsTab:PopulateSlots()
		timelessData.searchResults[index].label = "^xB2B2B2" .. timelessData.searchResults[index].label
	end
end

-- Builds the timeless jewel search model; search state lives in build.timelessData
-- and the returned models (also stored as self.timelessSearchControls) drive the search
function TreeTabClass:FindTimelessJewel()
	local treeData = self.build.spec.tree
	local legionNodes = treeData.legion.nodes
	local legionAdditions = treeData.legion.additions
	local timelessData = self.build.timelessData
	local controls = { }
	local modData = { }
	local ignoredMods = { "Might of the Vaal", "Legacy of the Vaal", "Strength", "Add Strength", "Dex", "Add Dexterity", "Devotion", "Price of Glory", "Ward" }
	local totalMods = { [2] = "Strength", [3] = "Dexterity", [4] = "Devotion" }
	local totalModIDs = {
		["total_strength"] = { ["karui_notable_add_strength"] = true, ["karui_attribute_strength"] = true, ["karui_small_strength"] = true },
		["total_dexterity"] = { ["maraketh_notable_add_dexterity"] = true, ["maraketh_attribute_dex"] = true, ["maraketh_small_dex"] = true },
		["total_devotion"] = { ["templar_notable_devotion"] = true, ["templar_devotion_node"] = true, ["templar_small_devotion"] = true }
	}
	local reverseTotalModIDs = {
		["karui_notable_add_strength"] = true,
		["karui_attribute_strength"] = true,
		["karui_small_strength"] = true,
		["maraketh_notable_add_dexterity"] = true,
		["maraketh_attribute_dex"] = true,
		["maraketh_small_dex"] = true,
		["templar_notable_devotion"] = true,
		["templar_devotion_node"] = true,
		["templar_small_devotion"] = true
	}
	local jewelTypes = {
		{ label = "Glorious Vanity", name = "vaal", id = 1 },
		{ label = "Lethal Pride", name = "karui", id = 2 },
		{ label = "Brutal Restraint", name = "maraketh", id = 3 },
		{ label = "Militant Faith", name = "templar", id = 4 },
		{ label = "Elegant Hubris", name = "eternal", id = 5 },
		{ label = "Heroic Tragedy", name = "kalguur", id = 6 },
		{ label = "Festering Vengeance", name = "abyss_murderous", id = 7 },
		{ label = "Extinguishing Grasp", name = "abyss_searching", id = 8 },
		{ label = "Baleful Dominion", name = "abyss_hypnotic", id = 9 },
		{ label = "Destructive Aspiration", name = "abyss_ghastly", id = 10 },
		{ label = "Reclaimed Malevolence", name = "abyss_special", id = 11 }
	}
	-- rebuild `timelessData.jewelType` as we only store the minimum amount of `jewelType` data in build XML
	if next(timelessData.jewelType) then
		for idx, jewelType in ipairs(jewelTypes) do
			if jewelType.id == timelessData.jewelType.id then
				timelessData.jewelType = jewelType
				break
			end
		end
	else
		timelessData.jewelType = jewelTypes[1]
	end
	local conquerorTypes = {
		[1] = {
			{ label = "Any", id = 1 },
			{ label = "Doryani (Corrupted Soul)", id = 2 },
			{ label = "Xibaqua (Divine Flesh)", id = 3 },
			{ label = "Ahuana (Immortal Ambition)", id = 4 }
		},
		[2] = {
			{ label = "Any", id = 1 },
			{ label = "Kaom (Strength of Blood)", id = 2 },
			{ label = "Rakiata (Tempered by War)", id = 3 },
			{ label = "Akoya (Chainbreaker)", id = 4 }
		},
		[3] = {
			{ label = "Any", id = 1 },
			{ label = "Asenath (Dance with Death)", id = 2 },
			{ label = "Nasima (Second Sight)", id = 3 },
			{ label = "Balbala (The Traitor)", id = 4 }
		},
		[4] = {
			{ label = "Any", id = 1 },
			{ label = "Avarius (Power of Purpose)", id = 2 },
			{ label = "Dominus (Inner Conviction)", id = 3 },
			{ label = "Maxarius (Transcendence)", id = 4 }
		},
		[5] = {
			{ label = "Any", id = 1 },
			{ label = "Cadiro (Supreme Decadence)", id = 2 },
			{ label = "Victario (Supreme Grandstanding)", id = 3 },
			{ label = "Caspiro (Supreme Ostentation)", id = 4 }
		},
		[6] = {
			{ label = "Any", id = 1 },
			{ label = "Vorana (Black Scythe Training)", id = 2 },
			{ label = "Uhtred (Celestial Mathematics)", id = 3 },
			{ label = "Medved (The Unbreaking Circle)", id = 4 }
		},
		[7] = { { label = "Tecrod (Overwhelming Hate)", id = 1 } },
		[8] = { { label = "Ulaman (Weighted Exchange)", id = 1 } },
		[9] = { { label = "Kurgal (Reconstructed Essence)", id = 1 } },
		[10] = { { label = "Amanamu (The Loyal Few)", id = 1 } },
		[11] = { { label = "Zorath", id = 1 } }
	}
	-- rebuild `timelessData.conquerorType` as we only store the minimum amount of `conquerorType` data in build XML
	if next(timelessData.conquerorType) then
		for idx, conquerorType in ipairs(conquerorTypes[timelessData.jewelType.id]) do
			if conquerorType.id == timelessData.conquerorType.id then
				timelessData.conquerorType = conquerorType
				break
			end
		end
	else
		timelessData.conquerorType = conquerorTypes[timelessData.jewelType.id][1]
	end
	local devotionVariants = {
		{ id = 1 , label = "Any" },
		{ id = 2 , label = "Totem Damage" },
		{ id = 3 , label = "Brand Damage" },
		{ id = 4 , label = "Channelling Damage" },
		{ id = 5 , label = "Area Damage" },
		{ id = 6 , label = "Elemental Damage" },
		{ id = 7 , label = "Elemental Resistances" },
		{ id = 8 , label = "Effect of non-Damaging Ailments" },
		{ id = 9 , label = "Elemental Ailment Duration" },
		{ id = 10, label = "Duration of Curses" },
		{ id = 11, label = "Minion Attack and Cast Speed" },
		{ id = 12, label = "Minions Accuracy Rating" },
		{ id = 13, label = "Mana Regen" },
		{ id = 14, label = "Skill Cost" },
		{ id = 15, label = "Non-Curse Aura Effect" },
		{ id = 16, label = "Defences from Shield" }
	}
	local abyssAscendancyOptions = { }
	for _, node in pairs(legionNodes) do
		if node.id:match("^abyss_special_ascendancy_notable_%d+$") then
			t_insert(abyssAscendancyOptions, {
				label = node.dn,
				descriptions = copyTable(node.sd),
				id = node.id,
			})
		end
	end
	t_sort(abyssAscendancyOptions, function(a, b) return a.label < b.label end)
	t_insert(abyssAscendancyOptions, 1, { label = "Any" })
	local jewelSockets = { }
	t_insert(jewelSockets, {
		label = "All Sockets",
		keystone = "Multi-Socket Search",
		id = -1
	})
	for socketId, socketData in pairs(self.build.spec.nodes) do
		if socketData.isJewelSocket and socketData.name ~= "Charm Socket"then
			local keystone = "Unknown"
			if socketId == 26725 then
				keystone = "Marauder"
			elseif socketId == 54127 then
				keystone = "Duelist"
			elseif socketId == 7960 then
				keystone = "Templar/Witch"
			else
				local minDistance = math.huge
				for _, nodeInRadius in pairs(treeData.nodes[socketId].nodesInRadius[3]) do
					if nodeInRadius.isKeystone then
						local distance = math.sqrt((nodeInRadius.x - socketData.x) ^ 2 + (nodeInRadius.y - socketData.y) ^ 2)
						if distance < minDistance then
							keystone = nodeInRadius.name
							minDistance = distance
						end
					end
				end
			end
			local label = keystone .. ": " .. socketId
			if self.build.spec.allocNodes[socketId] then
				label = "# " .. label
			end
			t_insert(jewelSockets, {
				label = label,
				keystone = keystone,
				id = socketId
			})
		end
	end
	-- Sort all sockets except all sockets option
	local allSocketsEntry = t_remove(jewelSockets, 1)
	t_sort(jewelSockets, function(a, b) return a.label < b.label end)
	t_insert(jewelSockets, 1, allSocketsEntry)
	-- rebuild `timelessData.jewelSocket` as we only store the minimum amount of `jewelSocket` data in build XML
	if next(timelessData.jewelSocket) then
		for idx, jewelSocket in ipairs(jewelSockets) do
			if jewelSocket.id == timelessData.jewelSocket.id then
				timelessData.jewelSocket = jewelSocket
				break
			end
		end
	else
		timelessData.jewelSocket = jewelSockets[1]
	end

	local function buildMods()
		wipeTable(modData)
		local smallModData = { }
		for _, node in pairs(legionNodes) do
			if node.id:match("^" .. timelessData.jewelType.name .. "_.+")
			and not node.id:match("^abyss_special_ascendancy_notable_")
			and not isValueInArray(ignoredMods, node.dn) and not node.ks then
				if node["not"] then
					t_insert(modData, {
						label = node.dn .. "                                                " .. node.sd[1],
						descriptions = copyTable(node.sd),
						type = timelessData.jewelType.name,
						id = node.id
					})
					if node.sd[2] then
						modData[#modData].label = modData[#modData].label .. " " .. node.sd[2]
					end
				else
					t_insert(smallModData, {
						label = node.dn,
						descriptions = copyTable(node.sd),
						type = timelessData.jewelType.name,
						id = node.id
					})
				end
			end
		end
		for _, addition in pairs(legionAdditions) do
			-- exclude passives that are already added (vaal, attributes, devotion)
			if addition.id:match("^" .. timelessData.jewelType.name .. "_.+") and not isValueInArray(ignoredMods, addition.dn) and timelessData.jewelType.name ~= "vaal" then
				t_insert(modData, {
					label = addition.dn,
					descriptions = copyTable(addition.sd),
					type = timelessData.jewelType.name,
					id = addition.id
				})
			end
		end
		t_sort(modData, function(a, b) return a.label < b.label end)
		t_sort(smallModData, function (a, b) return a.label < b.label end)
		if totalMods[timelessData.jewelType.id] then
			t_insert(modData, 1, {
				label = "Total " .. totalMods[timelessData.jewelType.id],
				descriptions = { "This is a hybrid node containing all additions to " .. totalMods[timelessData.jewelType.id] },
				type = timelessData.jewelType.name,
				id = "total_" .. totalMods[timelessData.jewelType.id]:lower(),
				totalMod = true
			})
		end
		t_insert(modData, 1, { label = "..." })
		for i = 1, #smallModData do
			modData[#modData + 1] = smallModData[i]
		end
	end

	local function getNodeWeights()
		local nodeWeights = {
			[1] = s_format("%.3f", controls.nodeSlider.val * 10),
			[2] = s_format("%.3f", controls.nodeSlider2.val * 10),
			[3] = controls.nodeSlider3.val == 1 and "required" or s_format("%.f", controls.nodeSlider3.val * 500)
		}
		for i, nodeWeight in ipairs(nodeWeights) do
			if tonumber(nodeWeight) ~= nil then
				nodeWeights[i] = round(tonumber(nodeWeight), 3)
			end
		end
		return nodeWeights
	end

	local searchListTbl = { }
	local searchListFallbackTbl = { }
	local function parseSearchList(mode, fallback)
		if mode == 0 then
			if fallback then
				-- timelessData.searchListFallback => searchListFallbackTbl
				if timelessData.searchListFallback then
					searchListFallbackTbl = { }
					for inputLine in timelessData.searchListFallback:gmatch("[^\r\n]+") do
						searchListFallbackTbl[#searchListFallbackTbl + 1] = { }
						for splitLine in inputLine:gmatch("([^,%s]+)") do
							searchListFallbackTbl[#searchListFallbackTbl][#searchListFallbackTbl[#searchListFallbackTbl] + 1] = splitLine
						end
					end
				end
			else
				-- timelessData.searchList => searchListTbl
				if timelessData.searchList then
					searchListTbl = { }
					for inputLine in timelessData.searchList:gmatch("[^\r\n]+") do
						searchListTbl[#searchListTbl + 1] = { }
						for splitLine in inputLine:gmatch("([^,%s]+)") do
							searchListTbl[#searchListTbl][#searchListTbl[#searchListTbl] + 1] = splitLine
						end
					end
				end
			end
		else
			if fallback then
				-- searchListFallbackTbl => controls.searchListFallback
				if controls.searchListFallback and controls.nodeSelect then
					local searchText = ""
					for _, curRow in ipairs(searchListFallbackTbl) do
						if curRow[1] == controls.nodeSelect.list[controls.nodeSelect.selIndex].id then
							local nodeWeights = getNodeWeights()
							curRow[2] = nodeWeights[1]
							curRow[3] = nodeWeights[2]
							curRow[4] = nodeWeights[3]
						end
						if #searchText > 0 then
							searchText = searchText .. "\n"
						end
						searchText = searchText .. t_concat(curRow, ", ")
					end
					if timelessData.searchListFallback ~= searchText then
						timelessData.searchListFallback = searchText
						controls.searchListFallback:SetText(searchText)
						self.build.modFlag = true
					end
				end
			else
				-- searchListTbl => controls.searchList
				if controls.searchList and controls.nodeSelect then
					local searchText = ""
					for _, curRow in ipairs(searchListTbl) do
						if curRow[1] == controls.nodeSelect.list[controls.nodeSelect.selIndex].id then
							local nodeWeights = getNodeWeights()
							curRow[2] = nodeWeights[1]
							curRow[3] = nodeWeights[2]
							curRow[4] = nodeWeights[3]
						end
						if #searchText > 0 then
							searchText = searchText .. "\n"
						end
						searchText = searchText .. t_concat(curRow, ", ")
					end
					if timelessData.searchList ~= searchText then
						timelessData.searchList = searchText
						controls.searchList:SetText(searchText)
						self.build.modFlag = true
					end
				end
			end
		end
	end
	parseSearchList(0, false) -- initial load: [timelessData.searchList => searchListTbl]
	parseSearchList(0, true)  -- initial load: [timelessData.searchListFallback => searchListFallbackTbl]
	local function updateSearchList(text, fallback)
		if fallback then
			timelessData.searchListFallback = text
			controls.searchListFallback:SetText(text)
		else
			timelessData.searchList = text
			controls.searchList:SetText(text)
		end
		parseSearchList(0, fallback)
		self.build.modFlag = true
	end

	controls.devotionSelect1 = new("Selector", devotionVariants, function(index, value)
		timelessData.devotionVariant1 = index
	end)
	controls.devotionSelect1.selIndex = timelessData.devotionVariant1
	controls.devotionSelect2 = new("Selector", devotionVariants, function(index, value)
		timelessData.devotionVariant2 = index
	end)
	controls.devotionSelect2.selIndex = timelessData.devotionVariant2

	local allocatedNodes = { }
	local protectedNodes = { }
	local protectedNodesCount = 0
	local setAllocatedNodes
	local clearProtected
	self.allocatedNodesInRadiusCount = 0

	controls.jewelSelect = new("Selector", jewelTypes, function(index, value)
		timelessData.jewelType = value
		controls.abyssAscendancySelect.selIndex = 1
		controls.conquerorSelect.list = conquerorTypes[timelessData.jewelType.id]
		controls.conquerorSelect.selIndex = 1
		timelessData.conquerorType = conquerorTypes[timelessData.jewelType.id][1]
		controls.nodeSelect.selIndex = 1
		buildMods()
		updateSearchList("", false)
		updateSearchList("", true)
		if controls.socketFilter.state then
			clearProtected()
			setAllocatedNodes()
		end
	end)
	controls.jewelSelect.selIndex = timelessData.jewelType.id

	controls.conquerorSelect = new("Selector", conquerorTypes[timelessData.jewelType.id], function(index, value)
		timelessData.conquerorType = value
		self.build.modFlag = true
	end)
	controls.conquerorSelect.selIndex = timelessData.conquerorType.id

	setAllocatedNodes = function()
		if timelessData.jewelSocket.id == -1 or not treeData.nodes[timelessData.jewelSocket.id] then
			return
		end
		wipeTable(allocatedNodes)
		local nodeOptions = { }
		if timelessData.jewelType.id == 11 then
			-- Reclaimed Malevolence can replace an allocated notable in the selected ascendancy.
			for nodeId in pairs(self.build.spec.allocNodes) do
				local baseNode = treeData.nodes[nodeId]
				if baseNode and baseNode.ascendancyName == self.build.spec.curAscendClassName and baseNode.type == "Notable" then
					allocatedNodes[nodeId] = true
					t_insert(nodeOptions, { label = baseNode.dn, descriptions = copyTable(baseNode.sd) })
				end
			end
		else
			local radiusNodes = treeData.nodes[timelessData.jewelSocket.id].nodesInRadius[3]
			for nodeId in pairs(radiusNodes) do
				if self.build.calcsTab.mainEnv.grantedPassives[nodeId] ~= nil or self.build.spec.allocNodes[nodeId] ~= nil then
					allocatedNodes[nodeId] = true
					if treeData.nodes[nodeId] and treeData.nodes[nodeId].isNotable then
						local baseNode = treeData.nodes[nodeId]
						t_insert(nodeOptions, { label = baseNode.dn, descriptions = copyTable(baseNode.sd) })
					end
				end
			end
		end
		t_sort(nodeOptions, function(a, b) return a.label < b.label end)
		controls.protectAllocatedSelect:SetList(nodeOptions)
		self.allocatedNodesInRadiusCount = #nodeOptions
	end

	
	controls.socketSelect = new("Selector", jewelSockets, function(index, value)
		timelessData.jewelSocket = value
		setAllocatedNodes() -- reset list when changing sockets
		self.build.modFlag = true
	end)
	-- we need to search through `jewelSockets` for the correct `id` as the `idx` can become stale due to dynamic sorting
	for idx, jewelSocket in ipairs(jewelSockets) do
		if jewelSocket.id == timelessData.jewelSocket.id then
			controls.socketSelect.selIndex = idx
			break
		end
	end
	clearProtected = function()
		protectedNodesCount = 0
		protectedNodes = { }
		for index, _ in pairs(controls) do
			if index:find("protected:") then
				controls[index] = nil
			end
		end
	end
	
	controls.socketFilter = { state = timelessData.socketFilter }
	controls.socketFilter.changeFunc = function(value)
		controls.socketFilter.state = value
		timelessData.socketFilter = value
		self.build.modFlag = true

		if value then
			setAllocatedNodes()
		else
			clearProtected()
		end
	end

	-- Protect notables that must not be replaced by Militant Faith or Reclaimed Malevolence.
	controls.protectAllocatedSelect = new("Selector")
	controls.protectAllocatedButtonAdd = { onClick = function()
		local selValue = controls.protectAllocatedSelect:GetSelValue()
		local nodeName = selValue and selValue.label
		if nodeName and not controls["protected:"..nodeName] then
			protectedNodesCount = protectedNodesCount + 1
			t_insert(protectedNodes, nodeName)
			controls["protected:"..nodeName] = { label = "^7"..nodeName }
		end
	end }
	controls.protectAllocatedButtonClear = { onClick = function()
		clearProtected()
	end }
	-- set list on load
	if controls.socketFilter.state then
		setAllocatedNodes()
	end

	-- This requirement is separate from ordinary node weights.
	controls.abyssAscendancySelect = new("Selector", abyssAscendancyOptions, function(index, value)
		local searchLines = { }
		for _, searchRow in ipairs(searchListTbl) do
			if not searchRow[1]:match("^abyss_special_ascendancy_notable_") then
				t_insert(searchLines, t_concat(searchRow, ", "))
			end
		end
		if value.id then
			t_insert(searchLines, value.id .. ", 1, 0, 1")
		end
		updateSearchList(t_concat(searchLines, "\n"), false)
	end)
	controls.abyssAscendancySelect.selIndex = 1
	for _, searchRow in ipairs(searchListTbl) do
		for optionIndex, option in ipairs(abyssAscendancyOptions) do
			if option.id == searchRow[1] then
				controls.abyssAscendancySelect.selIndex = optionIndex
				break
			end
		end
	end
	controls.abyssAscendancySelect.tooltipFunc = function(tooltip, mode, index, value)
		tooltip:Clear()
		if mode ~= "OUT" and value.descriptions then
			for _, line in ipairs(value.descriptions) do
				tooltip:AddLine(16, "^7" .. line)
			end
		end
	end
	controls.protectAllocatedSelect.tooltipFunc = function(tooltip, mode, index, value)
		tooltip:Clear()
		if mode ~= "OUT" and value and value.descriptions then
			for _, line in ipairs(value.descriptions) do
				tooltip:AddLine(16, "^7" .. line)
			end
		end
	end

	local socketFilterAdditionalDistanceMAX = 10
	controls.socketFilterAdditionalDistance = new("SliderModel", function(value)
		timelessData.socketFilterDistance = m_floor(value * socketFilterAdditionalDistanceMAX + 0.01)
	end)
	controls.socketFilterAdditionalDistance:SetVal((timelessData.socketFilterDistance or 0) / socketFilterAdditionalDistanceMAX)

	controls.nodeSlider = new("SliderModel", function(value)
		parseSearchList(1, controls.searchListFallback and controls.searchListFallback.shown or false)
	end)
	controls.nodeSlider:SetVal(0.1)

	controls.nodeSlider2 = new("SliderModel", function(value)
		parseSearchList(1, controls.searchListFallback and controls.searchListFallback.shown or false)
	end)
	controls.nodeSlider2:SetVal(0.1)

	controls.nodeSlider3 = new("SliderModel", function(value)
		parseSearchList(1, controls.searchListFallback and controls.searchListFallback.shown or false)
	end)
	controls.nodeSlider3:SetVal(0)

	local function updateSliders(sliderData)
		if sliderData[2] == "required" then
			controls.nodeSlider.val = 1
		else
			controls.nodeSlider.val = m_min(m_max((tonumber(sliderData[2]) or 0) / 10, 0), 1)
		end
		if controls.nodeSlider2.enabled then
			if sliderData[3] == "required" then
				controls.nodeSlider2.val = 1
			else
				controls.nodeSlider2.val = m_min(m_max((tonumber(sliderData[3]) or 0) / 10, 0), 1)
			end
		end
		if sliderData[4] == "required" then
			controls.nodeSlider3.val = 1
		else
			controls.nodeSlider3.val = m_min(m_max((tonumber(sliderData[4]) or 0) / 500, 0), 1)
		end
	end

	buildMods()
	local function getLegionStatLabels(legionPassive)
		local statCount = timelessData.jewelType.id >= 7 and #legionPassive.sortedStats or #legionPassive.sd
		if statCount > #legionPassive.sd then
			return statCount, "Minimum value: " .. legionPassive.sd[1], "Maximum value: " .. legionPassive.sd[1]
		end
		return statCount,
			statCount == 1 and t_concat(legionPassive.sd, " + ") or legionPassive.sd[1] or "None",
			legionPassive.sd[2] or "None"
	end
	controls.nodeSelect = new("Selector", modData, function(index, value)
		if value.id then
			local statCount = 0
			for _, legionNode in ipairs(legionNodes) do
				if legionNode.id == value.id then
					statCount = getLegionStatLabels(legionNode)
					break
				end
			end
			if statCount == 0 then
				for _, legionAddition in ipairs(legionAdditions) do
					if legionAddition.id == value.id then
						statCount = getLegionStatLabels(legionAddition)
						break
					end
				end
			end
			if statCount <= 1 then
				controls.nodeSlider2.val = 0
			end
			controls.nodeSlider2.enabled = statCount > 1

			local nodeWeights = getNodeWeights()
			local newNode = value.id .. ", " .. nodeWeights[1] .. ", " .. nodeWeights[2] .. ", " .. nodeWeights[3]
			if controls.searchListFallback and controls.searchListFallback.shown then
				for _, searchRow in ipairs(searchListFallbackTbl) do
					-- update nodeSlider values and prevent duplicate searchList entries
					if searchRow[1] == value.id then
						updateSliders(searchRow)
						return
					end
				end
				controls.searchListFallback:Insert((#controls.searchListFallback.buf > 0 and "\n" or "") .. newNode)
			else
				for _, searchRow in ipairs(searchListTbl) do
					-- update nodeSlider values and prevent duplicate searchList entries
					if searchRow[1] == value.id then
						updateSliders(searchRow)
						return
					end
				end
				controls.searchList:Insert((#controls.searchList.buf > 0 and "\n" or "") .. newNode)
			end
			self.build.modFlag = true
		end
	end)
	controls.nodeSelect.tooltipFunc = function(tooltip, mode, index, value)
		tooltip:Clear()
		if mode ~= "OUT" and value.descriptions then
			for _, line in ipairs(value.descriptions) do
				tooltip:AddLine(16, "^7" .. line)
			end
		end
	end

	local function generateFallbackWeights(nodes, powerStat)
		local calcFunc, calcBase = self.build.calcsTab:GetMiscCalculator(self.build)
		local newList = { }
		local basePower = data.powerStatList.GetFromOutput(calcBase, powerStat)
		for _, newNode in ipairs(nodes) do
			local powerEntry = { id = newNode.id }
			-- nodes that have multiple lines are represented as a list in newNode.node
			local nodeLines = newNode.node or { newNode }
			for i = 1, #nodeLines do
				local node = nodeLines[i]
				local nodeOutput = calcFunc({ addNodes = { [node] = true } })
				local nodePower = data.powerStatList.GetFromOutput(nodeOutput, powerStat)
				-- avoid infinity
				if basePower == 0 then
					powerEntry["weight" .. i] = 0
				else
					local powerGain = (nodePower - basePower) /
						-- normalize with absolute base power so that the result isn't negative
						math.abs(basePower)
					powerEntry["weight" .. i] = powerGain / (node.divisor or newNode.divisor or 1)
				end
			end
			t_insert(newList, powerEntry)
		end
		return newList
	end

	local function setupFallbackWeights()
		-- replaceHelperFunc is duplicated from PassiveSpec.lua
		local replaceHelperFunc = function(statToFix, statKey, statMod, value)
			if statMod.fmt == "g" then -- note the only one we actually care about is "Ritual of Flesh" life regen
				if statKey:find("per_minute") then
					value = round(value / 60, 1)
				elseif statKey:find("permyriad") then
					value = value / 100
				elseif statKey:find("_ms") then
					value = value / 1000
				end
			end
			--if statMod.fmt == "d" then -- only ever d or g, and we want both past here
			if statMod.min ~= statMod.max then
				return statToFix:gsub("%(" .. statMod.min .. "%-" .. statMod.max .. "%)", value)
			elseif statMod.min ~= value then -- only true for might/legacy of the vaal which can combine stats
				return statToFix:gsub(statMod.min, value)
			end
			return statToFix -- if it doesn't need to be changed
		end
		local function buildStatModLists(legionPassive)
			-- Give each stat its own mod list even when several stats share one display line.
			local modLists = { }
			for statIndex, statKey in ipairs(legionPassive.sortedStats) do
				local statValues = { }
				for key in pairs(legionPassive.stats) do
					statValues[key] = key == statKey and 100 or 0
				end
				local line = data.describeStats(statValues, "stat_descriptions")[1]
				modLists[statIndex] = { modList = modLib.parseMod(line), divisor = 100 }
			end
			return modLists
		end

		local nodes = { }
		local usesVariableRolls = timelessData.jewelType.id == 1 or timelessData.jewelType.id >= 7
		for _, modNode in ipairs(modData) do
			if modNode.id then
				local newNode = nil
				for _, legionNode in ipairs(legionNodes) do
					if legionNode.id == modNode.id or (totalModIDs[modNode.id] and totalModIDs[modNode.id][legionNode.id]) then
							newNode = { }
							newNode.id = modNode.id
							if usesVariableRolls then
								if #legionNode.sortedStats > 1 then
									newNode.calcMultiple = true
									if legionNode.modListGenerated then
										newNode.node = copyTable(legionNode.modListGenerated)
									else
										local modLists = buildStatModLists(legionNode)
										legionNode.modListGenerated = copyTable(modLists)
										newNode.node = copyTable(modLists)
									end
									for _, node in ipairs(newNode.node) do
										node.id = legionNode.id
									end
								else
									local originalLine = legionNode.sd[1]
									local line = replaceHelperFunc(originalLine, legionNode.sortedStats[1], legionNode.stats[legionNode.sortedStats[1]], 100)
									if line == originalLine and #legionNode.sd > 1 then
										-- Some fixed game stats represent several display lines; score the complete effect together.
										newNode.modList = legionNode.modList
									elseif legionNode.modListGenerated then
										newNode.modList = copyTable(legionNode.modListGenerated)
									else
										local modList, extra = modLib.parseMod(line)
										legionNode.modListGenerated = modList
										newNode.modList = modList
									end
									newNode.divisor = line ~= originalLine and 100 or 1
								end
							else
								newNode.modList = legionNode.modList
								if modNode.totalMod then
									newNode.divisor = legionNode.modList[1].value
								end
							end
						break
					end
				end
				if not newNode then
					for _, legionAddition in ipairs(legionAdditions) do
						if legionAddition.id == modNode.id or (totalModIDs[modNode.id] and totalModIDs[modNode.id][legionAddition.id]) then
							newNode = { }
							newNode.id = modNode.id
							if usesVariableRolls and #legionAddition.sortedStats > 1 then
								newNode.calcMultiple = true
								if legionAddition.modListGenerated then
									newNode.node = copyTable(legionAddition.modListGenerated)
								else
									local modLists = buildStatModLists(legionAddition)
									legionAddition.modListGenerated = copyTable(modLists)
									newNode.node = copyTable(modLists)
								end
								for _, node in ipairs(newNode.node) do
									node.id = legionAddition.id
								end
							elseif legionAddition.modList then
								newNode.modList = legionAddition.modList
							elseif legionAddition.modListGenerated then
								newNode.modList = legionAddition.modListGenerated
							else
								-- generate modList
								local originalLine = legionAddition.sd[1]
								local line = originalLine
								if usesVariableRolls then
									for key, stat in pairs(legionAddition.stats) do -- should only be length 1
										line = replaceHelperFunc(line, key, stat, 100)
									end
								end
								local modList, extra = modLib.parseMod(line)
								legionAddition.modListGenerated = modList
								newNode.modList = modList
								newNode.divisor = line ~= originalLine and 100 or 1
							end
							if not usesVariableRolls and modNode.totalMod then
								newNode.divisor = newNode.modList[1].value
							end
							break
						end
					end
				end
				if newNode then
					t_insert(nodes, newNode)
				end
			end
		end
		local output = generateFallbackWeights(nodes, controls.fallbackWeightsList.list[controls.fallbackWeightsList.selIndex])
		local newList = ""
		local weightScalar = 100
		for _, legionNode in ipairs(output) do
			if legionNode.weight1 ~= 0 or (legionNode.weight2 and legionNode.weight2 ~= 0) then
				if #newList > 0 then
					newList = newList .. "\n"
				end
				newList = newList .. legionNode.id .. ", " .. round(legionNode.weight1 * weightScalar, 3) .. ", " .. round((legionNode.weight2 or 0) * weightScalar, 3) .. ", 0"
			end
		end
		updateSearchList(newList, true)
	end

	local fallbackWeightsList = { }
	for _, stat in ipairs(data.powerStatList) do
		if not stat.ignoreForItems and stat.label ~= "Name" then
			t_insert(fallbackWeightsList, {
				label = "Sort by " .. stat.label,
				stat = stat.stat,
				transform = stat.transform,
			})
		end
	end
	controls.fallbackWeightsList = new("Selector", fallbackWeightsList, function(index)
		timelessData.fallbackWeightMode.idx = index
	end)
	controls.fallbackWeightsList.selIndex = timelessData.fallbackWeightMode.idx or 1
	controls.fallbackWeightsButton = { onClick = function()
		setupFallbackWeights()
	end }

	-- Text models for the desired/fallback search lists; the underlying state lives in timelessData
	local function newSearchListModel(fallback)
		local model = { buf = "" }
		local function textChanged(value)
			if fallback then
				timelessData.searchListFallback = value
			else
				timelessData.searchList = value
			end
			parseSearchList(0, fallback)
			self.build.modFlag = true
		end
		function model:SetText(text, notify)
			self.buf = text or ""
			if notify then
				textChanged(self.buf)
			end
		end
		function model:Insert(text)
			self.buf = self.buf .. text
			textChanged(self.buf)
		end
		return model
	end
	controls.searchList = newSearchListModel(false)
	controls.searchList.shown = true
	controls.searchList:SetText(timelessData.searchList and timelessData.searchList or "")

	controls.searchListFallback = newSearchListModel(true)
	controls.searchListFallback.shown = false
	controls.searchListFallback:SetText(timelessData.searchListFallback and timelessData.searchListFallback or "")

	controls.searchListButton = { onClick = function()
		controls.searchListFallback.shown = false
		controls.searchList.shown = true
	end }
	controls.searchListFallbackButton = { onClick = function()
		controls.searchList.shown = false
		controls.searchListFallback.shown = true
	end }

	-- Search result list model; double-clicking a result generates the jewel and adds it to the build
	controls.searchResults = { list = timelessData.searchResults }
	local searchResultsBuild = self.build
	function controls.searchResults:OnSelClick(index, data, doubleClick)
		return timelessJewelResultClick(searchResultsBuild, timelessData, index, data, doubleClick)
	end

	-- Helper function to search a single socket
	local function searchSingleSocket(socketId, socketInfo)
		if not treeData.nodes[socketId] or not treeData.nodes[socketId].isJewelSocket then
			return nil
		end
		
		local radiusNodes = treeData.nodes[socketId].nodesInRadius[3]
		local isAbyssJewel = timelessData.jewelType.id >= 7
		local zorathPath = timelessData.jewelType.id == 11 and self.build.spec:GetShortestPathToClassStart(socketId)
		if timelessData.jewelType.id == 11 and not zorathPath then
			return nil
		end
		local allocatedNodes = { }
		local unAllocatedNodesDistance = { }
		local targetNodes = { }
		local targetSmallNodes = { ["attributeSmalls"] = 0, ["otherSmalls"] = 0 }
		local desiredNodes = { }
		local minimumWeights = { }
		local resultNodes = { }
		local rootNodes = { }
		local desiredIdx = 0
		local searchListCombinedTbl = { }
		local searchListNodeFound = { }
		
		for _, curRow in ipairs(searchListTbl) do
			searchListNodeFound[curRow[1]] = true
			searchListCombinedTbl[#searchListCombinedTbl + 1] = copyTable(curRow)
		end
		for _, curRow in ipairs(searchListFallbackTbl) do
			if not searchListNodeFound[curRow[1]] then
				searchListCombinedTbl[#searchListCombinedTbl + 1] = copyTable(curRow)
			end
		end
		
		for _, desiredNode in ipairs(searchListCombinedTbl) do
			if #desiredNode > 1 then
				local displayName = nil
				local singleStat = false
				if totalMods[timelessData.jewelType.id] and desiredNode[1] == "total_" .. totalMods[timelessData.jewelType.id]:lower() then
					desiredNode[1] = "totalStat"
					displayName = totalMods[timelessData.jewelType.id]
				end
				if displayName == nil then
					for _, legionNode in ipairs(legionNodes) do
						if legionNode.id == desiredNode[1] then
							-- non-vaal replacements only support one nodeWeight
							if timelessData.jewelType.id > 1 and timelessData.jewelType.id < 7 then
								singleStat = true
							end
							displayName = t_concat(legionNode.sd, " + ")
							break
						end
					end
				end
				if displayName == nil then
					for _, legionAddition in ipairs(legionAdditions) do
						if legionAddition.id == desiredNode[1] then
							-- Original timeless additions only store an ID; Abyss records also store their rolls.
							singleStat = timelessData.jewelType.id < 7
							displayName = t_concat(legionAddition.sd, " + ")
							break
						end
					end
				end
				if displayName ~= nil then
					for i, val in ipairs(desiredNode) do
						if singleStat and i == 2 then
							desiredNode[2] = tonumber(desiredNode[2]) or tonumber(desiredNode[3]) or 1
						end
						if val == "required" then
							desiredNode[i] = (singleStat and i == 2) and desiredNode[2] or 0
							if desiredNode[4] == nil or desiredNode[4] < 0.001 then
								desiredNode[4] = 0.001
							end
						end
					end
					if desiredNode[4] ~= nil and tonumber(desiredNode[4]) > 0 then
						t_insert(minimumWeights, { reqNode = desiredNode[1], weight = tonumber(desiredNode[4]) })
					end
					-- if we're protecting a node and the number of protected nodes is less than the total allocated in radius and the total desired nodes is less than the total allocated in radius
					-- these constraints avoid a blank result in the case where you set a min weight of 1 onto a non devotion stat with zero unprotected nodes
					if timelessData.jewelType.id == 4 and protectedNodesCount > 0 and protectedNodesCount < self.allocatedNodesInRadiusCount and (#searchListCombinedTbl < self.allocatedNodesInRadiusCount) then
						t_insert(minimumWeights, { reqNode = desiredNode[1], weight = 1 })
					end
					if desiredNodes[desiredNode[1]] then
						desiredNodes[desiredNode[1]] = {
							nodeWeight = tonumber(desiredNode[2]) or 0.001,
							nodeWeight2 = tonumber(desiredNode[3]) or 0.001,
							displayName = displayName or desiredNode[1],
							desiredIdx = desiredNodes[desiredNode[1]].desiredIdx
						}
					else
						desiredIdx = desiredIdx + 1
						desiredNodes[desiredNode[1]] = {
							nodeWeight = tonumber(desiredNode[2]) or 0.001,
							nodeWeight2 = tonumber(desiredNode[3]) or 0.001,
							displayName = displayName or desiredNode[1],
							desiredIdx = desiredIdx
						}
					end
				end
			end
		end
		wipeTable(searchListCombinedTbl)
		
		for _, class in pairs(treeData.classes) do
			rootNodes[class.startNodeId] = true
		end
		
		if controls.socketFilter.state and not isAbyssJewel then
			timelessData.socketFilterDistance = timelessData.socketFilterDistance or 0
			for nodeId in pairs(radiusNodes) do
				allocatedNodes[nodeId] = self.build.calcsTab.mainEnv.grantedPassives[nodeId] ~= nil or self.build.spec.allocNodes[nodeId] ~= nil
				if timelessData.socketFilterDistance > 0 then
					unAllocatedNodesDistance[nodeId] = self.build.spec.nodes[nodeId].pathDist or 1000
				end
			end
		end
		
		if not isAbyssJewel then
			for nodeId in pairs(radiusNodes) do
				if not rootNodes[nodeId]
				and not treeData.nodes[nodeId].isJewelSocket
				and not treeData.nodes[nodeId].isKeystone
				and (not controls.socketFilter.state or allocatedNodes[nodeId] or (timelessData.socketFilterDistance > 0 and unAllocatedNodesDistance[nodeId] <= timelessData.socketFilterDistance)) then
					if (treeData.nodes[nodeId].isNotable or timelessData.jewelType.id == 1) then
						targetNodes[nodeId] = true
					elseif desiredNodes["totalStat"] and not treeData.nodes[nodeId].isNotable then
						if isValueInArray({ "Strength", "Intelligence", "Dexterity" }, treeData.nodes[nodeId].dn) then
							targetSmallNodes.attributeSmalls = targetSmallNodes.attributeSmalls + 1
						else
							targetSmallNodes.otherSmalls = targetSmallNodes.otherSmalls + 1
						end
					end
				end
			end
		end

		local seedWeights = { }
		local seedMultiplier = timelessData.jewelType.id == 5 and 20 or 1 -- Elegant Hubris
		for curSeed = data.timelessJewelSeedMin[timelessData.jewelType.id] * seedMultiplier, data.timelessJewelSeedMax[timelessData.jewelType.id] * seedMultiplier, seedMultiplier do
			seedWeights[curSeed] = 0
			resultNodes[curSeed] = { }
			if isAbyssJewel then
				-- Abyss files name the affected passives directly, so score those entries
				-- instead of checking every passive in a jewel radius.
				for targetNode, modification in pairs(data.readAbyssJewelLUT(curSeed, socketId, timelessData.jewelType.id, zorathPath, self.build.spec.curAscendClassName)) do
					local treeNode = treeData.nodes[targetNode]
					local scoreNode = treeNode and not rootNodes[targetNode] and not treeNode.isJewelSocket and not treeNode.isKeystone
					if scoreNode and treeNode.ascendancyName and isValueInTable(protectedNodes, treeNode.dn) then
						resultNodes[curSeed] = nil
						break
					end
					if scoreNode and controls.socketFilter.state and not zorathPath then
						timelessData.socketFilterDistance = timelessData.socketFilterDistance or 0
						local allocated = self.build.calcsTab.mainEnv.grantedPassives[targetNode] ~= nil or self.build.spec.allocNodes[targetNode] ~= nil
						local distance = self.build.spec.nodes[targetNode].pathDist or 1000
						scoreNode = allocated or timelessData.socketFilterDistance > 0 and distance <= timelessData.socketFilterDistance
					end
					if scoreNode then
						for _, component in ipairs(modification) do
							local changedNode = data.resolveAbyssJewelComponent(component, treeData.legion)
							local changedNodeId = changedNode and changedNode.id
							local desiredNode = desiredNodes[changedNodeId]
							if desiredNode then
								local _, statMod1, roll1 = data.getAbyssJewelComponentRoll(component, changedNode, 1)
								local _, statMod2, roll2 = data.getAbyssJewelComponentRoll(component, changedNode, 2)
								local weight = desiredNode.nodeWeight * (roll1 or 1)
								if statMod2 then
									weight = weight + desiredNode.nodeWeight2 * (roll2 or 1)
								end
								resultNodes[curSeed][changedNodeId] = resultNodes[curSeed][changedNodeId] or { targetNodeNames = { }, totalWeight = 0 }
								resultNodes[curSeed][changedNodeId].totalWeight = resultNodes[curSeed][changedNodeId].totalWeight + weight
								t_insert(resultNodes[curSeed][changedNodeId], targetNode)
								t_insert(resultNodes[curSeed][changedNodeId].targetNodeNames, treeNode.name)
								seedWeights[curSeed] = seedWeights[curSeed] + weight
							end
						end
					end
				end
			end
			-- This list is empty for Abyss jewels because they were scored above.
			for targetNode in pairs(targetNodes) do
				local jewelDataTbl = data.readLUT(curSeed, targetNode, timelessData.jewelType.id)
				if not next(jewelDataTbl) then
					ConPrintf("Missing LUT: " .. timelessData.jewelType.label)
				else
					local curNode = nil
					local curNodeId = nil
					if (timelessData.jewelType.id == 4 and isValueInTable(protectedNodes, treeData.nodes[targetNode].dn)) then -- protected
						if jewelDataTbl[1] >= data.timelessJewelAdditions then -- protected node is a replacement, invalidate seed
							resultNodes[curSeed] = nil
							break
						end
						if not desiredNodes["totalStat"] then -- only add if user has not entered their own Devotion to the table
							desiredNodes["totalStat"] = {
								nodeWeight = 0.1, -- keeps total score low to let desired stats decide sort
								nodeWeight2 = 0,
								displayName = "Devotion",
								desiredIdx = desiredIdx + 1
							}
						end
						curNodeId = "totalStat"
					end
					if jewelDataTbl[1] >= data.timelessJewelAdditions and not isValueInTable(protectedNodes, treeData.nodes[targetNode].dn) then -- replace
						curNode = legionNodes[jewelDataTbl[1] + 1 - data.timelessJewelAdditions]
						curNodeId = curNode and legionNodes[jewelDataTbl[1] + 1 - data.timelessJewelAdditions].id or nil
					else -- add
						curNode = legionAdditions[jewelDataTbl[1] + 1]
						curNodeId = curNode and legionAdditions[jewelDataTbl[1] + 1].id or nil
					end
					if desiredNodes["totalStat"] and reverseTotalModIDs[curNodeId] then
						curNodeId = "totalStat"
					end
					if timelessData.jewelType.id == 1 then
						local headerSize = #jewelDataTbl
						if headerSize == 2 or headerSize == 3 then
							if desiredNodes[curNodeId] then
								resultNodes[curSeed][curNodeId] = resultNodes[curSeed][curNodeId] or { targetNodeNames = { }, totalWeight = 0 }
								local statMod1 = curNode.stats[curNode.sortedStats[1]]
								local weight = desiredNodes[curNodeId].nodeWeight * jewelDataTbl[statMod1.index + 1]
								local statMod2 = curNode.stats[curNode.sortedStats[2]]
								if statMod2 then
									weight = weight + desiredNodes[curNodeId].nodeWeight2 * jewelDataTbl[statMod2.index + 1]
								end
								t_insert(resultNodes[curSeed][curNodeId], targetNode)
								t_insert(resultNodes[curSeed][curNodeId].targetNodeNames, treeData.nodes[targetNode].name)
								resultNodes[curSeed][curNodeId].totalWeight = resultNodes[curSeed][curNodeId].totalWeight + weight
								seedWeights[curSeed] = seedWeights[curSeed] + weight
							end
						elseif headerSize == 6 or headerSize == 8 then
							for i, jewelData in ipairs(jewelDataTbl) do
								curNode = legionAdditions[jewelDataTbl[i] + 1]
								curNodeId = curNode and legionAdditions[jewelDataTbl[i] + 1].id or nil
								if i <= (headerSize / 2) then
									if desiredNodes[curNodeId] then
										resultNodes[curSeed][curNodeId] = resultNodes[curSeed][curNodeId] or { targetNodeNames = { }, totalWeight = 0 }
										local weight = desiredNodes[curNodeId].nodeWeight * jewelDataTbl[i + (headerSize / 2)]
										resultNodes[curSeed][curNodeId].totalWeight = resultNodes[curSeed][curNodeId].totalWeight + weight
										t_insert(resultNodes[curSeed][curNodeId], targetNode)
										t_insert(resultNodes[curSeed][curNodeId].targetNodeNames, treeData.nodes[targetNode].name)
										seedWeights[curSeed] = seedWeights[curSeed] + weight
									end
								else
									break
								end
							end
						end
					elseif desiredNodes[curNodeId] then
						resultNodes[curSeed][curNodeId] = resultNodes[curSeed][curNodeId] or { targetNodeNames = { }, totalWeight = 0 }
						resultNodes[curSeed][curNodeId].totalWeight = resultNodes[curSeed][curNodeId].totalWeight + desiredNodes[curNodeId].nodeWeight
						t_insert(resultNodes[curSeed][curNodeId], targetNode)
						t_insert(resultNodes[curSeed][curNodeId].targetNodeNames, treeData.nodes[targetNode].name)
						seedWeights[curSeed] = seedWeights[curSeed] + desiredNodes[curNodeId].nodeWeight
					end
				end
			end
			if resultNodes[curSeed] and desiredNodes["totalStat"] then
				resultNodes[curSeed]["totalStat"] = resultNodes[curSeed]["totalStat"] or { targetNodeNames = { }, totalWeight = 0 }
				if timelessData.jewelType.id == 4 then -- Militant Faith
					local addedWeight = desiredNodes["totalStat"].nodeWeight * (5 * targetSmallNodes.otherSmalls + 10 * targetSmallNodes.attributeSmalls)
					addedWeight = addedWeight + resultNodes[curSeed]["totalStat"].totalWeight * 4
					resultNodes[curSeed]["totalStat"].totalWeight = resultNodes[curSeed]["totalStat"].totalWeight + addedWeight
					seedWeights[curSeed] = seedWeights[curSeed] + addedWeight
				else
					local addedWeight = desiredNodes["totalStat"].nodeWeight * (4 * targetSmallNodes.otherSmalls + 2 * targetSmallNodes.attributeSmalls)
					addedWeight = addedWeight + resultNodes[curSeed]["totalStat"].totalWeight * 19
					resultNodes[curSeed]["totalStat"].totalWeight = resultNodes[curSeed]["totalStat"].totalWeight + addedWeight
					seedWeights[curSeed] = seedWeights[curSeed] + addedWeight
				end
			end
			if resultNodes[curSeed] then
				-- check minimum weights
				for _, val in ipairs(minimumWeights) do
					if (resultNodes[curSeed][val.reqNode] and resultNodes[curSeed][val.reqNode].totalWeight or 0) < val.weight then
						resultNodes[curSeed] = nil
						break
					end
				end
			end
		end
		
		return {
			resultNodes = resultNodes,
			seedWeights = seedWeights,
			desiredNodes = desiredNodes,
			socketInfo = socketInfo
		}
	end

	local function formatSearchValue(input)
		local   matchPattern1 = " 0"
		local replacePattern1 = "   "
		local   matchPattern2 = ".0 "
		local replacePattern2 = "    "
		local   matchPattern3 = "  %."
		local replacePattern3 = "0."
		local   matchPattern4 = "%.([0-9])0"
		local replacePattern4 = ".%1  "
		return (" " .. s_format("%006.2f", input))
		:gsub(matchPattern1, replacePattern1):gsub(matchPattern1, replacePattern1)
		:gsub(matchPattern2, replacePattern2):gsub(matchPattern2, replacePattern2)
		:gsub(matchPattern3, replacePattern3)
		:gsub(matchPattern4, replacePattern4)
	end

	local function formatResults(resultNodes, seedWeights, desiredNodes, socketInfo)
		local results = { }
		for seedMatch, seedData in pairs(resultNodes) do
			-- filter out the results so that only the ones that beat the total minimum weight parameter remain in search results
			local passesMin = (not timelessData.totalMinimumWeight) or (seedWeights[seedMatch] >= timelessData.totalMinimumWeight)
			if seedWeights[seedMatch] > 0 and passesMin then
				local labelPrefix = socketInfo and (socketInfo.label .. " | ") or ""
				local result = { 
					label = labelPrefix .. seedMatch .. ":",
					seed = seedMatch,
					total = seedWeights[seedMatch]
				}
				if socketInfo then
					result.socketId = socketInfo.id
					result.socketLabel = socketInfo.label
				end
				if timelessData.jewelType.id == 1 or timelessData.jewelType.id == 3 or timelessData.jewelType.id >= 7 then
					-- These jewel types all use seeds below 10000.
					if seedMatch < 1000 then
						result.label = "  " .. result.label
					end
				elseif timelessData.jewelType.id == 4 then
					-- Militant Faith [2000-10000]
					if seedMatch < 10000 then
						result.label = "  " .. result.label
					end
				else
					-- Elegant Hubris [2000-160000]
					if seedMatch < 10000 then
						result.label = "    " .. result.label
					elseif seedMatch < 100000 then
						result.label = "  " .. result.label
					end
				end
				local sortedNodeArray = { }
				for legionId, desiredNode in pairs(desiredNodes) do
					if seedData[legionId] then
						if desiredNode.desiredIdx == 8 then
							sortedNodeArray[8] = " ..."
						elseif desiredNode.desiredIdx < 8 then
							sortedNodeArray[desiredNode.desiredIdx] = formatSearchValue(seedData[legionId].totalWeight)
						end
						result[legionId] = result[legionId] or { }
						result[legionId].targetNodeNames = seedData[legionId].targetNodeNames
					elseif desiredNode.desiredIdx < 8 then
						sortedNodeArray[desiredNode.desiredIdx] = "     0     "
					end
				end
				result.label = result.label .. t_concat(sortedNodeArray)
				t_insert(results, result)
			end
		end
		return results
	end

	controls.resetButton = { onClick = function()
		updateSearchList("", true)
		updateSearchList("", false)
		controls.abyssAscendancySelect.selIndex = 1
		wipeTable(timelessData.searchResults)
		clearProtected()
	end }
	controls.searchButton = { onClick = function()
		if timelessData.jewelSocket.id == -1 then
			wipeTable(timelessData.searchResults)
			wipeTable(timelessData.sharedResults)
			timelessData.sharedResults.type = timelessData.jewelType
			timelessData.sharedResults.conqueror = timelessData.conquerorType
			timelessData.sharedResults.devotionVariant1 = devotionVariants[timelessData.devotionVariant1]
			timelessData.sharedResults.devotionVariant2 = devotionVariants[timelessData.devotionVariant2]
			timelessData.sharedResults.multiSocket = true
			
			for socketIdx = 2, #jewelSockets do
				local currentSocket = jewelSockets[socketIdx]
				local searchResult = searchSingleSocket(currentSocket.id, currentSocket)
				if searchResult then
					local resultNodes = searchResult.resultNodes
					local seedWeights = searchResult.seedWeights
					local desiredNodes = searchResult.desiredNodes
					
					timelessData.sharedResults.desiredNodes = desiredNodes
					
					for _, result in ipairs(formatResults(resultNodes, seedWeights, desiredNodes, currentSocket)) do
						timelessData.searchResults[#timelessData.searchResults + 1] = result
					end
				end
			end
			
			t_sort(timelessData.searchResults, function(a, b) 
				if a.total == b.total then
					return (a.socketLabel or "") < (b.socketLabel or "")
				end
				return a.total > b.total 
			end)
			
			controls.searchResults.highlightIndex = nil
			controls.searchResults.selIndex = 1
		else
			local searchResult = searchSingleSocket(timelessData.jewelSocket.id, timelessData.jewelSocket)
			if searchResult then
				local resultNodes = searchResult.resultNodes
				local seedWeights = searchResult.seedWeights
				local desiredNodes = searchResult.desiredNodes
				
				wipeTable(timelessData.searchResults)
				wipeTable(timelessData.sharedResults)
				timelessData.sharedResults.type = timelessData.jewelType
				timelessData.sharedResults.conqueror = timelessData.conquerorType
				timelessData.sharedResults.devotionVariant1 = devotionVariants[timelessData.devotionVariant1]
				timelessData.sharedResults.devotionVariant2 = devotionVariants[timelessData.devotionVariant2]
				timelessData.sharedResults.socket = timelessData.jewelSocket
				timelessData.sharedResults.desiredNodes = desiredNodes
				
				for _, result in ipairs(formatResults(resultNodes, seedWeights, desiredNodes, nil)) do
					timelessData.searchResults[#timelessData.searchResults + 1] = result
				end
				t_sort(timelessData.searchResults, function(a, b) return a.total > b.total end)
				controls.searchResults.highlightIndex = nil
				controls.searchResults.selIndex = 1
			end
		end
	end }

	self.timelessSearchControls = controls
	main:OpenPopup(nil, nil, "Find a Timeless Jewel", controls)
	return controls
end
