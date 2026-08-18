-- Path of Building
--
-- Module: Party Tab
-- Party buff state for the current build.
--
local pairs = pairs
local ipairs = ipairs
local t_insert = table.insert
local m_max = math.max

local PartyTabClass = newClass("PartyTab", function(self, build)
	self.build = build

	self.actor = { Aura = { }, Curse = { }, Warcry = { }, Link = { }, modDB = new("ModDB"), output = { } }
	self.actor.modDB.actor = self.actor
	self.enemyModList = new("ModList")
	self.buffExports = { }
	self.enableExportBuffs = false

	-- Imported buff text blocks
	self.partyMemberStats = ""
	self.auras = ""
	self.curses = ""
	self.warcries = ""
	self.links = ""
	self.enemyConds = ""
	self.enemyModLines = ""

	self.destination = "All"
	self.append = false
	self.showAdvanceTools = false

	self.lastContent = {
		Aura = "",
		Curse = "",
		Warcry = "",
		Link = "",
		EnemyCond = "",
		EnemyMods = "",
		EnableExportBuffs = false,
		showAdvancedTools = false,
	}
end)

function PartyTabClass:Load(xml, fileName)
	for _, node in ipairs(xml) do
		if node.elem == "ImportedBuffs" then
			if not node.attrib.name then
				ConPrintf("missing name")
			elseif node.attrib.name == "PartyMemberStats" then
				self.partyMemberStats = node[1] or ""
				self:ParseBuffs(self.actor["modDB"], self.partyMemberStats, "PartyMemberStats", self.actor["output"])
			elseif node.attrib.name == "Aura" then
				self.auras = node[1] or ""
				self:ParseBuffs(self.actor["Aura"], self.auras, "Aura")
			elseif node.attrib.name == "Curse" then
				self.curses = node[1] or ""
				self:ParseBuffs(self.actor["Curse"], self.curses, "Curse")
			elseif node.attrib.name == "Warcry Skills" then
				self.warcries = node[1] or ""
				self:ParseBuffs(self.actor["Warcry"], self.warcries, "Warcry")
			elseif node.attrib.name == "Link Skills" then
				self.links = node[1] or ""
				self:ParseBuffs(self.actor["Link"], self.links, "Link")
			elseif node.attrib.name == "EnemyConditions" then
				self.enemyConds = node[1] or ""
				self:ParseBuffs(self.enemyModList, self.enemyConds, "EnemyConditions")
			elseif node.attrib.name == "EnemyMods" then
				self.enemyModLines = node[1] or ""
				self:ParseBuffs(self.enemyModList, self.enemyModLines, "EnemyMods")
			end
		elseif node.elem == "ExportedBuffs" then
			if not node.attrib.name then
				ConPrintf("missing name")
			end
		end
	end
	self.lastContent.PartyMemberStats = self.partyMemberStats
	self.lastContent.Aura = self.auras
	self.lastContent.Curse = self.curses
	self.lastContent.Warcry = self.warcries
	self.lastContent.Link = self.links
	self.lastContent.EnemyCond = self.enemyConds
	self.lastContent.EnemyMods = self.enemyModLines
	self.lastContent.EnableExportBuffs = self.enableExportBuffs

	self.destination = xml.attrib.destination or "All"
	self.append = xml.attrib.append == "true"
	self.showAdvanceTools = xml.attrib.ShowAdvanceTools == "true"

	self.lastContent.showAdvancedTools = self.showAdvanceTools
end

