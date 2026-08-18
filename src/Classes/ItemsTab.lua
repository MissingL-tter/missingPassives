-- Path of Building
--
-- Module: Items Tab
-- Item and equipment state for the current build.
--
local pairs = pairs
local ipairs = ipairs
local next = next
local t_insert = table.insert
local t_remove = table.remove
local s_format = string.format
local m_max = math.max
local m_min = math.min
local m_ceil = math.ceil
local m_floor = math.floor
local m_modf = math.modf

local gemTooltip = LoadModule("Classes/GemTooltip")

local baseSlots = { "Weapon 1", "Weapon 2", "Helmet", "Body Armour", "Gloves", "Boots", "Amulet", "Ring 1", "Ring 2", "Ring 3", "Belt", "Graft 1", "Graft 2", "Flask 1", "Flask 2", "Flask 3", "Flask 4", "Flask 5" }

local influenceInfo = itemLib.influenceInfo.all

local catalystQualityFormat = {
	"^x7F7F7FQuality (Attack Modifiers): "..colorCodes.MAGIC.."+%d%% (augmented)",
	"^x7F7F7FQuality (Speed Modifiers): "..colorCodes.MAGIC.."+%d%% (augmented)",
	"^x7F7F7FQuality (Suffix Modifiers): "..colorCodes.MAGIC.."+%d%% (augmented)",
	"^x7F7F7FQuality (Life and Mana Modifiers): "..colorCodes.MAGIC.."+%d%% (augmented)",
	"^x7F7F7FQuality (Caster Modifiers): "..colorCodes.MAGIC.."+%d%% (augmented)",
	"^x7F7F7FQuality (Attribute Modifiers): "..colorCodes.MAGIC.."+%d%% (augmented)",
	"^x7F7F7FQuality (Physical and Chaos Modifiers): "..colorCodes.MAGIC.."+%d%% (augmented)",
	"^x7F7F7FQuality (Resistance Modifiers): "..colorCodes.MAGIC.."+%d%% (augmented)",
	"^x7F7F7FQuality (Prefix Modifiers): "..colorCodes.MAGIC.."+%d%% (augmented)",
	"^x7F7F7FQuality (Defence Modifiers): "..colorCodes.MAGIC.."+%d%% (augmented)",
	"^x7F7F7FQuality (Elemental Modifiers): "..colorCodes.MAGIC.."+%d%% (augmented)",
	"^x7F7F7FQuality (Critical Modifiers): "..colorCodes.MAGIC.."+%d%% (augmented)",
}

local flavourLookup = {}

for _, entry in pairs(data.flavourText) do
	if entry.name and entry.id and entry.text then
		flavourLookup[entry.name] = flavourLookup[entry.name] or {}
		flavourLookup[entry.name][entry.id] = entry.text
	end
end

local function isAnointable(item)
	return item and item.base and not item.base.cannotBeAnointed
	    and item.base.subType ~= "Talisman"
		and (item.canBeAnointed or item.base.type == "Amulet")
end

