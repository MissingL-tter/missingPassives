-- Path of Building
--
-- Module: Notes Tab
-- Notes for the current build.
--
local t_insert = table.insert

local NotesTabClass = newClass("NotesTab", function(self, build)
	self.build = build

	self.text = ""
	self.lastContent = ""
end)

function NotesTabClass:SetText(text)
	self.text = text or ""
	self.modFlag = (self.lastContent ~= self.text)
end

function NotesTabClass:Load(xml, fileName)
	for _, node in ipairs(xml) do
		if type(node) == "string" then
			self.text = node
		end
	end
	self.lastContent = self.text
end

function NotesTabClass:Save(xml)
	t_insert(xml, self.text)
	self.lastContent = self.text
	self.modFlag = false
end