function PartyTabClass:Save(xml)
	local child
	if self.partyMemberStats and self.partyMemberStats ~= "" then
		child = { elem = "ImportedBuffs", attrib = { name = "PartyMemberStats" } }
		t_insert(child, self.partyMemberStats)
		t_insert(xml, child)
	end
	if self.auras and self.auras ~= "" then
		child = { elem = "ImportedBuffs", attrib = { name = "Aura" } }
		t_insert(child, self.auras)
		t_insert(xml, child)
	end
	if self.curses and self.curses ~= "" then
		child = { elem = "ImportedBuffs", attrib = { name = "Curse" } }
		t_insert(child, self.curses)
		t_insert(xml, child)
	end
	if self.warcries and self.warcries ~= "" then
		child = { elem = "ImportedBuffs", attrib = { name = "Warcry Skills" } }
		t_insert(child, self.warcries)
		t_insert(xml, child)
	end
	if self.links and self.links ~= "" then
		child = { elem = "ImportedBuffs", attrib = { name = "Link Skills" } }
		t_insert(child, self.links)
		t_insert(xml, child)
	end
	if self.enemyConds and self.enemyConds ~= "" then
		child = { elem = "ImportedBuffs", attrib = { name = "EnemyConditions" } }
		t_insert(child, self.enemyConds)
		t_insert(xml, child)
	end
	if self.enemyModLines and self.enemyModLines ~= "" then
		child = { elem = "ImportedBuffs", attrib = { name = "EnemyMods" } }
		t_insert(child, self.enemyModLines)
		t_insert(xml, child)
	end
	local exportString = self:exportBuffs("PlayerMods")
	if exportString ~= "" then
		child = { elem = "ExportedBuffs", attrib = { name = "PartyMemberStats" } }
		t_insert(child, exportString)
		t_insert(xml, child)
	end
	exportString = self:exportBuffs("Aura")
	if exportString ~= "" then
		child = { elem = "ExportedBuffs", attrib = { name = "Aura" } }
		t_insert(child, exportString)
		t_insert(xml, child)
	end
	exportString = self:exportBuffs("Curse")
	if exportString ~= "" then
		child = { elem = "ExportedBuffs", attrib = { name = "Curse" } }
		t_insert(child, exportString)
		t_insert(xml, child)
	end
	exportString = self:exportBuffs("Warcry")
	if exportString ~= "" then
		child = { elem = "ExportedBuffs", attrib = { name = "Warcry Skills" } }
		t_insert(child, exportString)
		t_insert(xml, child)
	end
	exportString = self:exportBuffs("Link")
	if exportString ~= "" then
		child = { elem = "ExportedBuffs", attrib = { name = "Link Skills" } }
		t_insert(child, exportString)
		t_insert(xml, child)
	end
	exportString = self:exportBuffs("EnemyConditions")
	if exportString ~= "" then
		child = { elem = "ExportedBuffs", attrib = { name = "EnemyConditions" } }
		t_insert(child, exportString)
		t_insert(xml, child)
	end
	exportString = self:exportBuffs("EnemyMods")
	if exportString ~= "" then
		child = { elem = "ExportedBuffs", attrib = { name = "EnemyMods" } }
		t_insert(child, exportString)
		t_insert(xml, child)
	end
	self.lastContent.PartyMemberStats = self.partyMemberStats
	self.lastContent.Aura = self.auras
	self.lastContent.Curse = self.curses
	self.lastContent.Warcry = self.warcries
	self.lastContent.Link = self.links
	self.lastContent.EnemyCond = self.enemyConds
	self.lastContent.EnemyMods = self.enemyModLines
	self.lastContent.EnableExportBuffs = self.enableExportBuffs
	xml.attrib = {
		destination = self.destination,
		append = tostring(self.append),
		ShowAdvanceTools = tostring(self.showAdvanceTools)
	}
	self.lastContent.showAdvancedTools = self.showAdvanceTools
end

function PartyTabClass:ParseBuffs(list, buf, buffType, label)
	if buffType == "EnemyConditions" then
		for line in buf:gmatch("([^\n]*)\n?") do
			if line ~= "" then
				list:NewMod(line:gsub("Condition:", "Condition:Party:"), "FLAG", true, "Party")
			end
		end
	elseif buffType == "EnemyMods" then
		local enemyModList = {}
		local currentName
		for line in buf:gmatch("([^\n]*)\n?") do
			if not line:find("|") then
				currentName = line
				if label and currentName ~= "" then
					enemyModList[currentName] = enemyModList[currentName] or {}
				end
			else
				local mod = modLib.parseFormattedSourceMod(line)
				if mod then
					mod.source = "Party"..mod.source
					list:AddMod(mod)
					if label then
						t_insert(enemyModList[currentName], {mod.value, mod.type})
					end
				end
			end
		end
	elseif buffType == "PartyMemberStats" then
		if not list then
		else
			for line in buf:gmatch("([^\n]*)\n?") do
				if line:find("=") then
					-- label is output for this type, as a special case
					local k1, k2, v = line:match("^([%w ]-%w+)%.([%w ]-%w+)=(.+)$")
					if k1 then
						if type(label[k1]) ~= "table" then
							label[k1] = { }
						end
						label[k1][k2] = tonumber(v)
					elseif line:match("|") then
						local k, tags, v = line:match("([%w ]-%w+)|(.+)=(.+)")
						v = tonumber(v)
						for tag in tags:gmatch("([^|]*)|?") do
							if tag == "percent" then
								v = v / 100
							elseif tag == "max" then
								v = m_max(v, label[k] or 1)
							end
						end
						label[k] = v
					else
						local k, v = line:match("([%w ]-%w+)=(.+)")
						label[k] = tonumber(v)
					end
				elseif line ~= "" then
					list:NewMod(line, "FLAG", true, "Party")
				end
			end
		end
	else
		local mode = "Name"
		if buffType == "Curse" then
			mode = "CurseLimit"
		end
		local currentName
		local currentEffect
		local isMark
		local currentModType = (buffType == "Link") and "Link" or (buffType == "Warcry") and "Warcry" or "Unknown"
		for line in buf:gmatch("([^\n]*)\n?") do
			if line ~= "---" and line:match("%-%-%-") then
				-- comment but not divider, skip the line
			elseif mode == "CurseLimit" and line ~= "" then
				list.limit = tonumber(line)
				mode = "Name"
			elseif mode == "Name" and line ~= "" then
				currentName = line:gsub("_Debuff", "")
				currentEffect = 0
				if line == "extraAura" or line == "otherEffects" then
					currentModType = line
					mode = "Stats"
				else
					mode = "Effect"
					currentModType = "Unknown"
				end
			elseif mode == "Effect" then
				currentEffect = tonumber(line)
				if buffType == "Curse" then
					mode = "isMark"
				else
					mode = "Stats"
				end
			elseif mode == "isMark" then
				isMark = line=="true"
				mode = "Stats"
			elseif line == "---" then
				mode = "Name"
			else
				if line:find("|") and currentName ~= "SKIP" and not line:find("MinionModifier|LIST") then
					if currentModType == "otherEffects" then
						currentName, currentEffect, line = line:match("([%w ']-%w+)|(%w+)|(.+)")
					end
					local mod = modLib.parseFormattedSourceMod(line)
					if mod then
						for _, tag in ipairs(mod) do
							if tag.type == "GlobalEffect" and currentModType ~= "Link" and currentModType ~= "Warcry" and currentModType ~= "otherEffects" then
								currentModType = tag.effectType
							end
						end
						list[currentModType] = list[currentModType] or {}
						local listElement = list[currentModType]
						if currentName:sub(1,4) == "Vaal" then
							list[currentModType]["Vaal"] = list[currentModType]["Vaal"] or {}
							listElement = list[currentModType]["Vaal"]
						end
						if not listElement[currentName] then
							listElement[currentName] = {
								modList = new("ModList"),
								effectMult = currentEffect
							}
							if isMark then
								listElement[currentName].isMark = true
							end
						elseif listElement[currentName].effectMult ~= currentEffect then
							if listElement[currentName].effectMult < currentEffect then
								listElement[currentName] = {
									modList = new("ModList"),
									effectMult = currentEffect
								}
							else
								currentName = "SKIP"
							end
						end
						if currentName ~= "SKIP" then
							if mod.source:match("Item") then
								local oldItem
								oldItem, mod.source = mod.source:match("Item:(%d+):(.+)")
								mod.source = "Party - "..mod.source
							end
							if mod.source:match("Skill") then
								local skillId = mod.source:match("Skill:(.+)")
								if not data.skills[skillId] then
									local minimisedName = currentName:gsub(" %l",string.upper):gsub(" ","")
									if data.skills[minimisedName] then
										mod.source = "Skill:"..minimisedName
									else
										mod.source = skillId
									end
								end
							end
							if buffType == "Link" then
								mod.name = mod.name:gsub("Parent", "PartyMember")
								for _, modTag in ipairs(mod) do
									if modTag.actor and modTag.actor == "parent" then
										modTag.actor = "partyMembers"
									end
								end
							end
							listElement[currentName].modList:AddMod(mod)
						end
					end
				end
			end
		end
	end
