-- Path of Building
--
-- Class: Selector
-- Plain-data selection model: a list of options with a selected index and an
-- optional callback fired when the selection changes. Replaces the dropdown
-- controls for engine-side state that a UI can drive.
--
local m_max = math.max
local m_min = math.min

local SelectorClass = newClass("Selector", function(self, list, selFunc)
	self.list = list or { }
	self.selIndex = 1
	self.selFunc = selFunc
end)

function SelectorClass:SetList(list)
	self.list = list or { }
	self.selIndex = m_max(1, m_min(#self.list, self.selIndex))
end

function SelectorClass:SelByValue(value, key)
	for index, listVal in ipairs(self.list) do
		if type(listVal) == "table" then
			if listVal[key] == value then
				self.selIndex = index
				return
			end
		else
			if listVal == value then
				self.selIndex = index
				return
			end
		end
	end
end

function SelectorClass:GetSelValueByKey(key)
	return self.list[self.selIndex][key]
end

function SelectorClass:GetSelValue()
	return self.list[self.selIndex]
end

function SelectorClass:SetSel(newSel, noCallSelFunc)
	newSel = m_max(1, m_min(#self.list, newSel))
	if newSel and newSel ~= self.selIndex then
		self.selIndex = newSel
		if not noCallSelFunc and self.selFunc then
			self.selFunc(newSel, self.list[newSel])
		end
	end
end

function SelectorClass:IsShown()
	if type(self.shown) == "function" then
		return self.shown()
	end
	return self.shown ~= false
end

function SelectorClass:IsEnabled()
	if type(self.enabled) == "function" then
		return self.enabled()
	end
	return self.enabled ~= false
end