local ItemsTabClass = newClass("ItemsTab", "UndoHandler", function(self, build)
	self.UndoHandler()

	self.build = build

	self.items = { }
	self.itemOrderList = { }

	self.showStatDifferences = true

	-- Persisted trade search weights and the request handler for trade site queries;
	-- statSortSelectionList stays nil until loaded or assigned, so fresh builds
	-- don't save an empty TradeSearchWeights section
	self.tradeQuery = { }
	self.tradeQuery.tradeQueryRequests = new("TradeQueryRequests", self.tradeQuery)

	-- Selection models for the item being viewed/edited; these replace the old
	-- crafting controls and are driven both by the engine and by a UI
	self.controls = { }

	-- Item slots
	self.slots = { }
	self.orderedSlots = { }
	self.slotOrder = { }
	local function addSlot(slot)
		self.slots[slot.slotName] = slot
		t_insert(self.orderedSlots, slot)
		self.slotOrder[slot.slotName] = #self.orderedSlots
	end
	for index, slotName in ipairs(baseSlots) do
		local slot = new("ItemSlot", self, slotName)
		addSlot(slot)
		if slotName:match("Weapon") then
			-- Add alternate weapon slot
			slot.weaponSet = 1
			local swapSlot = new("ItemSlot", self, slotName.." Swap", slotName)
			addSlot(swapSlot)
			swapSlot.weaponSet = 2
			for i = 1, 6 do
				local abyssal = new("ItemSlot", self, slotName.." Swap Abyssal Socket "..i, "Abyssal #"..i)
				addSlot(abyssal)
				abyssal.parentSlot = swapSlot
				abyssal.weaponSet = 2
				swapSlot.abyssalSocketList[i] = abyssal
			end
		end
		if slotName == "Weapon 1" or slotName == "Weapon 2" or slotName == "Helmet" or slotName == "Gloves" or slotName == "Body Armour" or slotName == "Boots" or slotName == "Belt" then
			-- Add Abyssal Socket slots
			for i = 1, 6 do
				local abyssal = new("ItemSlot", self, slotName.." Abyssal Socket "..i, "Abyssal #"..i)
				addSlot(abyssal)
				abyssal.parentSlot = slot
				if slotName:match("Weapon") then
					abyssal.weaponSet = 1
				end
				slot.abyssalSocketList[i] = abyssal
			end
		end
	end

	-- Jewel sockets
	self.sockets = { }
	local socketOrder = { }
	for _, node in pairs(build.latestTree.nodes) do
		if node.type == "Socket" then
			t_insert(socketOrder, node)
		end
	end
	table.sort(socketOrder, function(a, b)
		return a.id < b.id
	end)
	for _, node in ipairs(socketOrder) do
		local socket = new("ItemSlot", self, "Jewel "..node.id, "Socket", node.id)
		self.sockets[node.id] = socket
		addSlot(socket)
	end

	-- Display item description
	self.displayItemTooltip = new("Tooltip")

	-- Version/variant selection models
	self.controls.displayItemVersion = new("Selector", nil, function(index, value)
		self.displayItem.selectedVersion = index
		self.displayItem:NormaliseVariantSelections()
		self.displayItem:BuildAndParseRaw()
		self:UpdateDisplayItemVariantControls()
		self:UpdateDisplayItemTooltip()
		self:UpdateDisplayItemRangeLines()
	end)
	self.controls.displayItemVersion.shown = function()
		return self.displayItem and self.displayItem.usesVariantGroups
			and self.displayItem.versionList and #self.displayItem.versionList > 1
	end
	local variantSelectors = {
		{ name = "displayItemVariant", legacyField = "variant", legacyShown = function() return self.displayItem.variantList and #self.displayItem.variantList > 1 end },
		{ name = "displayItemAltVariant", legacyField = "variantAlt", legacyShown = function() return self.displayItem.hasAltVariant end },
		{ name = "displayItemAltVariant2", legacyField = "variantAlt2", legacyShown = function() return self.displayItem.hasAltVariant2 end },
		{ name = "displayItemAltVariant3", legacyField = "variantAlt3", legacyShown = function() return self.displayItem.hasAltVariant3 end },
		{ name = "displayItemAltVariant4", legacyField = "variantAlt4", legacyShown = function() return self.displayItem.hasAltVariant4 end },
		{ name = "displayItemAltVariant5", legacyField = "variantAlt5", legacyShown = function() return self.displayItem.hasAltVariant5 end },
	}
	for _, selectorDef in ipairs(variantSelectors) do
		local control
		control = new("Selector", nil, function(index, value)
			self:SelectDisplayItemVariant(index, value, selectorDef.legacyField, control)
		end)
		control.shown = function()
			return self.displayItem and (self.displayItem.usesVariantGroups and control.newVariantVisible
				or not self.displayItem.usesVariantGroups and selectorDef.legacyShown())
		end
		control.enabled = function()
			return not self.displayItem or not self.displayItem.usesVariantGroups or control.newVariantEnabled
		end
		self.controls[selectorDef.name] = control
	end

	-- Cluster jewel crafting models
	self.controls.displayItemClusterJewelSkill = new("Selector", { }, function(index, value)
		self.displayItem.clusterJewelSkill = value.skillId
		self:CraftClusterJewel()
	end)
	self.controls.displayItemClusterJewelSkill.shown = function()
		return self.displayItem and self.displayItem.crafted and self.displayItem.clusterJewel and true or false
	end
	self.controls.displayItemClusterJewelNodeCount = new("SliderModel", function(val)
		local clusterJewel = self.displayItem.clusterJewel
		self.displayItem.clusterJewelNodeCount = round(val * (clusterJewel.maxNodes - clusterJewel.minNodes) + clusterJewel.minNodes)
		self:CraftClusterJewel()
	end)

	-- Modifier sorting model for crafted items
	local sortingOptions = {
		{ stat = nil, label = "Default" }
	}
	for _, option in ipairs(data.powerStatList) do
		if not option.ignoreForItems and option.label ~= "Name" then
			table.insert(sortingOptions, option)
		end
	end
	self.controls.craftingSortingLabel = {
		shown = function()
			return self.displayItem and self.displayItem.crafted and
				-- cluster jewels don't have good comparison support and sorting would be misleading
				not (self.displayItem.base.type == "Jewel" and self.displayItem.base.subType == "Cluster")
		end
	}
	self.controls.craftingSorting = new("Selector", sortingOptions, function()
		self:UpdateAffixControls()
	end)

	-- Affix selection models for crafted items
	local maxModCount = 9
	self.controls.displayItemSectionAffix = new("Selector")
	self.controls.displayItemSectionAffix.shown = function()
		return self.displayItem and self.displayItem.crafted and true or false
	end
	for i = 1, maxModCount do
		local drop, slider
		drop = new("Selector", nil, function(index, value)
			local affix = { modId = "None", fractured = self.displayItem[drop.outputTable][drop.outputIndex].fractured }
			if value.modId then
				affix.modId = value.modId
				affix.range = slider.val
			elseif value.modList then
				slider.divCount = #value.modList
				local index, range = slider:GetDivVal()
				affix.modId = value.modList[index]
				affix.range = self:VerifyAffixRange(range, index, drop)
			end
			self.displayItem[drop.outputTable][drop.outputIndex] = affix
			self.displayItem:Craft()
			self:UpdateDisplayItemTooltip()
			self:UpdateAffixControls()
		end)
		drop.shown = function()
			return self.displayItem and self.displayItem.crafted and i <= self.displayItem.affixLimit and true or false
		end
		slider = new("SliderModel", function(val)
			local affix = self.displayItem[drop.outputTable][drop.outputIndex]
			local index, range = slider:GetDivVal()
			affix.modId = drop.list[drop.selIndex].modList[index]

			affix.range = self:VerifyAffixRange(range, index, drop)
			self.displayItem:Craft()
			self:UpdateDisplayItemTooltip()
		end)
		drop.slider = slider
		self.controls["displayItemAffix"..i] = drop
		self.controls["displayItemAffixRange"..i] = slider
	end

	-- Custom modifier models
	self.controls.displayItemSectionCustom = new("Selector")
	self.controls.displayItemAddCustom = new("Selector")
	self.controls.displayItemAddCustom.shown = function()
		return self.displayItem and (self.displayItem.rarity == "MAGIC" or self.displayItem.rarity == "RARE" or (self.displayItem.rareLikeUnique and self.displayItem.rareLikeUnique.supportsCustomModifiers)) and true or false
	end
	self.controls.displayItemAddCrucible = new("Selector")
	self.controls.displayItemAddCrucible.shown = function()
		return self.displayItem and (self.displayItem:GetPrimarySlot() == "Weapon 1" or self.displayItem.type == "Shield" or self.displayItem.canHaveShieldCrucibleTree) and true or false
	end

	-- Modifier range models
	self.controls.displayItemRangeLine = new("Selector", nil, function(index, value)
		self.controls.displayItemRangeSlider.val = self.displayItem.rangeLineList[index].range
	end)
	self.controls.displayItemRangeSlider = new("SliderModel", function(val)
		local line = self.displayItem.rangeLineList[self.controls.displayItemRangeLine.selIndex]
		line.range = val
		self.displayItem:BuildAndParseRaw()
		self:UpdateDisplayItemTooltip()
		self:UpdateCustomControls()
	end)
	self.controls.displayItemRangeLine.list = { }

	-- Initialise item sets
	self.itemSets = { }
	self.itemSetOrderList = { 1 }
	self:NewItemSet(1)
	self:SetActiveItemSet(1)

	self:PopulateSlots()
	self.lastSlot = self.slots[baseSlots[#baseSlots]]
end)

function ItemsTabClass:Load(xml, dbFileName)
	self.activeItemSetId = 0
	self.itemSets = { }
	self.itemSetOrderList = { }
	self.tradeQuery.statSortSelectionList = { }
	for _, node in ipairs(xml) do
		if node.elem == "Item" then
			local item = new("Item", "")
			item.id = tonumber(node.attrib.id)
			item.variant = tonumber(node.attrib.variant)
			if node.attrib.variantAlt then
				item.hasAltVariant = true
				item.variantAlt = tonumber(node.attrib.variantAlt)
			end
			if node.attrib.variantAlt2 then
				item.hasAltVariant2 = true
				item.variantAlt2 = tonumber(node.attrib.variantAlt2)
			end
			if node.attrib.variantAlt3 then
				item.hasAltVariant3 = true
				item.variantAlt3 = tonumber(node.attrib.variantAlt3)
			end
			if node.attrib.variantAlt4 then
				item.hasAltVariant4 = true
				item.variantAlt4 = tonumber(node.attrib.variantAlt4)
			end
			if node.attrib.variantAlt5 then
				item.hasAltVariant5 = true
				item.variantAlt5 = tonumber(node.attrib.variantAlt5)
			end
			for _, child in ipairs(node) do
				if type(child) == "string" then
					item:ParseRaw(child)
				elseif child.elem == "ModRange" then
					local id = tonumber(child.attrib.id) or 0
					local range = tonumber(child.attrib.range) or 1
					-- This is garbage, but needed due to change to separate mod line lists
					-- 'ModRange' elements are legacy though, so is this actually needed? :<
					-- Maybe it is? Maybe it isn't? Maybe up is down? Maybe good is bad? AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
					-- Sorry, cluster jewels are making me crazy(-ier)
					for _, list in ipairs{item.buffModLines, item.enchantModLines, item.scourgeModLines, item.implicitModLines, item.explicitModLines, item.crucibleModLines} do
						if id <= #list then
							list[id].range = range
							break
						end
						id = id - #list
					end
				end
			end
			if item.base then
				item:BuildModList()
				self.items[item.id] = item
				t_insert(self.itemOrderList, item.id)
			end
		-- Below is OBE and left for legacy compatibility (all Slots are part of ItemSets now)
		elseif node.elem == "Slot" then
			local slot = self.slots[node.attrib.name or ""]
			if slot then
				slot.selItemId = tonumber(node.attrib.itemId)
				if slot.slotName:match("Flask") then
					slot.active = node.attrib.active == "true"
				end
			end
		elseif node.elem == "ItemSet" then
			local itemSet = self:NewItemSet(tonumber(node.attrib.id))
			itemSet.title = node.attrib.title
			itemSet.useSecondWeaponSet = node.attrib.useSecondWeaponSet == "true"
			for _, child in ipairs(node) do
				if child.elem == "Slot" then
					local slotName = child.attrib.name or ""
					if itemSet[slotName] then
						itemSet[slotName].selItemId = tonumber(child.attrib.itemId)
						itemSet[slotName].active = child.attrib.active == "true"
						itemSet[slotName].pbURL = child.attrib.itemPbURL or ""
					end
				elseif child.elem == "SocketIdURL" then
					local id = tonumber(child.attrib.nodeId)
					itemSet[id] = { pbURL = child.attrib.itemPbURL or "" }
				end
			end
			t_insert(self.itemSetOrderList, itemSet.id)
		elseif node.elem == "TradeSearchWeights" then
			for _, child in ipairs(node) do
				local statSort = {
					label = child.attrib.label,
					stat = child.attrib.stat,
					weightMult = tonumber(child.attrib.weightMult)
				}
				for _, statEntry in ipairs(data.powerStatList) do
					if statSort.stat == statEntry.stat then
						-- update information which can be out of data or missing in the xml
						statSort.label = statEntry.label
						statSort.transform = statEntry.transform
						t_insert(self.tradeQuery.statSortSelectionList, statSort)
						break
					end
				end
			end
		end
	end
	if not self.itemSetOrderList[1] then
		self.activeItemSet = self:NewItemSet(1)
		self.activeItemSet.useSecondWeaponSet = xml.attrib.useSecondWeaponSet == "true"
		self.itemSetOrderList[1] = 1
	end
	self:SetActiveItemSet(tonumber(xml.attrib.activeItemSet) or 1)
	if xml.attrib.showStatDifferences then
		self.showStatDifferences = xml.attrib.showStatDifferences == "true"
	end
	self:ResetUndo()
end

function ItemsTabClass:Save(xml)
	xml.attrib = {
		activeItemSet = tostring(self.activeItemSetId),
		useSecondWeaponSet = tostring(self.activeItemSet.useSecondWeaponSet),
		showStatDifferences = tostring(self.showStatDifferences),
	}
	for _, id in ipairs(self.itemOrderList) do
		local item = self.items[id]
		local child = {
			elem = "Item",
			attrib = {
				id = tostring(id),
				variant = item.variant and tostring(item.variant),
				variantAlt = item.variantAlt and tostring(item.variantAlt),
				variantAlt2 = item.variantAlt2 and tostring(item.variantAlt2),
				variantAlt3 = item.variantAlt3 and tostring(item.variantAlt3),
				variantAlt4 = item.variantAlt4 and tostring(item.variantAlt4),
				variantAlt5 = item.variantAlt5 and tostring(item.variantAlt5)
			}
		}
		item:BuildAndParseRaw()
		t_insert(child, item.raw)
		local id = #item.buffModLines + 1
		for _, modLine in ipairs(item.enchantModLines) do
			if modLine.range then
				t_insert(child, { elem = "ModRange", attrib = { id = tostring(id), range = tostring(modLine.range) } })
			end
			id = id + 1
		end
		for _, modLine in ipairs(item.scourgeModLines) do
			if modLine.range then
				t_insert(child, { elem = "ModRange", attrib = { id = tostring(id), range = tostring(modLine.range) } })
			end
			id = id + 1
		end
		for _, modLine in ipairs(item.implicitModLines) do
			if modLine.range then
				t_insert(child, { elem = "ModRange", attrib = { id = tostring(id), range = tostring(modLine.range) } })
			end
			id = id + 1
		end
		for _, modLine in ipairs(item.explicitModLines) do
			if modLine.range then
				t_insert(child, { elem = "ModRange", attrib = { id = tostring(id), range = tostring(modLine.range) } })
			end
			id = id + 1
		end
		for _, modLine in ipairs(item.crucibleModLines) do
			if modLine.range then
				t_insert(child, { elem = "ModRange", attrib = { id = tostring(id), range = tostring(modLine.range) } })
			end
			id = id + 1
		end
		t_insert(xml, child)
	end
	for _, itemSetId in ipairs(self.itemSetOrderList) do
		local itemSet = self.itemSets[itemSetId]
		local child = { elem = "ItemSet", attrib = { id = tostring(itemSetId), title = itemSet.title, useSecondWeaponSet = tostring(itemSet.useSecondWeaponSet) } }
		for slotName, slot in pairs(self.slots) do
			if not slot.nodeId then
				t_insert(child, { elem = "Slot", attrib = { name = slotName, itemId = tostring(itemSet[slotName].selItemId), itemPbURL = itemSet[slotName].pbURL or "", active = itemSet[slotName].active and "true" }})
			else
				if self.build.spec.allocNodes[slot.nodeId] then
					t_insert(child, { elem = "SocketIdURL", attrib = { name = slotName, nodeId = tostring(slot.nodeId), itemPbURL = itemSet[slot.nodeId] and itemSet[slot.nodeId].pbURL or ""}})
				end
			end
		end
		t_insert(xml, child)
	end
	if self.tradeQuery.statSortSelectionList then
		local parent = {
			elem = "TradeSearchWeights"
		}
		for _, statSort in ipairs(self.tradeQuery.statSortSelectionList) do
			if statSort.weightMult and statSort.weightMult > 0 then
				local child = {
				elem = "Stat",
				attrib = {
					label = statSort.label,
					stat = statSort.stat,
					weightMult = s_format("%.2f", tostring(statSort.weightMult))
				}
			}
			t_insert(parent, child)
			end
		end
		t_insert(xml, parent)
	end
end

-- Creates a new item set
function ItemsTabClass:NewItemSet(itemSetId)
	local itemSet = { id = itemSetId }
	if not itemSetId then
		itemSet.id = 1
		while self.itemSets[itemSet.id] do
			itemSet.id = itemSet.id + 1
		end
	end
	for slotName, slot in pairs(self.slots) do
		if not slot.nodeId then
			itemSet[slotName] = { selItemId = 0 }
		end
	end
	self.itemSets[itemSet.id] = itemSet
	return itemSet
end

-- Changes the active item set
function ItemsTabClass:SetActiveItemSet(itemSetId)
	local prevSet = self.activeItemSet
	if not self.itemSets[itemSetId] then
		itemSetId = self.itemSetOrderList[1]
	end
	self.activeItemSetId = itemSetId
	self.activeItemSet = self.itemSets[itemSetId]
	local curSet = self.activeItemSet
	for slotName, slot in pairs(self.slots) do
		if not slot.nodeId then
			if prevSet then
				-- Update the previous set
				prevSet[slotName].selItemId = slot.selItemId
				prevSet[slotName].active = slot.active
			end
			-- Equip the incoming set's item
			slot.selItemId = curSet[slotName].selItemId
			slot.active = curSet[slotName].active
		end
	end
	self.build.buildFlag = true
	self:PopulateSlots()
	self.build:SyncLoadouts()
end

-- Equips the given item in the given item set
function ItemsTabClass:EquipItemInSet(item, itemSetId)
	local itemSet = self.itemSets[itemSetId]
	local slotName = item:GetPrimarySlot()
	if self.slots[slotName].weaponSet == 1 and itemSet.useSecondWeaponSet then
		-- Redirect to second weapon set
		slotName = slotName .. " Swap"
	end
	if not item.id or not self.items[item.id] then
		item = new("Item", item.raw)
		self:AddItem(item, true)
	end
	local altSlot = slotName:gsub("1","2")
	if IsKeyDown("SHIFT") then
		-- Redirect to second slot if possible
		if self:IsItemValidForSlot(item, altSlot, itemSet) then
			slotName = altSlot
		end
	end
	if itemSet == self.activeItemSet then
		self.slots[slotName]:SetSelItemId(item.id)
	else
		itemSet[slotName].selItemId = item.id
		if itemSet[altSlot].selItemId ~= 0 and not self:IsItemValidForSlot(self.items[itemSet[altSlot].selItemId], altSlot, itemSet) then
			itemSet[altSlot].selItemId = 0
		end
	end
	self:PopulateSlots()
	self:AddUndoState()
	self.build.buildFlag = true
end

-- Update the item lists for all the slot controls
function ItemsTabClass:PopulateSlots()
	for _, slot in pairs(self.slots) do
		slot:Populate()
	end
end

-- Updates the status and position of the socket controls
function ItemsTabClass:UpdateSockets()
	-- Build a list of active sockets
	local activeSocketList = { }
	for nodeId, slot in pairs(self.sockets) do
		if self.build.spec.allocNodes[nodeId] then
			t_insert(activeSocketList, nodeId)
			slot.inactive = false
		else
			slot.inactive = true
		end
	end
	table.sort(activeSocketList)

	-- Update the state of the active socket controls
	self.lastSlot = self.slots[baseSlots[#baseSlots]]
	for index, nodeId in ipairs(activeSocketList) do
		self.sockets[nodeId].label = "Socket #"..index
		self.lastSlot = self.sockets[nodeId]
	end
end

-- Returns the slot control and equipped jewel for the given node ID
function ItemsTabClass:GetSocketAndJewelForNodeID(nodeId)
	return self.sockets[nodeId], self.items[self.sockets[nodeId].selItemId]
end

-- Adds the given item to the build's item list
function ItemsTabClass:AddItem(item, noAutoEquip, index)
	if not item.id then
		-- Find an unused item ID
		item.id = 1
		while self.items[item.id] do
			item.id = item.id + 1
		end

		if index then
			t_insert(self.itemOrderList, index, item.id)
		else
			-- Add it to the end of the display order list
			t_insert(self.itemOrderList, item.id)
		end

		if not noAutoEquip then
			-- Autoequip it
			for _, slot in ipairs(self.orderedSlots) do
				if not slot.nodeId and slot.selItemId == 0 and slot:IsShown() and self:IsItemValidForSlot(item, slot.slotName) then
					slot:SetSelItemId(item.id)
					break
				end
			end
		end
	end

	-- Add it to the list
	local replacing = self.items[item.id]
	self.items[item.id] = item
	item:BuildModList()

	if replacing and (replacing.clusterJewel or item.clusterJewel or replacing.baseName == "Timeless Jewel") then
		-- We're replacing an existing item, and either the new or old one is a cluster jewel
		if isValueInTable(self.build.spec.jewels, item.id) then
			-- Item is currently equipped, so we need to rebuild the graphs
			self.build.spec:BuildClusterJewelGraphs()
		end
	end
end

-- Adds the current display item to the build's item list
function ItemsTabClass:AddDisplayItem(noAutoEquip)
	-- Add it to the list and clear the current display item
	self:AddItem(self.displayItem, noAutoEquip)
	self:SetDisplayItem()

	self:PopulateSlots()
	self:AddUndoState()
	self.build.buildFlag = true
end

-- Sorts the build's item list
function ItemsTabClass:SortItemList()
	table.sort(self.itemOrderList, function(a, b)
		local itemA = self.items[a]
		local itemB = self.items[b]
		local primSlotA = itemA:GetPrimarySlot()
		local primSlotB = itemB:GetPrimarySlot()
		if primSlotA ~= primSlotB then
			if not self.slotOrder[primSlotA] then
				return false
			elseif not self.slotOrder[primSlotB] then
				return true
			end
			return self.slotOrder[primSlotA] < self.slotOrder[primSlotB]
		end
		local equipSlotA, equipSetA = self:GetEquippedSlotForItem(itemA)
		local equipSlotB, equipSetB = self:GetEquippedSlotForItem(itemB)
		if equipSlotA and equipSlotB then
			if equipSlotA ~= equipSlotB then
				return self.slotOrder[equipSlotA.slotName] < self.slotOrder[equipSlotB.slotName]
			elseif equipSetA and not equipSetB then
				return false
			elseif not equipSetA and equipSetB then
				return true
			elseif equipSetA and equipSetB then
				return isValueInArray(self.itemSetOrderList, equipSetA.id) < isValueInArray(self.itemSetOrderList, equipSetB.id)
			end
		elseif equipSlotA then
			return true
		elseif equipSlotB then
			return false
		end
		return itemA.name < itemB.name
	end)
	self:AddUndoState()
end

-- Deletes an item
function ItemsTabClass:DeleteItem(item, deferUndoState)
	for slotName, slot in pairs(self.slots) do
		if slot.selItemId == item.id then
			slot:SetSelItemId(0)
			self.build.buildFlag = true
		end
		if not slot.nodeId then
			for _, itemSet in pairs(self.itemSets) do
				if itemSet[slotName].selItemId == item.id then
					itemSet[slotName].selItemId = 0
					self.build.buildFlag = true
				end
			end
		end
	end
	for index, id in pairs(self.itemOrderList) do
		if id == item.id then
			t_remove(self.itemOrderList, index)
			break
		end
	end
	for _, spec in pairs(self.build.treeTab.specList) do
		local rebuildClusterJewelGraphs = false
		for nodeId, itemId in pairs(spec.jewels) do
			if itemId == item.id then
				spec.jewels[nodeId] = 0
				rebuildClusterJewelGraphs = true
				-- Deallocate all nodes that required this jewel
				if spec.nodes[nodeId] then
					for depNodeId, depNode in ipairs(spec.nodes[nodeId].depends) do
						depNode.alloc = false
						spec.allocNodes[depNodeId] = nil
					end
					spec.nodes[nodeId].alloc = false
					spec.allocNodes[nodeId] = nil
				end
			end
		end
		if rebuildClusterJewelGraphs and not deferUndoState then
			spec:BuildClusterJewelGraphs()
		end
	end
	self.items[item.id] = nil
	if not deferUndoState then
		self:PopulateSlots()
		self:AddUndoState()
	end
end

function ItemsTabClass:CopyAnointsAndEldritchImplicits(newItem, copyEldritchImplicits, overwrite, sourceSlotName)
	local newItemType = sourceSlotName or (newItem.base.weapon and "Weapon 1" or newItem.base.type)
	if self.activeItemSet[newItemType] then
		local currentItem = self.activeItemSet[newItemType].selItemId and self.items[self.activeItemSet[newItemType].selItemId]
		-- if you don't have an equipped item that matches the type of the newItem, no need to do anything
		if currentItem then
			-- if the new item is anointable and does not have an anoint and your current respective item does, apply that anoint to the new item
			if isAnointable(newItem) and (#newItem.enchantModLines == 0 or overwrite) and self.activeItemSet[newItemType].selItemId > 0 then
				local currentAnoint = currentItem.enchantModLines
				if currentAnoint and #currentAnoint == 1 then -- skip if amulet has more than one anoint e.g. Stranglegasp
					newItem.enchantModLines = currentAnoint
				end
			end
			-- if the new item is a non-corrupted Normal, Magic, or Rare Helmet, Body Armour, Gloves, or Boots and does not have any influence
			-- and your current respective item is Eater and/or Exarch, apply those implicits and influence to the new item
			local eldritchBaseTypes = { "Helmet", "Body Armour", "Gloves", "Boots" }
			local eldritchRarities = { "NORMAL", "MAGIC", "RARE" }
			for _, influence in ipairs(itemLib.influenceInfo.default) do
				if newItem[influence.key] then
					return
				end
			end

			local modifiableItem = not (newItem.corrupted or newItem.mirrored)
			if copyEldritchImplicits and isValueInTable(eldritchBaseTypes, newItem.base.type) and isValueInTable(eldritchRarities, newItem.rarity)
				and (#newItem.implicitModLines == 0 or overwrite) and modifiableItem and (currentItem.cleansing or currentItem.tangle) and currentItem.implicitModLines then
					newItem.implicitModLines = currentItem.implicitModLines
					newItem.tangle = currentItem.tangle
					newItem.cleansing = currentItem.cleansing
			end

			-- harvest and heist enchantments on modifiable body armour or weapons
			if (newItem.base.weapon or newItem.base.type == "Body Armour") and (#newItem.enchantModLines == 0 or overwrite)	and self.activeItemSet[newItemType].selItemId > 0 and modifiableItem and currentItem.enchantModLines then
				newItem.enchantModLines = currentItem.enchantModLines
			end

			newItem:BuildAndParseRaw()
		end
	end
end

-- Attempt to create a new item from the given item raw text and sets it as the new display item
function ItemsTabClass:CreateDisplayItemFromRaw(itemRaw, normalise)
	local newItem = new("Item", itemRaw)
	if newItem.base then
		if normalise then
			self:CopyAnointsAndEldritchImplicits(newItem, main.migrateEldritchImplicits, false)
			newItem:NormaliseQuality()
			newItem:BuildModList()
		end
		self:SetDisplayItem(newItem)
	end
end

function ItemsTabClass:SelectDisplayItemVariant(index, value, legacyField, control)
	if self.displayItem.usesVariantGroups then
		if not value or not value.variantId then
			return
		end
		self.displayItem.variantGroupSelections[control.variantGroupId] = value.variantId
		self.displayItem:NormaliseVariantSelections()
	else
		self.displayItem[legacyField] = index
	end
	self.displayItem:BuildAndParseRaw()
	self:UpdateDisplayItemVariantControls()
	self:UpdateDisplayItemTooltip()
	self:UpdateDisplayItemRangeLines()
end

function ItemsTabClass:UpdateDisplayItemVariantControls()
	local item = self.displayItem
	local controls = {
		self.controls.displayItemVariant,
		self.controls.displayItemAltVariant,
		self.controls.displayItemAltVariant2,
		self.controls.displayItemAltVariant3,
		self.controls.displayItemAltVariant4,
		self.controls.displayItemAltVariant5,
	}
	for _, control in ipairs(controls) do
		control.newVariantVisible = false
	end
	if not item or not item.usesVariantGroups then
		return
	end

	self.controls.displayItemVersion.list = item.versionList or { }
	self.controls.displayItemVersion.selIndex = item.selectedVersion or 1
	local controlIndex = 1
	for groupId in pairsSortByKey(item.variantGroups) do
		local eligibleOptions = item:GetVariantGroupOptions(groupId, false)
		if #eligibleOptions > 0 then
			local control = controls[controlIndex]
			if not control then
				ConPrintf("Item '%s' has more than 6 variant groups", item.name)
				break
			end
			local availableOptions = item:GetVariantGroupOptions(groupId, true)
			local list = { }
			local selectedIndex
			for _, variantId in ipairs(availableOptions) do
				t_insert(list, {
					label = item.variantList[variantId],
					variantId = variantId,
				})
				if item.variantGroupSelections[groupId] == variantId then
					selectedIndex = #list
				end
			end
			if #list == 0 then
				t_insert(list, {
					label = "No available variants",
				})
			end
			control.list = list
			control.selIndex = selectedIndex or 1
			control.variantGroupId = groupId
			control.newVariantVisible = true
			control.newVariantEnabled = #availableOptions > 1
			controlIndex = controlIndex + 1
		end
	end
end

-- Sets the display item to the given item
function ItemsTabClass:SetDisplayItem(item)
	self.displayItem = item
	if item then
		-- Update the display item models
		self:UpdateDisplayItemTooltip()

		if item.usesVariantGroups then
			self:UpdateDisplayItemVariantControls()
		else
			self.controls.displayItemVariant.list = item.variantList
			self.controls.displayItemVariant.selIndex = item.variant
		end
		if not item.usesVariantGroups and item.hasAltVariant then
			self.controls.displayItemAltVariant.list = item.variantList
			self.controls.displayItemAltVariant.selIndex = item.variantAlt
		end
		if not item.usesVariantGroups and item.hasAltVariant2 then
			self.controls.displayItemAltVariant2.list = item.variantList
			self.controls.displayItemAltVariant2.selIndex = item.variantAlt2
		end
		if not item.usesVariantGroups and item.hasAltVariant3 then
			self.controls.displayItemAltVariant3.list = item.variantList
			self.controls.displayItemAltVariant3.selIndex = item.variantAlt3
		end
		if not item.usesVariantGroups and item.hasAltVariant4 then
			self.controls.displayItemAltVariant4.list = item.variantList
			self.controls.displayItemAltVariant4.selIndex = item.variantAlt4
		end
		if not item.usesVariantGroups and item.hasAltVariant5 then
			self.controls.displayItemAltVariant5.list = item.variantList
			self.controls.displayItemAltVariant5.selIndex = item.variantAlt5
		end
		if item.crafted then
			self:UpdateAffixControls()
		end

		self:UpdateCustomControls()
		self:UpdateDisplayItemRangeLines()
		if item.clusterJewel and item.crafted then
			self:UpdateClusterJewelControls()
		end
	end
end

function ItemsTabClass:UpdateDisplayItemTooltip()
	self.displayItemTooltip:Clear()
	self:AddItemTooltip(self.displayItemTooltip, self.displayItem)
	self.displayItemTooltip.center = true
end

function ItemsTabClass:ToggleDisplayItemModLine(modLine)
	if not self.displayItem or not modLine then
		return
	end
	modLine.disabled = not modLine.disabled
	self.displayItem:BuildAndParseRaw()
	self:UpdateDisplayItemTooltip()
	self:UpdateDisplayItemRangeLines()
	self:UpdateCustomControls()
	if self.displayItem.crafted then
		self:UpdateAffixControls()
	end
	self.build.buildFlag = true
end

function ItemsTabClass:UpdateClusterJewelControls()
	local item = self.displayItem

	local unavailableSkills = { ["affliction_strength"] = true, ["affliction_dexterity"] = true, ["affliction_intelligence"] = true, }

	-- Update list of skills
	local skillList = wipeTable(self.controls.displayItemClusterJewelSkill.list)
	for skillId, skill in pairs(item.clusterJewel.skills) do
		if not unavailableSkills[skillId] then
			t_insert(skillList, { label = skill.name, skillId = skillId })
		end
	end
	table.sort(skillList, function(a, b) return a.label < b.label end)
	if not item.clusterJewelSkill or not item.clusterJewel.skills[item.clusterJewelSkill] then
		item.clusterJewelSkill = skillList[1].skillId
	end
	self.controls.displayItemClusterJewelSkill:SelByValue(item.clusterJewelSkill, "skillId")

	-- Update added node count slider
	local countControl = self.controls.displayItemClusterJewelNodeCount
	item.clusterJewelNodeCount = m_min(m_max(item.clusterJewelNodeCount or item.clusterJewel.minNodes, item.clusterJewel.minNodes), item.clusterJewel.maxNodes)
	countControl.divCount = item.clusterJewel.maxNodes - item.clusterJewel.minNodes
	countControl.val = (item.clusterJewelNodeCount - item.clusterJewel.minNodes) / (item.clusterJewel.maxNodes - item.clusterJewel.minNodes)

	self:CraftClusterJewel()
end

function ItemsTabClass:CraftClusterJewel()
	local item = self.displayItem
	wipeTable(item.enchantModLines)
	t_insert(item.enchantModLines, { line = "Adds "..(item.clusterJewelNodeCount or item.clusterJewel.minNodes).." Passive Skills", crafted = true })
	if item.clusterJewel.size == "Large" then
		t_insert(item.enchantModLines, { line = "2 Added Passive Skills are Jewel Sockets", crafted = true })
	elseif item.clusterJewel.size == "Medium" then
		t_insert(item.enchantModLines, { line = "1 Added Passive Skill is a Jewel Socket", crafted = true })
	end
	local skill = item.clusterJewel.skills[item.clusterJewelSkill]
	t_insert(item.enchantModLines, { line = table.concat(skill.enchant, "\n"), crafted = true })
	item:BuildAndParseRaw()

	-- Update affixes manually to force out affixes that may now be invalid
	self:UpdateAffixControls()
	for i = 1, item.affixLimit do
		local drop = self.controls["displayItemAffix"..i]
		drop.selFunc(drop.selIndex, drop.list[drop.selIndex])
	end
end

-- Flips an affix range if it would form discontinuous values
function ItemsTabClass:VerifyAffixRange(range, index, drop)
	local priorMod = index - 1 > 0 and self.displayItem.affixes[drop.list[drop.selIndex].modList[index - 1]] or nil
	local nextMod = index + 1 < #drop.list[drop.selIndex].modList and self.displayItem.affixes[drop.list[drop.selIndex].modList[index + 1]] or nil
	local function flipRange(modA, modB) -- assumes all pairs are ordered the same
		local function getMinMax(mod) -- gets first valid range from a mod
			for _, line in ipairs(mod) do
				local min, max = line:match("%((%d[%d%.]*)%-(%d[%d%.]*)%)")
				if min and max then return tonumber(min), tonumber(max)	end
			end
		end

		local minA, maxA = getMinMax(modA)
		local minB, maxB = getMinMax(modB)

		if not minA or not minB or not maxA or not maxB then
			return false
		end

		local allInts = minA == m_floor(minA) and maxA == m_floor(maxA) and minB == m_floor(minB) and maxB == m_floor(maxB) -- if the mod goes in steps that aren't 1, then the code below this doesn't work
		if (minA and minB and maxA and maxB and allInts) then
			if (minA < minB) then -- ascending
				return minA + 1 == maxB
			else -- descending
				return minA - 1 == maxB
			end
		end
		return false
	end

	if priorMod then
		if flipRange(priorMod, self.displayItem.affixes[drop.list[drop.selIndex].modList[index]]) then
			range = 1 - range
		end
	elseif nextMod then
		if flipRange(self.displayItem.affixes[drop.list[drop.selIndex].modList[index]], nextMod) then
			range = 1 - range
		end
	end
	return range
end

-- Update affix selection controls
function ItemsTabClass:UpdateAffixControls()
	local item = self.displayItem
	local prefixLimit = item.prefixes.limit or (item.affixLimit / 2)
	local ignoreModType = item.rareLikeUnique and item.rareLikeUnique.ignoreModType
	local powerCache = {}
	for i = 1, item.affixLimit do
		if i <= prefixLimit then
			local modType = "Prefix"
			if ignoreModType then
				modType = nil
			end
			self:UpdateAffixControl(self.controls["displayItemAffix" .. i], item, modType, "prefixes", i, powerCache)
		else
			self:UpdateAffixControl(self.controls["displayItemAffix" .. i], item, "Suffix", "suffixes", i - prefixLimit, powerCache)
		end
	end
	-- The custom affixes may have had their indexes changed, so the custom control UI is also rebuilt so that it will
	-- reference the correct affix index.
	self:UpdateCustomControls()
end

function ItemsTabClass:UpdateAffixControl(control, item, affixType, outputTable, outputIndex, powerCache)
	local extraTags = { }
	local excludeGroups = { }
	local allowDuplicateGroups = item.rareLikeUnique and item.rareLikeUnique.allowDuplicateGroups
	for _, table in ipairs({"prefixes","suffixes"}) do
		for index = 1, (item[table].limit or (item.affixLimit / 2)) do
			if index ~= outputIndex or table ~= outputTable then
				local mod = item.affixes[item[table][index] and item[table][index].modId]
				if mod then
					if mod.group and not allowDuplicateGroups then
						excludeGroups[mod.group] = true
					end
					if mod.tags then
						for _, tag in ipairs(mod.tags) do
							extraTags[tag] = true
						end
					end
				end
			end
		end
	end
	if item.clusterJewel and item.clusterJewelSkill then
		local skill = item.clusterJewel.skills[item.clusterJewelSkill]
		if skill then
			extraTags[skill.tag] = true
		end
	end
	local selAffix = item[outputTable][outputIndex] and item[outputTable][outputIndex].modId
	local affixList = { }
	local retainedAffixes = { }
	for modId, mod in pairs(item.affixes) do
		if (not affixType or (mod.type == affixType)) and not excludeGroups[mod.group] and not item:CheckIfModIsDelve(mod) then
			if item:CanHaveMod(mod, extraTags) then
				t_insert(affixList, modId)
			elseif modId == selAffix then
				t_insert(affixList, modId)
				retainedAffixes[modId] = true
			end
		end
	end
	table.sort(affixList, function(a, b)
		local modA = item.affixes[a]
		local modB = item.affixes[b]
		for i = 1, m_max(#modA, #modB) do
			if not modA[i] then
				return true
			elseif not modB[i] then
				return false
			elseif modA.statOrder[i] ~= modB.statOrder[i] then
				return modA.statOrder[i] < modB.statOrder[i]
			end
		end
		if modA.level ~= modB.level then
			return modA.level < modB.level
		end
		return a < b
	end)
	control.selIndex = 1
	control.list = { "None" }
	control.outputTable = outputTable
	control.outputIndex = outputIndex
	control.slider.shown = false
	control.slider.val = main.defaultItemAffixQuality or 0.5
	local selAffix = item[outputTable][outputIndex].modId
	local lastSeries
	-- combine runs of modifiers to one group, which will only take up one row
	-- in the list
	for _, modId in ipairs(affixList) do
		local mod = item.affixes[modId]
		if not lastSeries or not tableDeepEquals(lastSeries.statOrder, mod.statOrder) then
			local modString = table.concat(mod, "/")
			lastSeries = {
				label = modString,
				modList = {},
				haveRange = modString:match("%(%-?[%d%.]+%-%-?[%d%.]+%)"),
				statOrder = mod.statOrder,
			}
			t_insert(control.list, lastSeries)
		end
		-- cluster jewel mods retained after changing the cluster type
		if retainedAffixes[modId] then
			lastSeries.label = "^8[Retained] " .. lastSeries.label
		end
		t_insert(lastSeries.modList, modId)
		if #lastSeries.modList == 2 then
			lastSeries.label = lastSeries.label:gsub("%(%-?[%d%.]+%-%-?[%d%.]+%)", "#"):gsub("%-?%d+%.?%d*", "#")
			lastSeries.haveRange = true
		end
	end

	local sortOption = self.controls.craftingSorting:GetSelValue()
	-- sort modifier groups by power
	if sortOption.stat and self.controls.craftingSortingLabel.shown() then
		local calcFunc = self.build.calcsTab:GetMiscCalculator()
		local slotName = self.displayItem:GetPrimarySlot()
		local testSubject = new("Item", self.displayItem:BuildRaw())
		local controlPowerCache = powerCache
		if selAffix and selAffix ~= "None" then
			testSubject[outputTable][outputIndex] = { modId = "None" }
			testSubject:Craft()
			controlPowerCache = { }
		end
		local function pickModifierFromList(modList)
			-- pick mid tier modifier from a group
			if #modList == 1 then
				return modList[1]
			else
				return modList[1 + round((#modList - 1) * main.defaultItemAffixQuality)]
			end
		end
		local function getPower(modId)
			if controlPowerCache[modId] then
				return controlPowerCache[modId]
			end
			local mod = testSubject.affixes[modId]

			local modCount = #mod
			-- magnitude scaling happens during item parsing, which means we
			-- can't use the faster path where items don't need to be
			-- re-parsed. note that this wouldn't be correct if we were
			-- adding a mod magnitude mod here, but currently all mod
			-- magnitude mods are custom modifiers and so this works
			local power
			if (#testSubject.modMagnitudeMods > 0) or (testSubject.catalyst and testSubject.catalyst > 0) then
				local originalItem = testSubject:BuildRaw()
				for _, subMod in ipairs(mod) do
					local modLine = { line = subMod, modTags = mod.modTags, [mod.type] = true }
					t_insert(testSubject.explicitModLines, modLine)
				end
				testSubject:BuildAndParseRaw()
				power = data.powerStatList.GetFromOutput(
					calcFunc({ repSlotName = slotName, repItem = testSubject }),
					sortOption
				)
				testSubject = new("Item", originalItem)
			else
				for _, line in ipairs(mod) do
					local rangedLine = itemLib.applyRange(line, main.defaultItemAffixQuality or 0.5, 1, 1)
					local modList, extra = modLib.parseMod(rangedLine)
					local modLine = { line = line, modList = modList, extra = extra, modTags = mod.modTags, [mod.type] = true }
					t_insert(testSubject.explicitModLines, modLine)
				end

				testSubject:BuildModList()
				power = data.powerStatList.GetFromOutput(
					calcFunc({ repSlotName = slotName, repItem = testSubject }),
					sortOption
				)
				for _ = 1, modCount do
					t_remove(testSubject.explicitModLines, #testSubject.explicitModLines)
				end
			end
			controlPowerCache[modId] = power
			return power
		end
		table.sort(control.list, function(a, b)
			-- keep "None" as the first option
			if not a.modList then
				return true
			elseif not b.modList then
				return false
			end

			local modIdA = pickModifierFromList(a.modList)
			local modIdB = pickModifierFromList(b.modList)

			return getPower(modIdA) > getPower(modIdB)
		end)
	end
	local function findSelectedIdx()
		for i, entry in ipairs(control.list) do
			if entry.modList then
				for _, modId in ipairs(entry.modList) do
					if selAffix == modId then
						return i
					end
				end
			else
				if selAffix == entry then
					return i
				end
			end
		end
		return 1
	end
	control.selIndex = findSelectedIdx()
	if control.list[control.selIndex].haveRange then
		control.slider.divCount = #control.list[control.selIndex].modList
		local index = isValueInArray(control.list[control.selIndex].modList, selAffix)
		-- Imported legacy rolls can sit outside the current 0-1 affix range.
		-- Keep that value on the affix, but show the nearest slider endpoint.
		local affixRange = item[outputTable][outputIndex].range
		local range = m_min(1, m_max(0, type(affixRange) == "table" and affixRange[1] or affixRange or 0.5))
		-- Avoid exact integer boundary that slider:GetDivVal's ceil would assign to the previous segment
		if range == 0 and index > 1 then
			range = 1e-4
		end
		control.slider.val = (index - 1 + range) / control.slider.divCount
		if control.slider.divCount == 1 then
			control.slider.divCount = nil
		end
		control.slider.shown = true
	end
end

-- Create/update custom modifier controls
function ItemsTabClass:UpdateCustomControls()
	local item = self.displayItem
	local i = 1
	local modLines = copyTable(item.explicitModLines)
	if item.crucibleModLines and #item.crucibleModLines > 0 then
		for _, line in ipairs(item.crucibleModLines) do
			t_insert(modLines, line)
		end
	end
	if item.rareLikeUnique or item.rarity == "MAGIC" or item.rarity == "RARE" or (item.crucibleModLines and #item.crucibleModLines > 0) then
		for index, modLine in ipairs(modLines) do
			if modLine.custom or modLine.crafted or modLine.crucible then
				local line = itemLib.formatModLine(modLine)
				if line then
					if not self.controls["displayItemCustomModifierRemove"..i] then
						self.controls["displayItemCustomModifierRemove"..i] = new("Selector")
						self.controls["displayItemCustomModifier"..i] = { }
						self.controls["displayItemCustomModifierLabel"..i] = { }
					end
					self.controls["displayItemCustomModifierRemove"..i].shown = true
					self.controls["displayItemCustomModifier"..i].label = line
					self.controls["displayItemCustomModifierLabel"..i].label = modLine.crafted and " ^7Crafted:" or modLine.crucible and "^7Crucible:" or " ^7Custom:"
					self.controls["displayItemCustomModifierRemove"..i].onClick = function()
						if index > #item.explicitModLines then
							t_remove(item.crucibleModLines, index - #item.explicitModLines)
						else
							t_remove(item.explicitModLines, index)
						end
						item:BuildAndParseRaw()
						local id = item.id
						self:CreateDisplayItemFromRaw(item:BuildRaw())
						self.displayItem.id = id
					end
					i = i + 1
				end
			end
		end
	end
	item.customCount = i - 1
	while self.controls["displayItemCustomModifierRemove"..i] do
		self.controls["displayItemCustomModifierRemove"..i].shown = false
		i = i + 1
	end
end

-- Updates the range line dropdown and range slider for the current display item
function ItemsTabClass:UpdateDisplayItemRangeLines()
	if self.displayItem and self.displayItem.rangeLineList[1] then
		wipeTable(self.controls.displayItemRangeLine.list)
		for _, modLine in ipairs(self.displayItem.rangeLineList) do
			if (modLine.modId and modLine.newModId) or modLine.range then
				t_insert(self.controls.displayItemRangeLine.list, { modLine = modLine, label = modLine.line })
			end
		end
		self.controls.displayItemRangeLine.selIndex = 1
		self.controls.displayItemRangeSlider.val = self.displayItem.rangeLineList[1].range
	end
end

local function checkLineForAllocates(line, nodes)
	if nodes and string.match(line, "Allocates") then
		local nodeId = tonumber(string.match(line, "%d+"))
		if nodes[nodeId] then
			return "Allocates "..nodes[nodeId].name
		end
	end
	return line
end

function ItemsTabClass:AddModComparisonTooltip(tooltip, mod)
	local slotName = self.displayItem:GetPrimarySlot()
	local newItem = new("Item", self.displayItem:BuildRaw())

	for _, subMod in ipairs(mod) do
		t_insert(newItem.explicitModLines, { line = checkLineForAllocates(subMod, self.build.spec.nodes), modTags = mod.modTags, [mod.type] = true })
	end

	newItem:BuildAndParseRaw()

	local calcFunc = self.build.calcsTab:GetMiscCalculator()
	local outputBase = calcFunc({ repSlotName = slotName, repItem = self.displayItem })
	local outputNew = calcFunc({ repSlotName = slotName, repItem = newItem })
	self.build:AddStatComparesToTooltip(tooltip, outputBase, outputNew, "\nAdding this mod will give: ")
end

-- Returns the first slot in which the given item is equipped
function ItemsTabClass:GetEquippedSlotForItem(item)
	for _, slot in ipairs(self.orderedSlots) do
		if not slot.inactive then
			if slot.selItemId == item.id then
				return slot
			end
			for _, itemSetId in ipairs(self.itemSetOrderList) do
				local itemSet = self.itemSets[itemSetId]
				if itemSetId ~= self.activeItemSetId and itemSet[slot.slotName] and itemSet[slot.slotName].selItemId == item.id then
					return slot, itemSet
				end
			end
		end
	end
end

function ItemsTabClass:GetComparisonSlotNameForItem(item)
	local equippedSlot = self:GetEquippedSlotForItem(item)
	if equippedSlot then
		return equippedSlot.slotName
	end
	if item.type == "Jewel" then
		for _, slot in ipairs(self.orderedSlots) do
			if not slot.inactive and slot.selItemId == 0 and slot:IsShown() and self:IsItemValidForSlot(item, slot.slotName) then
				return slot.slotName
			end
		end
	end
	return item:GetPrimarySlot()
end
-- Check if the given item could be equipped in the given slot, taking into account possible conflicts with currently equipped items
-- For example, a shield is not valid for Weapon 2 if Weapon 1 is a staff, and a wand is not valid for Weapon 2 if Weapon 1 is a dagger
function ItemsTabClass:IsItemValidForSlot(item, slotName, itemSet)
	itemSet = itemSet or self.activeItemSet
	local slotType, slotId = slotName:match("^([%a ]+) (%d+)$")
	if not slotType then
		slotType = slotName
	end
	if slotType == "Jewel" then
		-- Special checks for jewel sockets
		local node = self.build.spec.tree.nodes[tonumber(slotId)] or self.build.spec.nodes[tonumber(slotId)]
		if not node or item.type ~= "Jewel" then
			return false
		elseif node.charmSocket or item.base.subType == "Charm" then
			-- Charm sockets can only have charms, and charms can only be in charm sockets
			if node.charmSocket and item.base.subType == "Charm" then
				return true
			end
			return false
		elseif item.clusterJewel and not node.expansionJewel then
			-- Don't allow cluster jewels in inner sockets
			return false
		elseif not node.expansionJewel or node.expansionJewel.size == 2 then
			-- Outer sockets can fit anything
			return true
		else
			-- Only allow jewels that fit in this socket
			return not item.clusterJewel or item.clusterJewel.sizeIndex <= node.expansionJewel.size
		end
	elseif item.type == slotType then
		return true
	elseif item.type == "Tincture" and slotType == "Flask" then
		return true
	elseif item.type == "Jewel" and item.base.subType == "Abyss" and slotName:match("Abyssal Socket") then
		return true
	elseif slotName == "Weapon 1" or slotName == "Weapon 1 Swap" or slotName == "Weapon" then
		return item.base.weapon ~= nil
	elseif slotName == "Weapon 2" or slotName == "Weapon 2 Swap" then
		local weapon1Sel = itemSet[slotName == "Weapon 2" and "Weapon 1" or "Weapon 1 Swap"].selItemId or 0
		local weapon1Type = self.items[weapon1Sel] and self.items[weapon1Sel].base.type or "None"
		if weapon1Type == "None" then
			return item.type == "Shield" or (self.build.data.weaponTypeInfo[item.type] and self.build.data.weaponTypeInfo[item.type].oneHand)
		elseif weapon1Type == "Bow" then
			return item.type == "Quiver"
		elseif self.build.data.weaponTypeInfo[weapon1Type].oneHand then
			return item.type == "Shield" or (self.build.data.weaponTypeInfo[item.type] and self.build.data.weaponTypeInfo[item.type].oneHand and ((weapon1Type == "Wand" and item.type == "Wand") or (weapon1Type ~= "Wand" and item.type ~= "Wand")))
		end
	end
end


---Gets the name of the anointed node on an item
---@param item table @The item to get the anoint from
---@return string @The name of the anointed node, or nil if there is no anoint
function ItemsTabClass:getAnoint(item)
	local result = { }
	if item then
		for _, modList in ipairs{item.enchantModLines, item.scourgeModLines, item.implicitModLines, item.explicitModLines, item.crucibleModLines} do
			for _, mod in ipairs(modList) do
				local line = mod.line
				local anoint = line:find("Allocates ([a-zA-Z ]+)")
				if anoint then
					local nodeName = line:sub(anoint + string.len("Allocates "))
					t_insert(result, nodeName)
				end
			end
		end
	end
	return result
end

---Gets how many anoint slots are still missing on an item.
---@param item table @The item to inspect
---@return number @How many additional anoints can still be applied
function ItemsTabClass:getMissingAnointCount(item)
	if not isAnointable(item) then
		return 0
	end
	local maxAnoints = item.canHaveFourEnchants and 4 or item.canHaveThreeEnchants and 3 or item.canHaveTwoEnchants and 2 or 1
	local anointCount = #self:getAnoint(item)
	return m_max(0, maxAnoints - m_min(anointCount, maxAnoints))
end

---Returns a copy of the currently displayed item, but anointed with a new node.
---Removes any existing enchantments before anointing. (Anoints are considered enchantments)
---@param node table @The passive tree node to anoint, or nil to just remove existing anoints.
---@return table @The new item
function ItemsTabClass:anointItem(node)
	self.anointEnchantSlot = self.anointEnchantSlot or 1
	local item = new("Item", self.displayItem:BuildRaw())
	item.id = self.displayItem.id
	if #item.enchantModLines >= self.anointEnchantSlot then
		t_remove(item.enchantModLines, self.anointEnchantSlot)
	end
	if node then
		t_insert(item.enchantModLines, self.anointEnchantSlot, { crafted = true, line = "Allocates " .. node.dn })
	end
	item:BuildAndParseRaw()
	return item
end

---Appends tooltip information for anointing a new passive tree node onto the currently editing item
---@param tooltip table @The tooltip to append into
---@param node table @The passive tree node that will be anointed, or nil to remove the current anoint.
function ItemsTabClass:AppendAnointTooltip(tooltip, node, actionText)
	if not self.displayItem then
		return
	end

	if not actionText then
		actionText = "Anointing"
	end

	local header
	if node then
		if self.build.spec.allocNodes[node.id] then
			tooltip:AddLine(14, "^7"..actionText.." "..node.dn.." changes nothing because this node is already allocated on the tree.")
			return
		end

		local curAnoints = self:getAnoint(self.displayItem)
		if curAnoints and #curAnoints > 0 then
			for _, curAnoint in ipairs(curAnoints) do
				if curAnoint == node.dn then
					tooltip:AddLine(14, "^7"..actionText.." "..node.dn.." changes nothing because this node is already anointed.")
					return
				end
			end
		end

		header = "^7"..actionText.." "..node.dn.." will give you: "
	else
		header = "^7"..actionText.." nothing will give you: "
	end
	local calcFunc = self.build.calcsTab:GetMiscCalculator()
	local repSlotName = self.displayItem.base and self.displayItem.base.type or "Amulet"
	local outputBase = calcFunc({ repSlotName = repSlotName, repItem = self.displayItem })
	local outputNew = calcFunc({ repSlotName = repSlotName, repItem = self:anointItem(node) })
	local numChanges = self.build:AddStatComparesToTooltip(tooltip, outputBase, outputNew, header)
	if node and numChanges == 0 then
		tooltip:AddLine(14, "^7"..actionText.." "..node.dn.." changes nothing.")
	end
end

---Appends tooltip with information about added notable passive node if it would be allocated.
---@param tooltip table @The tooltip to append into
---@param node table @The passive tree node that will be added
function ItemsTabClass:AppendAddedNotableTooltip(tooltip, node)
	local calcFunc, calcBase = self.build.calcsTab:GetMiscCalculator()
	local outputNew = calcFunc({ addNodes = { [node] = true } })
	local numChanges = self.build:AddStatComparesToTooltip(tooltip, calcBase, outputNew, "^7Allocating "..node.dn.." will give you: ")
	if numChanges == 0 then
		tooltip:AddLine(14, "^7Allocating "..node.dn.." changes nothing.")
	end
end



function ItemsTabClass:AddItemSetTooltip(tooltip, itemSet)
	for _, slot in ipairs(self.orderedSlots) do
		if not slot.nodeId then
			local item = self.items[itemSet[slot.slotName].selItemId]
			if item then
				tooltip:AddLine(16, s_format("^7%s: %s%s", slot.label, colorCodes[item.rarity], item.name))
			end
		end
	end
end

function ItemsTabClass:SetTooltipHeaderInfluence(tooltip, item)
	tooltip.influenceHeader1 = nil
	tooltip.influenceHeader2 = nil

	local function addInfluence(name)
		if not tooltip.influenceHeader1 then
			tooltip.influenceHeader1 = name
		elseif not tooltip.influenceHeader2 then
			tooltip.influenceHeader2 = name
		end
	end

	-- Eater and Exarch combo takes priority over fractured icon.
	if item.cleansing and item.tangle then
		addInfluence("Exarch")
		addInfluence("Eater")
	else
		-- Dual influence with fracture will show fractured icon and highest priority influence.
		if item.fractured then
			addInfluence("Fractured")
		end
		-- Replica Eternity Shroud has Experimented icon and Shaper icon on the right.
		if item.title and item.title:find("Replica") then
			addInfluence("Experimented")
		end
		if item.foulborn then
			addInfluence("Foulborn")
		end
		if item.veiled then
			addInfluence("Veiled")
		end
		if item.cleansing then
			addInfluence("Exarch")
		end
		if item.tangle then
			addInfluence("Eater")
		end
		if item.shaper then
			addInfluence("Shaper")
		end
		if item.elder then
			addInfluence("Elder")
		end
		if item.crusader then
			addInfluence("Crusader")
		end
		if item.eyrie then
			addInfluence("Redeemer")
		end
		if item.basilisk then
			addInfluence("Hunter")
		end
		if item.adjudicator then
			addInfluence("Warlord")
		end
		if item.vestigial then
			addInfluence("Vestigial")
		end
		if item.synthesised and not tooltip.influenceHeader1 then
			addInfluence("Synthesis")
		end
		if item.memoryStrands and not tooltip.influenceHeader1 then
			addInfluence("Memory")
		end
	end

	if tooltip.influenceHeader1 and not tooltip.influenceHeader2 then
		tooltip.influenceHeader2 = tooltip.influenceHeader1
	end
end

function ItemsTabClass:FormatItemSource(text)
	return text:gsub("unique{([^}]+)}",colorCodes.UNIQUE.."%1"..colorCodes.SOURCE)
			   :gsub("normal{([^}]+)}",colorCodes.NORMAL.."%1"..colorCodes.SOURCE)
			   :gsub("currency{([^}]+)}",colorCodes.CURRENCY.."%1"..colorCodes.SOURCE)
			   :gsub("prophecy{([^}]+)}",colorCodes.PROPHECY.."%1"..colorCodes.SOURCE)
end

local function itemChangesPassiveTree(item)
	return not not (item and item.type == "Jewel" and item.jewelData
		and (item.jewelData.conqueredBy or item.jewelRadiusIndex
			and (item.jewelData.intuitiveLeapLike or item.jewelData.impossibleEscapeKeystone)))
end

-- These jewels can replace passive nodes or disconnect allocated passives, so
-- rebuild the passive tree before comparing their stats.
-- Keep this list in sync with PassiveSpec's constructor, Init, and Select*
-- methods; omitted fields fail safe as nil on the comparison spec.
local sharedSpecKeysForJewelComparison = {
	build = true,
	treeVersion = true,
	tree = true,
	title = true,
	ignoreAllocatingSubgraph = true,
	clusterHashFormatVersion = true,
	curClassId = true,
	curClass = true,
	curClassName = true,
	curAscendClassId = true,
	curAscendClass = true,
	curAscendClassName = true,
	curAscendClassBaseName = true,
	curSecondaryAscendClassId = true,
	curSecondaryAscendClass = true,
	curSecondaryAscendClassName = true,
}

local function cloneSpecForJewelComparison(spec)
	local specCopy = setmetatable({ }, getmetatable(spec))
	-- Share only immutable/scalar spec state. Tables that BuildAllDependsAndPaths
	-- may mutate must be owned by the comparison spec.
	for key in pairs(sharedSpecKeysForJewelComparison) do
		specCopy[key] = spec[key]
	end

	specCopy.nodes = { }
	for id, node in pairs(spec.nodes) do
		local nodeCopy = setmetatable({ }, getmetatable(node))
		for key, value in pairs(node) do
			if key ~= "linked" and key ~= "depends" and key ~= "intuitiveLeapLikesAffecting"
			and key ~= "path" and key ~= "power" then
				nodeCopy[key] = value
			end
		end
		nodeCopy.alloc = false
		nodeCopy.linked = { }
		nodeCopy.depends = { }
		nodeCopy.intuitiveLeapLikesAffecting = { }
		nodeCopy.power = { }
		specCopy.nodes[id] = nodeCopy
	end
	for id, nodeCopy in pairs(specCopy.nodes) do
		for _, linkedNode in ipairs(spec.nodes[id].linked or { }) do
			local linkedCopy = specCopy.nodes[linkedNode.id]
			if linkedCopy then
				t_insert(nodeCopy.linked, linkedCopy)
			end
		end
	end

	specCopy.allocNodes = { }
	for id in pairs(spec.allocNodes) do
		local nodeCopy = specCopy.nodes[id]
		if nodeCopy then
			nodeCopy.alloc = true
			specCopy.allocNodes[id] = nodeCopy
		end
	end
	specCopy.jewels = copyTable(spec.jewels, true)
	specCopy.masterySelections = copyTable(spec.masterySelections, true)
	specCopy.hashOverrides = copyTable(spec.hashOverrides, true)
	specCopy.ignoredNodes = copyTable(spec.ignoredNodes, true)
	specCopy.splitPersonalityPath = { }
	specCopy.allocSubgraphNodes = { }
	specCopy.allocExtendedNodes = { }
	specCopy.subGraphs = { }

	return specCopy
end

local function buildSpecForJewelComparison(itemsTab, compareSlot, replacementItem)
	local tempItemId
	local spec = cloneSpecForJewelComparison(itemsTab.build.spec)
	if replacementItem then
		if replacementItem.id and itemsTab.items[replacementItem.id] == replacementItem then
			spec.jewels[compareSlot.nodeId] = replacementItem.id
		else
			tempItemId = -1
			while itemsTab.items[tempItemId] do
				tempItemId = tempItemId - 1
			end
			itemsTab.items[tempItemId] = replacementItem
			spec.jewels[compareSlot.nodeId] = tempItemId
		end
	else
		spec.jewels[compareSlot.nodeId] = nil
	end

	local ok, err = xpcall(function()
		spec:BuildAllDependsAndPaths()
	end, debug.traceback)
	if tempItemId then
		itemsTab.items[tempItemId] = nil
	end
	if not ok then
		error(err, 0)
	end
	return spec
end

function ItemsTabClass:GetSocketDescriptionLine(item)
	-- Sockets/links
	local group = 0
	local line = ""
	for i, socket in ipairs(item.sockets) do
		if i > 1 then
			if socket.group == group then
				line = line .. "^7="
			else
				line = line .. "  "
			end
			group = socket.group
		end
		local code
		if socket.color == "R" then
			code = colorCodes.STRENGTH
		elseif socket.color == "G" then
			code = colorCodes.DEXTERITY
		elseif socket.color == "B" then
			code = colorCodes.INTELLIGENCE
		elseif socket.color == "W" then
			code = colorCodes.SCION
		elseif socket.color == "A" then
			code = "^xB0B0B0"
		end
		line = line .. code .. socket.color
	end
	return line
end
function ItemsTabClass:AddItemTooltip(tooltip, item, slot, dbMode, maxWidth)
	local fontSizeSmall = main.showFlavourText and 16 or 14
	local fontSizeBig = main.showFlavourText and 18 or 16
	local fontSizeTitle = main.showFlavourText and 22 or 20
	local rarityCode = colorCodes[item.rarity]
	tooltip.maxWidth = m_min(maxWidth or 600, 600) -- Cap very long lines. Can use a narrower width for small viewports
	tooltip.tooltipHeader = item.rarity
	tooltip.foilType = item.foilType
	tooltip.center = true
	tooltip.color = rarityCode
	-- Shared items can use old base names that no longer exist. Add a tooltip so they can be copied or removed without causing a crash.
	if not item.base or not item.baseName then
		tooltip:AddLine(fontSizeTitle, rarityCode..(item.title or item.name or "Unknown Item"), "FONTIN SC")
		tooltip:AddSeparator(30)
		tooltip:AddLine(fontSizeTitle, colorCodes.NEGATIVE.."Item base is not supported by the current version.", "FONTIN SC")
		return
	end
	self:SetTooltipHeaderInfluence(tooltip, item)
	-- Item name
	if item.title then
		tooltip:AddLine(fontSizeTitle, rarityCode..item.title, "FONTIN SC")
		tooltip:AddLine(fontSizeTitle, rarityCode..item.baseName:gsub(" %(.+%)",""),"FONTIN SC")
	else
		tooltip:AddLine(fontSizeTitle, rarityCode..item.namePrefix..item.baseName:gsub(" %(.+%)","")..item.nameSuffix, "FONTIN SC")
	end
	for _, curInfluenceInfo in ipairs(influenceInfo) do
		if item[curInfluenceInfo.key] and not main.showFlavourText then
			tooltip:AddLine(fontSizeBig, curInfluenceInfo.color..curInfluenceInfo.display.." Item", "FONTIN SC")
		end
	end
	if item.fractured and not main.showFlavourText then
		tooltip:AddLine(fontSizeBig, colorCodes.FRACTURED.."Fractured Item")
	end
	if item.synthesised and not main.showFlavourText then
		tooltip:AddLine(fontSizeBig, colorCodes.CRAFTED.."Synthesised Item")
	end
	tooltip:AddSeparator(10)

	-- Special fields for database items
	if dbMode then
		if item.usesVariantGroups then
			if item.versionList and item.selectedVersion then
				tooltip:AddLine(fontSizeBig, "^xFFFF30Version: " .. item.versionList[item.selectedVersion], "FONTIN SC")
			end
			local selectedVariants = { }
			for groupId in pairsSortByKey(item.variantGroups) do
				local variantId = item.variantGroupSelections[groupId]
				if variantId and item:IsVariantGroupOptionEligible(groupId, variantId) then
					t_insert(selectedVariants, item.variantList[variantId])
				end
			end
			if #selectedVariants > 0 then
				tooltip:AddLine(fontSizeBig, "^xFFFF30Variants: " .. table.concat(selectedVariants, ", "), "FONTIN SC")
			end
		elseif item.variantList then
			if #item.variantList == 1 then
				tooltip:AddLine(fontSizeBig, "^xFFFF30Variant: "..item.variantList[1], "FONTIN SC")
			else
				tooltip:AddLine(fontSizeBig, "^xFFFF30Variant: "..item.variantList[item.variant].." ("..#item.variantList.." variants)", "FONTIN SC")
			end
		end
		if item.league then
			tooltip:AddLine(fontSizeBig, "^xFF5555Exclusive to: "..item.league, "FONTIN SC")
		end
		if item.unreleased then
			tooltip:AddLine(fontSizeBig, colorCodes.NEGATIVE.."Not yet available", "FONTIN SC")
		end
		if item.source then
			tooltip:AddLine(fontSizeBig, colorCodes.SOURCE.."Source: "..self:FormatItemSource(item.source), "FONTIN SC")
		end
		if item.upgradePaths then
			for _, path in ipairs(item.upgradePaths) do
				tooltip:AddLine(fontSizeBig, colorCodes.SOURCE..self:FormatItemSource(path), "FONTIN SC")
			end
		end
		tooltip:AddSeparator(10)
	end

	local base = item.base
	local slotNum = slot and slot.slotNum or (IsKeyDown("SHIFT") and 2 or 1)
	local modList = item.modList or item.slotModList[slotNum]
	if base.weapon then
		-- Weapon-specific info
		local weaponData = item.weaponData[slotNum]
		tooltip:AddLine(fontSizeBig, s_format("^x7F7F7F%s", self.build.data.weaponTypeInfo[base.type].label or base.subType or base.type), "FONTIN SC")
		if item.quality > 0 then
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FQuality: "..colorCodes.MAGIC.."+%d%%", item.quality), "FONTIN SC")
		end
		local totalDamageTypes = 0
		if weaponData.PhysicalDPS then
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FPhysical Damage: "..colorCodes.MAGIC.."%d-%d (%.1f DPS)", weaponData.PhysicalMin, weaponData.PhysicalMax, weaponData.PhysicalDPS), "FONTIN SC")
			totalDamageTypes = totalDamageTypes + 1
		end
		if weaponData.ElementalDPS then
			local elemLine
			for _, var in ipairs({"Fire","Cold","Lightning"}) do
				if weaponData[var.."DPS"] then
					elemLine = elemLine and elemLine.."^x7F7F7F, " or "^x7F7F7FElemental Damage: "
					elemLine = elemLine..s_format("%s%d-%d", colorCodes[var:upper()], weaponData[var.."Min"], weaponData[var.."Max"])
				end
			end
			tooltip:AddLine(fontSizeBig, elemLine, "FONTIN SC")
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FElemental DPS: "..colorCodes.MAGIC.."%.1f", weaponData.ElementalDPS), "FONTIN SC")
			totalDamageTypes = totalDamageTypes + 1
		end
		if weaponData.ChaosDPS then
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FChaos Damage: "..colorCodes.CHAOS.."%d-%d "..colorCodes.MAGIC.."(%.1f DPS)", weaponData.ChaosMin, weaponData.ChaosMax, weaponData.ChaosDPS), "FONTIN SC")
			totalDamageTypes = totalDamageTypes + 1
		end
		if totalDamageTypes > 1 then
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FTotal DPS: "..colorCodes.MAGIC.."%.1f", weaponData.TotalDPS), "FONTIN SC")
		end
		tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FCritical Strike Chance: %s%.2f%%", main:StatColor(weaponData.CritChance, base.weapon.CritChanceBase), weaponData.CritChance), "FONTIN SC")
		tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FAttacks per Second: %s%.2f", main:StatColor(weaponData.AttackRate, base.weapon.AttackRateBase), weaponData.AttackRate), "FONTIN SC")
		if weaponData.range < 120 then
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FWeapon Range: %s%.1f ^x7F7F7Fmetres", main:StatColor(weaponData.range, base.weapon.Range), weaponData.range / 10), "FONTIN SC")
		end
	elseif base.armour then
		-- Armour-specific info
		local armourData = item.armourData
		if item.quality > 0 then
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FQuality: "..colorCodes.MAGIC.."+%d%%", item.quality), "FONTIN SC")
		end
		if base.armour.BlockChance and armourData.BlockChance > 0 then
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FChance to Block: %s%d%%", main:StatColor(armourData.BlockChance, base.armour.BlockChance), armourData.BlockChance), "FONTIN SC")
		end
		if armourData.Armour > 0 then
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FArmour: %s%d", main:StatColor(armourData.Armour, base.armour.ArmourBase), armourData.Armour), "FONTIN SC")
		end
		if armourData.Evasion > 0 then
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FEvasion Rating: %s%d", main:StatColor(armourData.Evasion, base.armour.EvasionBase), armourData.Evasion), "FONTIN SC")
		end
		if armourData.EnergyShield > 0 then
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FEnergy Shield: %s%d", main:StatColor(armourData.EnergyShield, base.armour.EnergyShieldBase), armourData.EnergyShield), "FONTIN SC")
		end
		if armourData.Ward > 0 then
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FWard: %s%d", main:StatColor(armourData.Ward, base.armour.WardBase), armourData.Ward), "FONTIN SC")
		end
	elseif base.flask then
		-- Flask-specific info
		local flaskData = item.flaskData
		if item.quality > 0 then
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FQuality: "..colorCodes.MAGIC.."+%d%%", item.quality), "FONTIN SC")
		end
		if flaskData.lifeTotal then
			if flaskData.lifeGradual ~= 0 then
				tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FRecovers %s%d ^x7F7F7FLife over %s%.1f0 ^x7F7F7FSeconds",
					main:StatColor(flaskData.lifeTotal, base.flask.life), flaskData.lifeGradual,
					main:StatColor(flaskData.duration, base.flask.duration), flaskData.duration
					), "FONTIN SC")
			end
			if flaskData.lifeInstant ~= 0 then
				tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FRecovers %s%d ^x7F7F7FLife instantly", main:StatColor(flaskData.lifeTotal, base.flask.life), flaskData.lifeInstant), "FONTIN SC")
			end
		end
		if flaskData.manaTotal then
			if flaskData.manaGradual ~= 0 then
				tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FRecovers %s%d ^x7F7F7FMana over %s%.1f0 ^x7F7F7FSeconds",
					main:StatColor(flaskData.manaTotal, base.flask.mana), flaskData.manaGradual,
					main:StatColor(flaskData.duration, base.flask.duration), flaskData.duration
					), "FONTIN SC")
			end
			if flaskData.manaInstant ~= 0 then
				tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FRecovers %s%d ^x7F7F7FMana instantly", main:StatColor(flaskData.manaTotal, base.flask.mana), flaskData.manaInstant), "FONTIN SC")
			end
		end
		if not flaskData.lifeTotal and not flaskData.manaTotal then
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FLasts %s%.2f ^x7F7F7FSeconds", main:StatColor(flaskData.duration, base.flask.duration), flaskData.duration), "FONTIN SC")
		end
		tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FConsumes %s%d ^x7F7F7Fof %s%d ^x7F7F7FCharges on use",
			main:StatColor(flaskData.chargesUsed, base.flask.chargesUsed), flaskData.chargesUsed,
			main:StatColor(flaskData.chargesMax, base.flask.chargesMax), flaskData.chargesMax
		), "FONTIN SC")
		for _, modLine in pairs(item.buffModLines) do
			if modLine.extra then
				local line = colorCodes.UNSUPPORTED..modLine.line
				line = main.notSupportedModTooltips and (line .. main.notSupportedTooltipText) or line
				tooltip:AddLine(fontSizeBig, line, "FONTIN SC")
			else
				tooltip:AddLine(fontSizeBig, colorCodes.MAGIC..modLine.line, "FONTIN SC")
			end
		end
	elseif base.tincture then
		-- Tincture-specific info
		local tinctureData = item.tinctureData

		if item.quality and item.quality > 0 then
			tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FQuality: "..colorCodes.MAGIC.."+%d%%", item.quality), "FONTIN SC")
		end

		tooltip:AddLine(fontSizeBig, s_format("^x7F7F7FInflicts Mana Burn every %s%.2f ^x7F7F7FSeconds", main:StatColor(tinctureData.manaBurn, base.tincture.manaBurn), tinctureData.manaBurn), "FONTIN SC")
		tooltip:AddLine(fontSizeBig, s_format("^x7F7F7F%s%.2f ^x7F7F7FSecond Cooldown When Deactivated", main:StatColor(tinctureData.cooldown, base.tincture.cooldown), tinctureData.cooldown), "FONTIN SC")
		for _, modLine in pairs(item.buffModLines) do
			if modLine.extra then
				local line = colorCodes.UNSUPPORTED..modLine.line
				line = main.notSupportedModTooltips and (line .. main.notSupportedTooltipText) or line
				tooltip:AddLine(fontSizeBig, line, "FONTIN SC")
			else
				tooltip:AddLine(fontSizeBig, colorCodes.MAGIC..modLine.line, "FONTIN SC")
			end
		end
	elseif item.type == "Jewel" then
		-- Jewel-specific info
		if item.limit then
			tooltip:AddLine(fontSizeBig, "^x7F7F7FLimited to: ^7"..item.limit, "FONTIN SC")
		end
		if item.classRestriction then
			tooltip:AddLine(fontSizeBig, "^x7F7F7FRequires Class "..(self.build.spec.curClassName == item.classRestriction and colorCodes.POSITIVE or colorCodes.NEGATIVE)..item.classRestriction, "FONTIN SC")
		end
		if item.jewelRadiusLabel then
			tooltip:AddLine(fontSizeBig, "^x7F7F7FRadius: ^7"..item.jewelRadiusLabel, "FONTIN SC")
		end
		if item.jewelRadiusData and slot and item.jewelRadiusData[slot.nodeId] then
			local radiusData = item.jewelRadiusData[slot.nodeId]
			local line
			local codes = { colorCodes.STRENGTH, colorCodes.DEXTERITY, colorCodes.INTELLIGENCE }
			for i, stat in ipairs({"Str","Dex","Int"}) do
				if radiusData[stat] and radiusData[stat] ~= 0 then
					line = (line and line .. ", " or "") .. s_format("%s%d %s^7", codes[i], radiusData[stat], stat)
				end
			end
			if line then
				tooltip:AddLine(fontSizeBig, "^x7F7F7FAttributes in Radius: "..line, "FONTIN SC")
			end
		end
	end
	if item.catalyst and item.catalyst > 0 and item.catalyst <= #catalystQualityFormat and item.catalystQuality and item.catalystQuality > 0 then
		tooltip:AddLine(fontSizeBig, s_format(catalystQualityFormat[item.catalyst], item.catalystQuality), "FONTIN SC")
		tooltip:AddSeparator(10)
	end

	if #item.sockets > 0 then
		local line = self:GetSocketDescriptionLine(item)
		tooltip:AddLine(fontSizeBig, "^x7F7F7FSockets: "..line, "FONTIN SC")
	end

	if item.talismanTier then
		tooltip:AddLine(fontSizeBig, "^x7F7F7FTalisman Tier ^xFFFFFF"..item.talismanTier, "FONTIN SC")
		tooltip:AddSeparator(10)
	end
	if item.memoryStrands then
		if main.showFlavourText then tooltip:AddLine(5, "") end
		tooltip:AddLine(fontSizeBig, colorCodes.MEMORY .. s_format("Memory Strands: ^7%d", item.memoryStrands), "FONTIN SC", nil, "Assets/memorybg.png")
		if main.showFlavourText then tooltip:AddLine(5, "") end
	end
	if item.intangibility then
		if main.showFlavourText then tooltip:AddLine(5, "") end
		tooltip:AddLine(fontSizeBig, colorCodes.INTANGIBILITY .. s_format("Intangibility: ^7%d%%", item.intangibility), "FONTIN SC", nil, "Assets/intangibilitybg.png")
		if main.showFlavourText then tooltip:AddLine(5, "") end
	end
	tooltip:AddSeparator(10)
	-- Requirements
	self.build:AddRequirementsToTooltip(tooltip, item.requirements.level,
		item.requirements.strMod, item.requirements.dexMod, item.requirements.intMod,
		item.requirements.str or 0, item.requirements.dex or 0, item.requirements.int or 0)

	-- Modifiers
	for _, modList in ipairs{item.enchantModLines, item.scourgeModLines, item.implicitModLines, item.explicitModLines, item.crucibleModLines} do
		if modList[1] then
			for _, modLine in ipairs(modList) do
				local variantCount = item:GetModLineVariantCount(modLine)
				if variantCount > 0 then
					local formattedModLine = itemLib.formatModLine(modLine, dbMode)
					if formattedModLine then
						for _ = 1, variantCount do
							tooltip:AddLine(fontSizeBig, formattedModLine, "FONTIN SC", modLine)
						end
					end
				end
			end
			tooltip:AddSeparator(10)
		end
	end

	-- Cluster jewel notables/keystone
	if item.clusterJewel then
		tooltip:AddSeparator(10)
		if #item.jewelData.clusterJewelNotables > 0 then
			for _, name in ipairs(item.jewelData.clusterJewelNotables) do
				local node = self.build.spec.tree.clusterNodeMap[name]
				if node then
					tooltip:AddLine(fontSizeBig, colorCodes.MAGIC .. node.dn, "FONTIN SC")
					for _, stat in ipairs(node.sd) do
						tooltip:AddLine(fontSizeBig, "^x7F7F7F"..stat, "FONTIN SC")
					end
				end
			end
		elseif item.jewelData.clusterJewelKeystone then
			local node = self.build.spec.tree.clusterNodeMap[item.jewelData.clusterJewelKeystone]
			if node then
				tooltip:AddLine(fontSizeBig, colorCodes.MAGIC .. node.dn, "FONTIN SC")
				for _, stat in ipairs(node.sd) do
					tooltip:AddLine(fontSizeBig, "^x7F7F7F"..stat, "FONTIN SC")
				end
			end
		end
		tooltip:AddSeparator(10)
	end

	-- Corrupted item label
	if item.corrupted or item.split or item.mirrored then
		if #item.explicitModLines == 0 then
			tooltip:AddSeparator(10)
		end
		if item.split then
			tooltip:AddLine(fontSizeBig, colorCodes.NEGATIVE.."Split", "FONTIN SC")
		end
		if item.mirrored then
			tooltip:AddLine(fontSizeBig, colorCodes.NEGATIVE.."Mirrored", "FONTIN SC")
		end
		if item.corrupted then
			tooltip:AddLine(fontSizeBig, colorCodes.NEGATIVE.."Corrupted", "FONTIN SC")
		end
		tooltip:AddSeparator(14)
	end

	-- Show flavour text:
	if (item.rarity == "UNIQUE" or item.rarity == "RELIC" or item.base.flavourText) and main.showFlavourText then
		local flavour = nil
		local flavourTable = nil

		if item.base.flavourText then
			flavour = item.base.flavourText
		else
			flavourTable = flavourLookup[item.title:gsub("^Foulborn%s+", "")]
		end

		if flavourTable then
			if (item.title and item.title:match("Grand Spectrum")) then
				local selectedFlavourId = nil
				for _, lineEntry in ipairs(tooltip.lines or {}) do
					local lineText = lineEntry.text or ""
					if lineText:find("Power") then
						selectedFlavourId = "UniqueJewel170"
						break
					elseif lineText:find("Endurance") then
						selectedFlavourId = "UniqueJewel168"
						break
					elseif lineText:find("Frenzy") then
						selectedFlavourId = "UniqueJewel169"
						break
					elseif lineText:find("Elemental Damage") then
						selectedFlavourId = "UniqueJewel168"
						break
					elseif lineText:find("Elemental Ailments") then
						selectedFlavourId = "UniqueJewel166"
						break
					elseif lineText:find("Elemental Resistances") or lineText:find("Armour") then
						selectedFlavourId = "UniqueJewel76"
						break
					elseif lineText:find("Maximum Life") then
						selectedFlavourId = "UniqueJewel165"
						break
					elseif lineText:find("Critical Strike Multiplier") then
						selectedFlavourId = "UniqueJewel167"
						break
					elseif lineText:find("Critical Strike Chance") or lineText:find("Mana") then
						selectedFlavourId = "UniqueJewel75"
						break
					end
				end
				if selectedFlavourId and flavourTable[selectedFlavourId] then
					flavour = flavourTable[selectedFlavourId]
				end
			else
				for _, text in pairs(flavourTable) do
					flavour = text
					break
				end
			end
		end

		if flavour then
			for _, line in ipairs(flavour) do
				tooltip:AddLine(fontSizeBig, colorCodes.UNIQUE .. line, "FONTIN SC ITALIC")
			end
			tooltip:AddSeparator(14)
		end
	end

	-- Skill tooltip. We add child tooltips, which will be rendered to the right of the main
	-- tooltip, growing downwards
	if not tooltip.childTooltips then
		tooltip.childTooltips = {}
	end
	for _, tt in ipairs(tooltip.childTooltips) do
		tt:Clear()
	end
	local itemSkills = copyTable(item.grantedSkills or {})
	-- append "Supported by #" to active skills
	for _, mod in ipairs(modList) do
		if mod.name == "ExtraSupport" then
			t_insert(itemSkills, mod.value)
		end
	end
	if #itemSkills > 0 then
		tooltip:AddSeparator(14)
		tooltip:AddLine(14,
			colorCodes.TIP ..
			"Tip: Hold Shift to display a tooltip for the granted skill" ..
			(#itemSkills > 1 and "s" or "") .. ".")
		for i, itemSkill in ipairs(itemSkills) do
			if not tooltip.childTooltips[i] then
				tooltip.childTooltips[i] = new("Tooltip")
			end
			-- find gem since the item data only contains the skill id
			local skill = data.skills[itemSkill.skillId]
			if skill and skill.id and IsKeyDown("SHIFT") then
				local gemId = data.gemForSkill[skill] or ""
				local gem = data.gems[gemId]
				-- if the skill has no matching gem, make up one. it will lack some information, but should still display somewhat correctly
				---@type GemToolTipOptions
				local options = {}
				if not gem then
					gem = { grantedEffect = skill, tags = {} }
					options.skipRequirements = true
				end
				local gemInst = {
					gemData = gem,
					level = itemSkill.level or 1,
					quality = 0,
					grantedEffect = skill
				}
				gemTooltip.AddGemTooltip(tooltip.childTooltips[i], self.build, gemInst, options)
			end
		end
	end
	-- Stat differences
	local itemTabHint = self.build.viewMode == "ITEMS" and "" or " in the Items tab"
	if not self.showStatDifferences then
		tooltip:AddSeparator(14)
		tooltip:AddLine(14, colorCodes.TIP.."Tip: Press Ctrl+D"..itemTabHint.." to enable the display of stat differences.")
		return
	end
	local calcFunc, calcBase = self.build.calcsTab:GetMiscCalculator()
	if base.flask then
		-- Special handling for flasks
		local stats = { }
		local flaskData = item.flaskData
		local modDB = self.build.calcsTab.mainEnv.modDB
		local output = self.build.calcsTab.mainOutput
		local durInc = modDB:Sum("INC", nil, "FlaskDuration")
		local effectInc = modDB:Sum("INC", { actor = "player" }, "FlaskEffect")
		local lifeDur = 0
		local manaDur = 0

		if item.rarity == "MAGIC" and not item.base.flask.life and not item.base.flask.mana then
			effectInc = effectInc + modDB:Sum("INC", { actor = "player" }, "MagicUtilityFlaskEffect")
		end

		if item.base.flask.life or item.base.flask.mana then
			local rateInc = modDB:Sum("INC", nil, "FlaskRecoveryRate")
			if item.base.flask.life then
				local lifeInc = modDB:Sum("INC", nil, "FlaskLifeRecovery")
				local lifeMore = modDB:More(nil, "FlaskLifeRecovery")
				local lifeRateInc = modDB:Sum("INC", nil, "FlaskLifeRecoveryRate")
				local instantPerc = flaskData.instantPerc + modDB:Sum("BASE", nil, "LifeFlaskInstantRecovery")

				-- More life recovery while on low life is not affected by flask effect (verified ingame).
				-- Since this will be multiplied by the flask effect value below we have to counteract this by removing the flask effect from the value beforehand.
				-- This is also the reason why this value needs a separate multiplier and cannot just be calculated into FlaskLifeRecovery.
				local lifeMoreOnLowLife = modDB:More(nil, "FlaskLifeRecoveryLowLife")
				local lowLifeMulti = (lifeMoreOnLowLife > 1 and ((lifeMoreOnLowLife - 1) / (1 + effectInc / 100)) + 1 or 1)

				local inst = flaskData.lifeBase * instantPerc / 100 * (1 + lifeInc / 100) * lifeMore * (1 + effectInc / 100) * lowLifeMulti
				local base = flaskData.lifeBase * (1 - instantPerc / 100) * (1 + lifeInc / 100) * lifeMore * (1 + effectInc / 100) * (1 + durInc / 100) * lowLifeMulti
				local grad = base * output.LifeRecoveryRateMod
				local esGrad = base * output.EnergyShieldRecoveryRateMod
				lifeDur = flaskData.duration * (1 + durInc / 100) / (1 + rateInc / 100) / (1 + lifeRateInc / 100)

				-- LocalLifeFlaskAdditionalLifeRecovery flask mods
				if flaskData.lifeAdditional > 0 and not self.build.configTab.input.conditionFullLife then
					local totalAdditionalAmount = (flaskData.lifeAdditional/100) * flaskData.lifeTotal * output.LifeRecoveryRateMod
					local additionalGrad = (lifeDur/10) * totalAdditionalAmount
					local leftoverDur = 10 - lifeDur
					local leftoverAmount = totalAdditionalAmount - additionalGrad

					if inst > 0 then
						if grad > 0 then
							t_insert(stats, s_format("^8Life recovered: ^7%d ^8(^7%d^8 instantly, plus ^7%d ^8over^7 %.2fs^8, and an additional ^7%d ^8over subsequent ^7%.2fs^8)",
									inst + grad + totalAdditionalAmount, inst, grad + additionalGrad, lifeDur, leftoverAmount, leftoverDur))
						else
							lifeDur = 0
							t_insert(stats, s_format("^8Life recovered: ^7%d ^8(^7%d^8 instantly, and an additional ^7%d ^8over ^7%.2fs^8)",
									inst + totalAdditionalAmount, inst, totalAdditionalAmount, 10))
						end
					else
						t_insert(stats, s_format("^8Life recovered: ^7%d ^8(^7%d ^8over ^7%.2fs^8, and an additional ^7%d ^8over subsequent ^7%.2fs^8)",
						grad + totalAdditionalAmount, grad + additionalGrad, lifeDur, leftoverAmount, leftoverDur))
					end
				else
					if inst > 0 and grad > 0 then
						t_insert(stats, s_format("^8Life recovered: ^7%d ^8(^7%d^8 instantly, plus ^7%d ^8over^7 %.2fs^8)", inst + grad, inst, grad, lifeDur))
					-- modifiers to recovery amount or duration
					elseif inst + grad ~= flaskData.lifeTotal or (inst == 0 and lifeDur ~= flaskData.duration) then
						if inst > 0 then
							lifeDur = 0
							t_insert(stats, s_format("^8Life recovered: ^7%d ^8instantly", inst))
						elseif grad > 0 then
							t_insert(stats, s_format("^8Life recovered: ^7%d ^8over ^7%.2fs", grad, lifeDur))
						end
					end
				end
				if modDB:Flag(nil, "LifeFlaskAppliesToEnergyShield") then
					if inst > 0 and esGrad > 0 then
						t_insert(stats, s_format("^8Energy Shield recovered: ^7%d ^8(^7%d^8 instantly, plus ^7%d ^8over^7 %.2fs^8)", inst + esGrad, inst, esGrad, lifeDur))
					elseif inst > 0 and esGrad == 0 then
						t_insert(stats, s_format("^8Energy Shield recovered: ^7%d ^8instantly", inst))
					elseif inst == 0 and esGrad > 0 then
						t_insert(stats, s_format("^8Energy Shield recovered: ^7%d ^8over ^7%.2fs", esGrad, lifeDur))
					end
				end
			end
			if item.base.flask.mana then
				local manaInc = modDB:Sum("INC", nil, "FlaskManaRecovery")
				local manaRateInc = modDB:Sum("INC", nil, "FlaskManaRecoveryRate")
				local instantPerc = flaskData.instantPerc + modDB:Sum("BASE", nil, "ManaFlaskInstantRecovery")
				local inst = flaskData.manaBase * instantPerc / 100 * (1 + manaInc / 100) * (1 + effectInc / 100)
				local base = flaskData.manaBase * (1 - instantPerc / 100) * (1 + manaInc / 100) * (1 + effectInc / 100) * (1 + durInc / 100)
				local grad = base * output.ManaRecoveryRateMod
				local lifeGrad = base * output.LifeRecoveryRateMod
				manaDur = flaskData.duration * (1 + durInc / 100) / (1 + rateInc / 100) / (1 + manaRateInc / 100)

				if inst > 0 and grad > 0 then
					t_insert(stats, s_format("^8Mana recovered: ^7%d ^8(^7%d^8 instantly, plus ^7%d ^8over^7 %.2fs^8)", inst + grad, inst, grad, manaDur))
				elseif inst + grad ~= flaskData.manaTotal or (inst == 0 and manaDur ~= flaskData.duration) then
					if inst > 0 then
						manaDur = 0
						t_insert(stats, s_format("^8Mana recovered: ^7%d ^8instantly", inst))
					elseif grad > 0 then
						t_insert(stats, s_format("^8Mana recovered: ^7%d ^8over ^7%.2fs", grad, manaDur))
					end
				end
				if modDB:Flag(nil, "ManaFlaskAppliesToLife") then
					if lifeGrad > 0 then
						t_insert(stats, s_format("^8Life recovered: ^7%d ^8over ^7%.2fs", lifeGrad, manaDur))
					end
				end
			end
		else
			if durInc ~= 0 then
				t_insert(stats, s_format("^8Flask effect duration: ^7%.1f0s", flaskData.duration * (1 + durInc / 100)))
			end
		end
		local effectMod = 1 + (flaskData.effectInc + effectInc) / 100
		if effectMod ~= 1 then
			t_insert(stats, s_format("^8Flask effect modifier: ^7%+d%%", effectMod * 100 - 100))
		end
		local usedInc = modDB:Sum("INC", nil, "FlaskChargesUsed")
		if usedInc ~= 0 then
			local used = m_floor(flaskData.chargesUsed * (1 + usedInc / 100))
			t_insert(stats, s_format("^8Charges used: ^7%d ^8of ^7%d ^8(^7%d ^8uses)", used, flaskData.chargesMax, m_floor(flaskData.chargesMax / used)))
		end
		local gainMod = flaskData.gainMod * (1 + modDB:Sum("INC", nil, "FlaskChargesGained") / 100)
		if gainMod ~= 1 then
			t_insert(stats, s_format("^8Charge gain modifier: ^7%+d%%", gainMod * 100 - 100))
		end

		-- charge generation
		local chargesGenerated = modDB:Sum("BASE", nil, "FlaskChargesGenerated")
		if item.base.flask.life then
			chargesGenerated = chargesGenerated + modDB:Sum("BASE", nil, "LifeFlaskChargesGenerated")
		end
		if item.base.flask.mana then
			chargesGenerated = chargesGenerated + modDB:Sum("BASE", nil, "ManaFlaskChargesGenerated")
		end
		if not item.base.flask.mana and not item.base.flask.life then
			chargesGenerated = chargesGenerated + modDB:Sum("BASE", nil, "UtilityFlaskChargesGenerated")
		end
		local chargesGeneratedOnWardBreak = 0
		if item.baseName == "Iron Flask" then
			chargesGeneratedOnWardBreak = chargesGeneratedOnWardBreak + modDB:Sum("BASE", nil, "IronFlaskChargesGeneratedOnWardBreak")
		end

		local chargesGeneratedPerFlask = modDB:Sum("BASE", nil, "FlaskChargesGeneratedPerEmptyFlask")
		local emptyFlaskSlots = 0
		for slotName, slot in pairs(self.slots) do
			if slotName:find("^Flask") ~= nil and slot.selItemId == 0 then
				emptyFlaskSlots = emptyFlaskSlots + 1
			end
		end
		chargesGeneratedPerFlask = chargesGeneratedPerFlask * emptyFlaskSlots
		chargesGenerated = chargesGenerated * gainMod
		chargesGeneratedOnWardBreak = chargesGeneratedOnWardBreak * gainMod
		chargesGeneratedPerFlask = chargesGeneratedPerFlask * gainMod

		local totalChargesGenerated = chargesGenerated + chargesGeneratedPerFlask
		if totalChargesGenerated > 0 then
			t_insert(stats, s_format("^8Charges generated: ^7%.2f^8 per second", totalChargesGenerated))
		end
		if chargesGeneratedOnWardBreak > 0 then
			t_insert(stats, s_format("^8Charges generated on Ward Break: ^7%.2f", chargesGeneratedOnWardBreak))
		end

		local chanceToNotConsumeCharges = m_min(modDB:Sum("BASE", nil, "FlaskChanceNotConsumeCharges"), 100)
		if chanceToNotConsumeCharges ~= 0 then
			t_insert(stats, s_format("^8Chance to not consume charges: ^7%d%%", chanceToNotConsumeCharges))
		end

		-- flask uptime
		local hasUptime = not item.base.flask.life and not item.base.flask.mana
		local flaskDuration = flaskData.duration * (1 + durInc / 100)

		if item.base.flask.life and (flaskData.lifeEffectNotRemoved or modDB:Flag(nil, "LifeFlaskEffectNotRemoved")) then
			hasUptime = true
			flaskDuration = lifeDur
		elseif item.base.flask.mana and (flaskData.manaEffectNotRemoved or modDB:Flag(nil, "ManaFlaskEffectNotRemoved")) then
			hasUptime = true
			flaskDuration = manaDur
		end

		if hasUptime then
			local flaskChargesUsed = flaskData.chargesUsed * (1 + usedInc / 100)
			if flaskChargesUsed > 0 and flaskDuration > 0 then
				local per3Duration = flaskDuration - (flaskDuration % 3)
				local per5Duration = flaskDuration - (flaskDuration % 5)
				local minimumChargesGenerated = per3Duration * chargesGenerated + per5Duration * chargesGeneratedPerFlask
				local percentageMin = m_min(minimumChargesGenerated / flaskChargesUsed * 100, 100)
				if percentageMin < 100 and chanceToNotConsumeCharges < 100 then
					local averageChargesGenerated = (chargesGenerated + chargesGeneratedPerFlask) * flaskDuration
					local averageChargesUsed = flaskChargesUsed * (100 - chanceToNotConsumeCharges) / 100
					local percentageAvg = m_min(averageChargesGenerated / averageChargesUsed * 100, 100)
					t_insert(stats, s_format("^8Flask uptime: ^7%d%%^8 average, ^7%d%%^8 minimum", percentageAvg, percentageMin))
				else
					t_insert(stats, s_format("^8Flask uptime: ^7100%%^8"))
				end
			end
		end

		if stats[1] then
			tooltip:AddLine(14, "^7Effective flask stats:")
			for _, stat in ipairs(stats) do
				tooltip:AddLine(14, stat)
			end
		end
		local output = calcFunc({ toggleFlask = item })
		local header
		if self.build.calcsTab.mainEnv.flasks[item] then
			header = "^7Deactivating this flask will give you:"
		else
			header = "^7Activating this flask will give you:"
		end
		self.build:AddStatComparesToTooltip(tooltip, calcBase, output, header)
	elseif base.tincture then
		-- Special handling for tinctures
		local stats = { }
		local tinctureData = item.tinctureData
		local modDB = self.build.calcsTab.mainEnv.modDB
		local output = self.build.calcsTab.mainOutput
		local effectInc = modDB:Sum("INC", { actor = "player" }, "TinctureEffect")

		if item.rarity == "MAGIC" then
			effectInc = effectInc + modDB:Sum("INC", { actor = "player" }, "MagicTinctureEffect")
		end
		local effectMod = (1 + (tinctureData.effectInc + effectInc) / 100) * (1 + (item.quality or 0) / 100)
		if effectMod ~= 1 then
			t_insert(stats, s_format("^8Tincture effect modifier: ^7%+d%%", effectMod * 100 - 100))
		end
		t_insert(stats, s_format("^8Mana Burn Inflicted Every Second: ^7%.2f", 1 / (tinctureData.manaBurn / (1 + modDB:Sum("INC", { actor = "player" }, "TinctureManaBurnRate") / 100) / (1 + modDB:Sum("MORE", { actor = "player" }, "TinctureManaBurnRate") / 100))))
		local TincturesNotInflictManaBurn = m_min(modDB:Sum("BASE", nil, "TincturesNotInflictManaBurn"), 100)
		if TincturesNotInflictManaBurn ~= 0 then
			t_insert(stats, s_format("^8Chance to not inflict Mana Burn: ^7%d%%", TincturesNotInflictManaBurn))
		end
		t_insert(stats, s_format("^8Tincture Cooldown when deactivated: ^7%.2f^8 seconds", base.tincture.cooldown / (1 + (modDB:Sum("INC", { actor = "player" }, "TinctureCooldownRecovery") + tinctureData.cooldownInc) / 100)))

		if stats[1] then
			tooltip:AddLine(14, "^7Effective tincture stats:")
			for _, stat in ipairs(stats) do
				tooltip:AddLine(14, stat)
			end
		end
		local output = calcFunc({ toggleTincture = item })
		local header
		if self.build.calcsTab.mainEnv.tinctures[item] then
			header = "^7Deactivating this tincture will give you:"
		else
			header = "^7Activating this tincture will give you:"
		end
		self.build:AddStatComparesToTooltip(tooltip, calcBase, output, header)
	else
		self:UpdateSockets()
		-- Build sorted list of slots to compare with
		local compareSlots = { }
		for slotName, slot in pairs(self.slots) do
			if self:IsItemValidForSlot(item, slotName) and not slot.inactive and (not slot.weaponSet or slot.weaponSet == (self.activeItemSet.useSecondWeaponSet and 2 or 1)) and slot:IsShown() then
				t_insert(compareSlots, slot)
			end
		end

		tooltip:AddLine(14, colorCodes.TIP .. "Tip: Press Ctrl+D"..itemTabHint.." to disable the display of stat differences.")

		local function getReplacedItemAndOutput(compareSlot)
			local selItem = self.items[compareSlot.selItemId]
			local override = { repSlotName = compareSlot.slotName, repItem = item ~= selItem and item or nil }
			if compareSlot.nodeId and (itemChangesPassiveTree(selItem) or itemChangesPassiveTree(item)) then
				override.spec = buildSpecForJewelComparison(self, compareSlot, override.repItem)
			end
			local output = calcFunc(override)
			return selItem, output
		end
		local function addCompareForSlot(compareSlot, selItem, output)
			if not selItem or not output then
				selItem, output = getReplacedItemAndOutput(compareSlot)
			end
			local header
			if item == selItem then
				header = "^7Removing this item from "..compareSlot.label.." will give you:"
			else
				header = string.format("^7Equipping this item in %s will give you:%s", compareSlot.label or compareSlot.slotName, selItem and "\n(replacing "..colorCodes[selItem.rarity]..selItem.name.."^7)" or "")
			end
			self.build:AddStatComparesToTooltip(tooltip, calcBase, output, header)
		end

		-- if we have a specific slot to compare to, and the user has "Show
		-- tooltips only for affected slots" checked, we can just compare that
		-- one slot
		if main.slotOnlyTooltips and slot then
			slot = type(slot) ~= "string" and slot or self.slots[slot]
			if slot then addCompareForSlot(slot) end
			return
		end


		local slots = {}
		local isUnique = item.rarity == "UNIQUE" or item.rarity == "RELIC"
		local currentSameUniqueCount = 0
		for _, compareSlot in ipairs(compareSlots) do
			local selItem, output = getReplacedItemAndOutput(compareSlot)
			local isSameUnique = isUnique and selItem and item.name == selItem.name
			if isUnique and isSameUnique and item.limit then
				currentSameUniqueCount = currentSameUniqueCount + 1
			end
			table.insert(slots,
				{ selItem = selItem, output = output, compareSlot = compareSlot, isSameUnique = isSameUnique })
		end

		-- limited uniques: only compare to slots with the same item if more don't fit
		if currentSameUniqueCount == item.limit then
			for _, slotEntry in ipairs(slots) do
				if slotEntry.isSameUnique then
					addCompareForSlot(slotEntry.compareSlot, slotEntry.selItem, slotEntry.output)
				end
			end
			return
		end


		-- either the same unique or same base type
		local function similar(compareItem, sameUnique)
			-- empty slot
			if not compareItem then return 0 end

			local sameBaseType = not isUnique
				and compareItem.rarity ~= "UNIQUE" and compareItem.rarity ~= "RELIC"
				and item.base.type == compareItem.base.type
				and item.base.subType == compareItem.base.subType
			if sameBaseType or sameUnique then
				return 1
			else
				return 0
			end
		end
		-- sort by:
		-- 1. empty sockets
		-- 2. same base group jewel or unique
		-- 3. DPS
		-- 4. EHP
		local function sortFunc(a, b)
			if a == b then return end

			local aParams = { a.compareSlot.selItemId == 0 and 1 or 0, similar(a.selItem, a.isSameUnique), a.output.FullDPS, a.output.CombinedDPS, a.output.TotalEHP, a.compareSlot.label, a.compareSlot.slotName }
			local bParams = { b.compareSlot.selItemId == 0 and 1 or 0, similar(b.selItem, b.isSameUnique), b.output.FullDPS, b
				.output.CombinedDPS, b.output.TotalEHP, b.compareSlot.label, b.compareSlot.slotName }
			for i = 1, #aParams do
				if aParams[i] == nil or bParams[i] == nil then
					-- continue
				elseif aParams[i] > bParams[i] then
					return true
				elseif aParams[i] < bParams[i] then
					return false
				end
			end
			return false
		end
		table.sort(slots, sortFunc)

		for _, slotEntry in ipairs(slots) do
			addCompareForSlot(slotEntry.compareSlot, slotEntry.selItem, slotEntry.output)
		end

	end

	if launch.devModeAlt then
		-- Modifier debugging info
		tooltip:AddSeparator(10)
		for _, mod in ipairs(modList) do
			tooltip:AddLine(14, "^7"..modLib.formatMod(mod))
		end
	end
end

function ItemsTabClass:CreateUndoState()
	local state = { }
	state.activeItemSetId = self.activeItemSetId
	state.items = { }
	for k, v in pairs(self.items) do
		state.items[k] = copyTableSafe(self.items[k], true, true)
	end
	state.itemOrderList = copyTable(self.itemOrderList)
	state.slotSelItemId = { }
	for slotName, slot in pairs(self.slots) do
		state.slotSelItemId[slotName] = slot.selItemId
	end
	state.itemSets = copyTableSafe(self.itemSets)
	state.itemSetOrderList = copyTable(self.itemSetOrderList)
	return state
end

function ItemsTabClass:RestoreUndoState(state)
	self.items = state.items
	wipeTable(self.itemOrderList)
	for k, v in pairs(state.itemOrderList) do
		self.itemOrderList[k] = v
	end
	for slotName, selItemId in pairs(state.slotSelItemId) do
		self.slots[slotName]:SetSelItemId(selItemId)
	end
	self.itemSets = state.itemSets
	wipeTable(self.itemSetOrderList)
	for k, v in pairs(state.itemSetOrderList) do
		self.itemSetOrderList[k] = v
	end
	self.activeItemSetId = state.activeItemSetId
	self.activeItemSet = self.itemSets[self.activeItemSetId]
	self:PopulateSlots()
end
