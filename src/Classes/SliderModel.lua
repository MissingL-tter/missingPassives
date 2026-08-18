-- Path of Building
--
-- Class: Slider Model
-- Plain-data 0-1 value model with optional division count. Replaces the
-- slider controls for engine-side state that a UI can drive.
--
local m_max = math.max
local m_min = math.min
local m_ceil = math.ceil

local SliderModelClass = newClass("SliderModel", function(self, changeFunc)
	self.val = 0
	self.changeFunc = changeFunc
end)

function SliderModelClass:SetVal(newVal)
	newVal = m_max(0, m_min(1, newVal))
	if newVal ~= self.val then
		self.val = newVal
		if self.changeFunc then
			self.changeFunc(self.val)
		end
	end
end

function SliderModelClass:GetDivVal(val)
	val = val or self.val
	if self.divCount and self.divCount > 1 then
		local divIndex = m_max(m_ceil(val * self.divCount), 1)
		return divIndex, val * self.divCount - divIndex + 1
	else
		return 1, val
	end
end

function SliderModelClass:IsShown()
	if type(self.shown) == "function" then
		return self.shown()
	end
	return self.shown ~= false
end
