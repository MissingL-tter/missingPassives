-- Path of Building
--
-- Class: Item Slot
-- Holds the state of one equipment slot: which item is selected, whether the
-- slot is active (flasks), and which items are eligible for it.
--
local t_insert = table.insert

local ItemSlotClass = newClass("ItemSlot", function(self, itemsTab, slotName, slotLabel, nodeId)
	self.itemsTab = itemsTab
	self.items = { }
	self.selItemId = 0
	self.slotName = slotName
	self.slotNum = tonumber(slotName:match("%d+$") or slotName:match("%d+"))
	self.abyssalSocketList = { }
	self.label = slotLabel or slotName
	self.nodeId = nodeId
end)

-- Whether the slot is present in the current build state; mirrors the old
-- slot control visibility rules
function ItemSlotClass:IsShown()
	if self.inactive then
		return false
	end
	local itemsTab = self.itemsTab
	if self.weaponSet == 1 then
		return not itemsTab.activeItemSet.useSecondWeaponSet
	elseif self.weaponSet == 2 then
		return itemsTab.activeItemSet.useSecondWeaponSet
	elseif self.slotName == "Graft 1" or self.slotName == "Graft 2" then
		return not not itemsTab.build.spec.treeVersion:find("3_27")
	elseif self.slotName == "Ring 3" then
		return itemsTab.build.calcsTab.mainEnv.modDB:Flag(nil, "AdditionalRingSlot")
	end
	return true
end

function ItemSlotClass:SetSelItemId(selItemId)
	if self.nodeId then
		if self.itemsTab.build.spec then
			self.itemsTab.build.spec.jewels[self.nodeId] = selItemId
			if selItemId ~= self.selItemId then
				self.itemsTab.build.spec:BuildClusterJewelGraphs()
			end
		end
	else
		self.itemsTab.activeItemSet[self.slotName].selItemId = selItemId
	end
	self.selItemId = selItemId
end

function ItemSlotClass:Populate()
	if self.nodeId and self.itemsTab.build.spec then
		self.selItemId = self.itemsTab.build.spec.jewels[self.nodeId] or 0
	end

	wipeTable(self.items)
	self.items[1] = 0
	for _, item in pairs(self.itemsTab.items) do
		if self.itemsTab:IsItemValidForSlot(item, self.slotName) then
			t_insert(self.items, item.id)
		end
	end
	if not self.selItemId or not self.itemsTab.items[self.selItemId] or not self.itemsTab:IsItemValidForSlot(self.itemsTab.items[self.selItemId], self.slotName) then
		self:SetSelItemId(0)
	end

	-- Update Abyssal Sockets
	local abyssalSocketCount = 0
	if self.selItemId > 0 then
		local selItem = self.itemsTab.items[self.selItemId]
		abyssalSocketCount = selItem.abyssalSocketCount or 0
	end
	for i, abyssalSocket in ipairs(self.abyssalSocketList) do
		abyssalSocket.inactive = i > abyssalSocketCount
		if abyssalSocket.inactive then
			-- this can be inconvenient, but otherwise it is possible to double
			-- equip jewels by moving the jewel while the socket is inactive
			abyssalSocket:SetSelItemId(0)
		end
	end
end
