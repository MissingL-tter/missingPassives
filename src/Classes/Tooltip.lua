-- Path of Building
--
-- Class: Tooltip
-- Structured text block describing an item, node, or comparison. Holds lines
-- of sized, coloured text; consumers render or read them as needed.
--
local ipairs = ipairs
local t_insert = table.insert
local s_gmatch = string.gmatch

local TooltipClass = newClass("Tooltip", function(self)
	self.lines = { }
	self.blocks = { }
	self.childTooltips = nil
	self:Clear()
end)

function TooltipClass:Clear(clearUpdateParams)
	wipeTable(self.lines)
	wipeTable(self.blocks)
	if self.updateParams and clearUpdateParams then
		wipeTable(self.updateParams)
	end
	---@type string|boolean
	self.tooltipHeader = false
	self.titleYOffset = 0
	self.recipe = nil
	self.center = false
	self.maxWidth = nil
	---@type string|[number, number, number]
	self.color = { 0.5, 0.3, 0 }
	t_insert(self.blocks, { height = 0 })
end

function TooltipClass:CheckForUpdate(...)
	local doUpdate = false
	if not self.updateParams then
		self.updateParams = { }
	end

	for i = 1, select('#', ...) do
		local temp = select(i, ...)
		if self.updateParams[i] ~= temp then
			self.updateParams[i] = temp
			doUpdate = true
		end
	end
	if doUpdate or self.updateParams.notSupportedModTooltips ~= main.notSupportedModTooltips then
		self.updateParams.notSupportedModTooltips = main.notSupportedModTooltips
		self:Clear()
		return true
	end
end

function TooltipClass:AddLine(size, text, font, modLine, background)
	if text then
		local fontToUse
		if main.showFlavourText then
			fontToUse = font or "VAR"
		else
			fontToUse = "VAR"
		end
		for line in s_gmatch(text .. "\n", "([^\n]*)\n") do
			if line:match("^.*(Equipping)") == "Equipping" or line:match("^.*(Removing)") == "Removing" then
				t_insert(self.blocks, { height = size + 2})
			else
				self.blocks[#self.blocks].height = self.blocks[#self.blocks].height + size + 2
			end
			t_insert(self.lines, { size = size, text = line, block = #self.blocks, font = fontToUse, center = self.center, modLine = modLine, background = background })
		end
	end
end

function TooltipClass:SetRecipe(recipe)
	self.recipe = recipe
end

function TooltipClass:AddSeparator(size)
	size = size or 10

	local lastLine = self.lines[#self.lines]
	if lastLine and lastLine.separatorHeader then
		-- Prevent back-to-back separator lines
		return
	end

	-- The header rarity styles the separator; its presence also enables the
	-- back-to-back check above, mirroring the old image-based separators
	local separatorHeader
	if self.tooltipHeader then
		separatorHeader = tostring(self.tooltipHeader):upper()
	end

	local lastBlock = lastLine and lastLine.block or 1
	t_insert(self.lines, {
		separator = true,
		separatorHeader = separatorHeader,
		size = size,
		block = lastBlock,
	})
end