end

function PartyTabClass:setBuffExports(buffExports)
	if not self.enableExportBuffs then
		return
	end
	wipeTable(self.buffExports)
	self.buffExports = copyTable(buffExports, true)
end

function PartyTabClass:exportBuffs(buffType)
	if not self.enableExportBuffs or not self.buffExports or not self.buffExports[buffType] then
		return ""
	end
	if self.buffExports[buffType].ConvertedToText then
		return self.buffExports[buffType].string
	end
	local buf = ((buffType == "Curse") and ("--- Curse Limit ---\n" .. tostring(self.buffExports["CurseLimit"]))) or ""
	for buffName, buff in pairs(self.buffExports[buffType]) do
		if buffName ~= "extraAura" or #buff.modList > 0 then
			if #buf > 0 then
				buf = buf.."\n"
			end
			buf = buf..buffName
			if buffType == "Curse" then
				buf = buf.."\n"..tostring(buff.effectMult * 100)
				if buff.isMark then
					buf = buf.."\ntrue"
				else
					buf = buf.."\nfalse"
				end
			elseif buffType == "Link" or buffType == "Warcry" or buffType == "Aura" and buffName ~= "extraAura" and buffName ~= "otherEffects" then
				buf = buf.."\n"..tostring(buff.effectMult * 100)
			end
			if buffType == "Aura" and buffName == "otherEffects" then
				for innerBuffName, innerBuff in pairs(buff) do
					for _, mod in ipairs(innerBuff.modList) do
						buf = buf.."\n"..innerBuffName.."|"..tostring(innerBuff.effectMult * 100).."|"..modLib.formatSourceMod(mod)
					end
				end
				buf = buf.."\n---"
			elseif buffType == "Curse" or buffType == "Aura" or buffType == "Warcry" or buffType == "Link" then
				for _, mod in ipairs(buff.modList) do
					buf = buf.."\n"..modLib.formatSourceMod(mod)
				end
				buf = buf.."\n---"
			elseif buffType == "EnemyMods" then
				if buff.MultiStat then
					for _, buffInner in ipairs(buff) do
						buf = buf.."\n"..modLib.formatSourceMod(buffInner)
					end
				else
					buf = buf.."\n"..modLib.formatSourceMod(buff)
				end
			end
		end
	end
	wipeTable(self.buffExports[buffType])
	self.buffExports[buffType] = { ConvertedToText = true }
	self.buffExports[buffType].string = buf
	return buf
end
