-- Path of Building
--
-- Module: Import Tab
-- Import/Export tab for the current build.
--
local ipairs = ipairs
local t_insert = table.insert
local t_remove = table.remove
local b_rshift = bit.rshift
local band = bit.band
local m_max = math.max
local dkjson = require "dkjson"



local influenceInfo = itemLib.influenceInfo.all

local realmList = {
	{ label = "PC",      id = "PC",   realmCode = "pc",   hostName = "https://www.pathofexile.com/", profileURL = "account/view-profile/" },
	{ label = "Xbox",    id = "XBOX", realmCode = "xbox", hostName = "https://www.pathofexile.com/", profileURL = "account/view-profile/" },
	{ label = "Sony",    id = "SONY", realmCode = "sony", hostName = "https://www.pathofexile.com/", profileURL = "account/view-profile/" },
}

local function addAccountNameControls(self)
	self.charImportMode = "GETACCOUNTNAME"
	self.charImportStatus = "Idle"

	-- Account-name character import state
	self.controls.siteAccountRealm = new("Selector", realmList)
	self.controls.siteAccountRealm:SelByValue(main.lastRealm or "PC", "id")
	self.controls.siteAccountName = { buf = main.lastAccountName or "" }
	self.controls.siteCharSelectLeague = new("Selector", nil, function(index, value)
		local realm = self.controls.siteAccountRealm:GetSelValue()
		self:BuildCharacterList(realm.realmCode, value.league, self.lastCharList, self.controls.siteCharSelect)
	end)
	self.controls.siteCharSelectLeague.list = { }
	self.controls.siteCharSelect = new("Selector")
	self.controls.siteCharImportTreeClearJewels = { state = true }
	self.controls.siteCharImportItemsClearSkills = { state = true }
	self.controls.siteCharImportItemsClearItems = { state = true }
	self.controls.siteCharImportItemsIgnoreWeaponSwap = { state = false }

	if not historyList then
		historyList = {}
		for accountName, account in pairs(main.gameAccounts) do
			t_insert(historyList, accountName)
			historyList[accountName] = true
		end
		table.sort(historyList, function(a, b)
			return a:lower() < b:lower()
		end)
	end -- don't load the list many times
end

local ImportTabClass = newClass("ImportTab", function(self, build)
	self.build = build

	if not main.api then
		main.api = new("PoEAPI", main.lastToken, main.lastRefreshToken, main.tokenExpiry)
	end

	self.controls = { }
	addAccountNameControls(self)

	self.exportParty = false

	-- validate the status of the api the first time
	if main.api.authToken then
		main.api:ValidateAuth(function(_, errMsg)
			if errMsg then
				self.oauthErrCode = string.format("Token refresh failed: %s", errMsg)
			end
		end)
	end
end)

-- Generate a build sharing code for the current build
function ImportTabClass:GenerateBuildCode()
	return common.base64.encode(Deflate(self.build:SaveDB("code"))):gsub("+","-"):gsub("/","_")
end

-- Decode and validate a build code or build-site URL, storing the decoded state on the tab
function ImportTabClass:ProcessImportCode(buf)
	self.importCodeSite = nil
	self.importCodeDetail = ""
	self.importCodeXML = nil
	self.importCodeValid = false
	self.importCodeJson = nil
	self.importCodeURL = nil

	if #buf == 0 then
		return
	end

	self.importCodeDetail = colorCodes.NEGATIVE.."Invalid input"
	local urlText = buf:gsub("^[%s?]+", ""):gsub("[%s?]+$", "") -- Quick Trim
	if urlText:match("youtube%.com/redirect%?") or urlText:match("google%.com/url%?") then
		local nested_url = urlText:gsub(".*[?&]q=([^&]+).*", "%1")
		urlText = UrlDecode(nested_url)
	end

	for j=1,#buildSites.websiteList do
		if urlText:match(buildSites.websiteList[j].matchURL) then
			self.importCodeValid = true
			self.importCodeDetail = colorCodes.POSITIVE.."URL is valid ("..buildSites.websiteList[j].label..")"
			self.importCodeSite = j
			self.importCodeURL = urlText
			return
		end
	end

	-- If we are in dev mode and the string is a json
	if launch.devMode and urlText:match("^%{.*%}$") ~= nil then
		local jsonData, _, errDecode = dkjson.decode(urlText)
		if errDecode then
			self.importCodeDetail = colorCodes.NEGATIVE.."Invalid JSON format (decode error)"
			return
		end
		if not jsonData.character then
			self.importCodeDetail = colorCodes.NEGATIVE.."Invalid JSON format (character missing)"
			return
		end
		jsonData = jsonData.character
		if not jsonData.equipment or not jsonData.passives then
			self.importCodeDetail = colorCodes.NEGATIVE.."Invalid JSON format (equipment or passives missing)"
			return
		end
		self.importCodeJson = jsonData
		self.importCodeDetail = colorCodes.POSITIVE.."JSON is valid"
		self.importCodeValid = true
		return
	end

	local xmlText = Inflate(common.base64.decode(buf:gsub("-","+"):gsub("_","/")))
	if not xmlText then
		return
	end
	self.importCodeValid = true
	self.importCodeDetail = colorCodes.POSITIVE.."Code is valid"
	self.importCodeXML = xmlText
end

-- Import the build code decoded by ProcessImportCode, into the current build or as a new unsaved build
function ImportTabClass:ImportBuildCode(intoCurrentBuild)
	if not self.importCodeValid or self.importCodeFetching then
		return
	end

	if self.importCodeJson then
		self:ImportItemsAndSkills(self.importCodeJson, true, true, false)
		self:ImportPassiveTreeAndJewels(self.importCodeJson, true)
		return
	end

	if intoCurrentBuild and self.build.dbFileName then
		self.build:Shutdown()
		self.build:Init(self.build.dbFileName, self.build.buildName, self.importCodeXML, false, self.importCodeSite and self.importCodeURL or nil)
	else
		self.build:Shutdown()
		self.build:Init(false, "Imported build", self.importCodeXML, false, self.importCodeSite and self.importCodeURL or nil)
	end
	self.build.viewMode = "TREE"
end

function ImportTabClass:Load(xml, fileName)
	self.lastRealm = xml.attrib.lastRealm
	self.lastLeague = xml.attrib.lastLeague
	self.lastAccountHash = xml.attrib.lastAccountHash
	self.importLink = xml.attrib.importLink
	self.exportParty = xml.attrib.exportParty == "true"
	self.build.partyTab.enableExportBuffs = self.exportParty
	if self.lastAccountHash and false then
		for accountName in pairs(main.gameAccounts) do
			if common.sha1(accountName) == self.lastAccountHash then
				self.controls.siteAccountName.buf = accountName
			end
		end
	end
	self.lastCharacterHash = xml.attrib.lastCharacterHash
end

function ImportTabClass:Save(xml)
	xml.attrib = {
		lastRealm = self.lastRealm,
		lastLeague = self.lastLeague,
		lastAccountHash = self.lastAccountHash,
		lastCharacterHash = self.lastCharacterHash,
		exportParty = tostring(self.exportParty),
		importLink = self.importLink
	}

	if self.build.importLink then
		xml.attrib.importLink = self.build.importLink
	end
	-- Gets rid of erroneous, potentially infinitely nested full base64 XML stored as an import link
	xml.attrib.importLink = (xml.attrib.importLink and xml.attrib.importLink:len() < 100) and xml.attrib.importLink or nil
end


function ImportTabClass:ProcessSiteJSON(json)
	local func, errMsg = loadstring("return " .. jsonToLua(json))
	if errMsg then
		return nil, errMsg
	end
	setfenv(func, {}) -- Sandbox the function just in case
	local data = func()
	if type(data) ~= "table" then
		return nil, "Return type is not a table"
	end
	return data
end

function ImportTabClass:SaveAccountHistory()
	if not historyList[self.controls.siteAccountName.buf] then
		t_insert(historyList, self.controls.siteAccountName.buf)
		historyList[self.controls.siteAccountName.buf] = true
		table.sort(historyList, function(a, b)
			return a:lower() < b:lower()
		end)
	end
end

function ImportTabClass:DownloadPassiveTree(realm)
	self.charImportMode = "IMPORTING"
	self.charImportStatus = "Retrieving character passive tree..."
	local accountName = self.controls.siteAccountName.buf
	local charSelect = self.controls.siteCharSelect
	local charListData = charSelect.list[charSelect.selIndex].char
	launch:DownloadPage(
	realm.hostName ..
	"character-window/get-passive-skills?accountName=" ..
	accountName:gsub("#", "%%23") .. "&character=" .. urlEncode(charListData.name) .. "&realm=" .. realm.realmCode,
		function(response, errMsg)
			self.charImportMode = "SELECTCHAR"
			if errMsg then
				self.charImportStatus = colorCodes.NEGATIVE ..
				"Error importing character data, try again (" .. errMsg:gsub("\n", " ") .. ")"
				return
			elseif response.body == "false" then
				self.charImportStatus = colorCodes.NEGATIVE .. "Failed to retrieve character data, try again."
				return
			end
			self.lastCharacterHash = common.sha1(charListData.name)
			if not self.lastLeague then
				self.lastLeague = self.controls.siteCharSelectLeague:GetSelValueByKey("league")
			end
			local responseLua = dkjson.decode(response.body)
			-- Account-name imports omit quest choices, so keep the build's current values.
			responseLua.bandit_choice = responseLua.bandit_choice or self.build.configTab.input.bandit
			responseLua.pantheon_major = responseLua.pantheon_major or self.build.configTab.input.pantheonMajorGod
			responseLua.pantheon_minor = responseLua.pantheon_minor or self.build.configTab.input.pantheonMinorGod
			-- modify response to be like the oauth API response
			local charData = copyTable(charListData)
			charData.passives = responseLua
			charData.jewels = responseLua.items
			local deleteJewels = self.controls.siteCharImportTreeClearJewels.state
			self:ImportPassiveTreeAndJewels(charData, deleteJewels)
		end)
end

function ImportTabClass:DownloadItems(realm)
	self.charImportMode = "IMPORTING"
	self.charImportStatus = "Retrieving character items..."
	local accountName = self.controls.siteAccountName.buf
	local charSelect = self.controls.siteCharSelect
	local charListData = charSelect.list[charSelect.selIndex].char
	launch:DownloadPage(
	realm.hostName ..
	"character-window/get-items?accountName=" ..
		accountName:gsub("#", "%%23") .. "&character=" .. urlEncode(charListData.name) .. "&realm=" .. realm.realmCode,
		function(response, errMsg)
			self.charImportMode = "SELECTCHAR"
			if errMsg then
				self.charImportStatus = colorCodes.NEGATIVE ..
				"Error importing character data, try again (" .. errMsg:gsub("\n", " ") .. ")"
				return
			elseif response.body == "false" then
				self.charImportStatus = colorCodes.NEGATIVE .. "Failed to retrieve character data, try again."
				return
			end
			self.lastCharacterHash = common.sha1(charListData.name)
			if not self.lastLeague then
				self.lastLeague = self.controls.siteCharSelectLeague:GetSelValueByKey("league")
			end
			local responseLua = dkjson.decode(response.body)
			-- modify response to be like the oauth API response
			local charData = copyTable(charListData)
			charData.equipment = responseLua.items
			charData.guardian = responseLua.guardian
			local clearItems = self.controls.siteCharImportItemsClearItems.state
			local clearSkills = self.controls.siteCharImportItemsClearSkills.state
			local ignoreWeaponSwap = self.controls.siteCharImportItemsIgnoreWeaponSwap.state
			self:ImportItemsAndSkills(charData, clearItems, clearSkills, ignoreWeaponSwap)
		end)
end
function ImportTabClass:DownloadSiteCharacterList(realm)
	function FindMatchingStandardLeague(league)
		-- Find a Standard league name for a given league name
		-- Reference https://api.pathofexile.com/league?realm=pc
		if string.find(league, "Hardcore") then
			return "Hardcore"
		elseif string.find(league, "HC SSF") then
			-- includes Ruthless "HC SSF R "
			return "SSF Hardcore"
		elseif string.find(league, "SSF") then
			-- Any non HardCore SSF's - includes Ruthless "SSF R "
			return "SSF Standard"
		else
			-- normal league and ruthless league (Sanctum, Ruthless Sanctum)
			return "Standard"
		end
	end

	self.charImportMode = "DOWNLOADCHARLIST"
	self.charImportStatus = "Retrieving character list..."
	local accountName
	-- Handle spaces in the account name
	if realm.realmCode == "pc" then
		accountName = self.controls.siteAccountName.buf:gsub("%s+", "")
	else
		accountName = self.controls.siteAccountName.buf:gsub("^[%s?]+", ""):gsub("[%s?]+$", ""):gsub("%s", "+")
	end
	accountName = accountName:gsub("(.*)[#%-]", "%1#")
	launch:DownloadPage(
	realm.hostName ..
	"character-window/get-characters?accountName=" .. accountName:gsub("#", "%%23") .. "&realm=" .. realm.realmCode,
		function(response, errMsg)
			if errMsg == "Response code: 401" or errMsg == "Response code: 403" then
				self.charImportStatus = colorCodes.NEGATIVE .. "Account profile is private or does not exist."
				self.charImportMode = "GETACCOUNTNAME"
				return
			elseif errMsg == "Response code: 404" then
				self.charImportStatus = colorCodes.NEGATIVE .. "Account name is incorrect."
				self.charImportMode = "GETACCOUNTNAME"
				return
			elseif errMsg then
				self.charImportStatus = colorCodes.NEGATIVE ..
				"Error retrieving character list, try again (" .. errMsg:gsub("\n", " ") .. ")"
				self.charImportMode = "GETACCOUNTNAME"
				return
			end
			local charList, errMsg = self:ProcessSiteJSON(response.body)
			if errMsg then
				self.charImportStatus = colorCodes.NEGATIVE .. "Error processing character list, try again later"
				self.charImportMode = "GETACCOUNTNAME"
				return
			end
			--ConPrintTable(charList)
			if #charList == 0 then
				self.charImportStatus = colorCodes.NEGATIVE .. "The account has no characters to import."
				self.charImportMode = "GETACCOUNTNAME"
				return
			end
			-- GGG's character API has an issue where for /get-characters the account name is not case-sensitive, but for /get-passive-skills and /get-items it is.
			-- This workaround grabs the profile page and extracts the correct account name from one of the URLs.
			launch:DownloadPage(realm.hostName .. realm.profileURL .. accountName:gsub("#", "%%23"),
				function(response, errMsg)
					if errMsg then
						self.charImportStatus = colorCodes.NEGATIVE ..
						"Error retrieving character list, try again (" .. errMsg:gsub("\n", " ") .. ")"
						self.charImportMode = "GETACCOUNTNAME"
						return
					end
					local realAccountName = response and response.body and
					response.body:match("/view%-profile/([^/]+)/characters"):gsub(".",
						function(c) if c:byte(1) > 127 then return string.format("%%%2X", c:byte(1)) else return c end end)
					if not realAccountName then
						self.charImportStatus = colorCodes.NEGATIVE .. "Failed to retrieve character list."
						self.charImportMode = "GETACCOUNTNAME"
						return
					end
					realAccountName = realAccountName:gsub("(.*)[#%-]", "%1#")
					accountName = realAccountName
					self.controls.siteAccountName.buf = realAccountName
					self.charImportStatus = "Character list successfully retrieved."
					self.charImportMode = "SELECTCHAR"
					self.lastRealm = realm.id
					main.lastRealm = realm.id
					self.lastAccountHash = common.sha1(accountName)
					main.lastAccountName = accountName
					main.gameAccounts[accountName] = main.gameAccounts[accountName] or {}
					main.gameAccounts[accountName].sessionID = sessionID
					local leagueList = {}
					for i, char in ipairs(charList) do
						if not isValueInArray(leagueList, char.league) then
							t_insert(leagueList, char.league)
						end
					end
					table.sort(leagueList)
					local charSelectLeague = self.controls.siteCharSelectLeague
					wipeTable(self.controls.siteCharSelectLeague.list)
					for _, league in ipairs(leagueList) do
						t_insert(self.controls.siteCharSelectLeague.list, {
							label = league,
							league = league,
						})
					end
					t_insert(self.controls.siteCharSelectLeague.list, {
						label = "All",
					})
					-- set the league combo to the last used if possible, used for previously imported characters
					if self.lastLeague then
						charSelectLeague:SelByValue(self.lastLeague, "league")
						-- check that it worked
						if charSelectLeague:GetSelValueByKey("league") ~= self.lastLeague then
							-- League maybe over, Character will be in standard
							local standardLeagueName = FindMatchingStandardLeague(self.lastLeague)
							self.controls.siteCharSelectLeague:SelByValue(standardLeagueName, "league")
							if charSelectLeague:GetSelValueByKey("league") ~= standardLeagueName then
								-- give up and select the first entry. Ruthless mode may not have Standard equivalents
								charSelectLeague.selIndex = 1
							else
								self.lastLeague = standardLeagueName
							end
						end
					else
						if self.controls.siteCharSelectLeague.selIndex > #self.controls.siteCharSelectLeague.list then
							self.controls.siteCharSelectLeague.selIndex = 1
						end
					end
					self.lastCharList = charList
					self:BuildCharacterList(realm.realmCode,
						self.controls.siteCharSelectLeague:GetSelValueByKey("league"),
						self.lastCharList, self.controls.siteCharSelect)

					-- We only get here if the accountname was correct, found, and not private, so add it to the account history.
					self:SaveAccountHistory()
				end)
		end)
end

--- @param realm string
--- @param league string
--- @param characters table?
--- @param control table
function ImportTabClass:BuildCharacterList(realm, league, characters, control)
	wipeTable(control.list)
	if not characters then
		return
	end
	for i, char in ipairs(characters) do
		if realm == char.realm and ((not league) or char.league == league) then
			local charLvl = char.level or 0
			local charLeague = char.league or "?"
			local charName = char.name or "?"
			local charClass = char.class or "?"

			local classColor = colorCodes.DEFAULT
			if charClass ~= "?" then
				classColor = colorCodes[charClass:upper()]

				if classColor == nil then
					if (charClass == "Elementalist" or charClass == "Necromancer" or charClass == "Occultist" or
							charClass == "Harbinger" or charClass == "Herald" or charClass == "Bog Shaman") then
						classColor = colorCodes["WITCH"]
					elseif (charClass == "Guardian" or charClass == "Inquisitor" or charClass == "Hierophant" or
							charClass == "Architect of Chaos" or charClass == "Polytheist" or charClass == "Puppeteer") then
						classColor = colorCodes["TEMPLAR"]
					elseif (charClass == "Assassin" or charClass == "Trickster" or charClass == "Saboteur" or
							charClass == "Surfcaster" or charClass == "Servant of Arakaali" or charClass == "Blind Prophet") then
						classColor = colorCodes["SHADOW"]
					elseif (charClass == "Gladiator" or charClass == "Slayer" or charClass == "Champion" or
							charClass == "Gambler" or charClass == "Paladin" or charClass == "Aristocrat") then
						classColor = colorCodes["DUELIST"]
					elseif (charClass == "Raider" or charClass == "Pathfinder" or charClass == "Deadeye" or charClass == "Warden" or
							charClass == "Daughter of Oshabi" or charClass == "Whisperer" or charClass == "Wildspeaker") then
						classColor = colorCodes["RANGER"]
					elseif (charClass == "Juggernaut" or charClass == "Berserker" or charClass == "Chieftain" or
							charClass == "Antiquarian" or charClass == "Behemoth" or charClass == "Ancestral Commander") then
						classColor = colorCodes["MARAUDER"]
					elseif (charClass == "Ascendant" or charClass == "Reliquarian" or charClass == "Luminary" or
							charClass == "Scavenger") then
						classColor = colorCodes["SCION"]
					end
				end
			end

			local detail
			if league == nil then
				detail = string.format("%s%s ^x808080lvl %d in %s", classColor, charClass, charLvl, charLeague)
			else
				detail = string.format("%s%s ^x808080lvl %d", classColor, charClass, charLvl)
			end
			t_insert(control.list, {
				label = charName,
				char = char,
				searchFilter = charName.." "..charClass,
				detail = detail
			})
		end
	end
	table.sort(control.list, function(a,b)
		return a.char.name:lower() < b.char.name:lower()
	end)
	control.selIndex = 1
	if self.lastCharacterHash then
		for i, char in ipairs(control.list) do
			if common.sha1(char.char.name) == self.lastCharacterHash then
				control.selIndex = i
				break
			end
		end
	end
end
-- https://www.pathofexile.com/developer/docs/reference#type-Character
--- @class CharacterBasicData
--- @field id string? not present on website
--- @field name string
--- @field realm "pc" | "xbox" | "sony"
--- @field class string
--- @field league string
--- @field level integer
--- @field experience integer
---
--- @alias Bandits "Kraityn" | "Alira" | "Oak" | "Eramir"
--- @alias MajorPantheon "TheBrineKing" | "Arakaali" | "Solaris" | "Lunaris"
--- @alias MinorPantheon "Abberath" | "Gruthkul" | "Yugul" | "Shakari" | "Tukohama" | "Ralakesh" | "Garukhan" | "Ryslatha"


--- @class CharacterPassives
--- @field mastery_effects table<string, int>
--- @field skill_overrides table<string, table>
--- @field jewel_data table<string, table>
--- @field hashes_ex integer[]
--- @field hashes integer[]
--- @field bandit_choice Bandits?
--- @field pantheon_major MajorPantheon?
--- @field pantheon_minor MinorPantheon?
--- @field alternate_ascendancy string | integer integer on website, string on oauth

-- https://www.pathofexile.com/developer/docs/reference#type-Item
--- @alias Item any

--- @class CharacterPassivesData : CharacterBasicData
--- @field jewels Item[]
--- @field passives CharacterPassives
--- @param charData CharacterPassivesData
--- @param deleteJewels boolean
--- @return string
function ImportTabClass:ImportPassiveTreeAndJewels(charData, deleteJewels)
	local charPassives = copyTable(charData.passives)

	-- fix table keys being strings
	local masteries = {}
	for key, value in pairs(charPassives.mastery_effects or {}) do
		masteries[tonumber(key)] = value
	end
	self.build.spec.jewel_data = {}
	for key, value in pairs(charPassives.jewel_data or {}) do
		self.build.spec.jewel_data[tonumber(key)] = value
	end
	local skillOverrides = {}
	for nodeId, override in pairs(charPassives.skill_overrides or {}) do
		-- json keys are strings, not numbers
		local nodeIdNum = tonumber(nodeId)
		self.build.spec:ReplaceNode(override, self.build.spec.tree.tattoo.nodes[override.name])
		override.id = nodeIdNum
		skillOverrides[nodeIdNum] = override
	end

	self.build.spec.extended_hashes = copyTable(charPassives.hashes_ex)
	if deleteJewels then
		for _, slot in pairs(self.build.itemsTab.slots) do
			if slot.selItemId ~= 0 and slot.nodeId then
				self.build.itemsTab.build.spec.ignoreAllocatingSubgraph = true -- ignore allocated cluster nodes on Import when Delete Jewel is true, clean slate
				self.build.itemsTab:DeleteItem(self.build.itemsTab.items[slot.selItemId])
			end
		end
	end
	for _, itemData in ipairs(charData.jewels or {}) do
		self:ImportItem(itemData)
	end
	self.build.itemsTab:PopulateSlots()
	self.build.itemsTab:AddUndoState()

	-- Alternate trees don't have an identifier, so we're forced to look up something that is unique to that tree
	-- Hopefully this changes, because it's totally unmaintainable
	local function isAscendancyInTree(className, treeVersion)
		local classes = main.tree[treeVersion].classes
		for _, class in pairs(classes) do
			if class.name == className then
				return true
			end
			for i = 0, #class.classes do
				local ascendClass = class.classes[i]
				if ascendClass.name == className then
					return true
				end
			end
		end
	end

	local alternateAscendancyId
	if charPassives.alternate_ascendancy then
		-- oauth responses have bloodline names
		if type(charPassives.alternate_ascendancy) == "string" then
			local bloodline = self.build.latestTree.secondaryAscendNameMap[charPassives.alternate_ascendancy]
			alternateAscendancyId = bloodline and bloodline.ascendClassId
		-- site responses have integer ids
		else
			alternateAscendancyId = charPassives.alternate_ascendancy
		end
	else
		alternateAscendancyId = 0
	end
	-- Character import uses current GGG cluster hashes.
	self.build.spec.clusterHashFormatVersion = 2
	local ruthlessSuffix = charData.league:match("Ruthless") and "_ruthless" or ""
	local phreciaSuffix = isAscendancyInTree(charData.class, latestTreeVersion) and "" or "_alternate"
	self.build.spec:ImportFromNodeList(charData.class,
		nil,
		nil,
		alternateAscendancyId,
		charPassives.hashes,
		skillOverrides,
		masteries,
		latestTreeVersion .. ruthlessSuffix .. phreciaSuffix
		)
	self.build.treeTab:SetActiveSpec(self.build.treeTab.activeSpec)
	self.build.spec:BuildClusterJewelGraphs()
	self.build.spec:AddUndoState()
	self.build.characterLevel = charData.level or 100
	self.build.characterLevelAutoMode = false
	self.build.configTab:UpdateLevel()
	self.build:EstimatePlayerProgress()
	local resistancePenaltyIndex = 3
	if self.build.Act then -- Estimate resistance penalty setting based on act progression estimate
		if type(self.build.Act) == "string" and self.build.Act == "Endgame" then resistancePenaltyIndex = 3
		elseif type(self.build.Act) == "number" then
			if self.build.Act < 5 then resistancePenaltyIndex = 1
			elseif self.build.Act > 5 and self.build.Act < 11 then resistancePenaltyIndex = 2
			elseif self.build.Act > 10 then resistancePenaltyIndex = 3 end
		end
	end
	self.build.configTab:SetOptionByIndex("resistancePenalty", resistancePenaltyIndex)

	local bandit = (charPassives.bandit_choice == "Eramir" or not charPassives.bandit_choice) and "None" or
		charPassives.bandit_choice
	self.build.configTab:SetOption("bandit", bandit)

	local majorGod = charPassives.pantheon_major or "None"
	self.build.configTab:SetOption("pantheonMajorGod", majorGod)

	local minorGod = charPassives.pantheon_minor or "None"
	self.build.configTab:SetOption("pantheonMinorGod", minorGod)

	return colorCodes.POSITIVE.."Passive tree and jewels successfully imported."
end

local SOCKET_GROUP_REIMPORT_KEY_SEPARATOR = "\31"

local function getSocketGroupReimportKey(socketGroup)
	-- Use a rarely-used separator to avoid accidental collisions when concatenating fields.
	local gemNameParts = { }
	for _, gem in ipairs(socketGroup.gemList) do
		t_insert(gemNameParts, (gem.nameSpec or ""):lower())
	end
	return table.concat({
		socketGroup.slot or "",
		socketGroup.source or "",
		tostring(#socketGroup.gemList),
		table.concat(gemNameParts, SOCKET_GROUP_REIMPORT_KEY_SEPARATOR),
	}, SOCKET_GROUP_REIMPORT_KEY_SEPARATOR)
end

local function snapshotSocketGroupReimportState(socketGroup, isMainGroup)
	local gemStates = { }
	for gemIndex, gem in ipairs(socketGroup.gemList) do
		gemStates[gemIndex] = {
			enabled = gem.enabled,
			count = gem.count,
			skillPart = gem.skillPart,
			skillPartCalcs = gem.skillPartCalcs,
			skillStageCount = gem.skillStageCount,
			skillStageCountCalcs = gem.skillStageCountCalcs,
			skillMineCount = gem.skillMineCount,
			skillMineCountCalcs = gem.skillMineCountCalcs,
			skillMinion = gem.skillMinion,
			skillMinionCalcs = gem.skillMinionCalcs,
			skillMinionItemSet = gem.skillMinionItemSet,
			skillMinionItemSetCalcs = gem.skillMinionItemSetCalcs,
			skillMinionSkill = gem.skillMinionSkill,
			skillMinionSkillCalcs = gem.skillMinionSkillCalcs,
			enableGlobal1 = gem.enableGlobal1,
			enableGlobal2 = gem.enableGlobal2,
		}
	end
	return {
		enabled = socketGroup.enabled,
		includeInFullDPS = socketGroup.includeInFullDPS,
		groupCount = socketGroup.groupCount,
		label = socketGroup.label,
		mainActiveSkill = socketGroup.mainActiveSkill,
		mainActiveSkillCalcs = socketGroup.mainActiveSkillCalcs,
		gemStates = gemStates,
		isMainGroup = isMainGroup,
	}
end

local function applyGemReimportState(gem, state)
	gem.enabled = state.enabled
	gem.count = state.count
	gem.skillPart = state.skillPart
	gem.skillPartCalcs = state.skillPartCalcs
	gem.skillStageCount = state.skillStageCount
	gem.skillStageCountCalcs = state.skillStageCountCalcs
	gem.skillMineCount = state.skillMineCount
	gem.skillMineCountCalcs = state.skillMineCountCalcs
	gem.skillMinion = state.skillMinion
	gem.skillMinionCalcs = state.skillMinionCalcs
	gem.skillMinionItemSet = state.skillMinionItemSet
	gem.skillMinionItemSetCalcs = state.skillMinionItemSetCalcs
	gem.skillMinionSkill = state.skillMinionSkill
	gem.skillMinionSkillCalcs = state.skillMinionSkillCalcs
	gem.enableGlobal1 = state.enableGlobal1
	gem.enableGlobal2 = state.enableGlobal2
end

local function applySocketGroupReimportState(socketGroup, state)
	socketGroup.enabled = state.enabled
	socketGroup.includeInFullDPS = state.includeInFullDPS
	socketGroup.groupCount = state.groupCount
	socketGroup.label = state.label
	socketGroup.mainActiveSkill = state.mainActiveSkill
	socketGroup.mainActiveSkillCalcs = state.mainActiveSkillCalcs
	if state.gemStates then
		for gemIndex, gemState in ipairs(state.gemStates) do
			if socketGroup.gemList[gemIndex] then
				applyGemReimportState(socketGroup.gemList[gemIndex], gemState)
			end
		end
	end
end

local GUARD_ITEM_SET = "Animate Guardian"
-- Locates AG's item set from the import
function ImportTabClass:GetOrCreateGuardianItemSet()
	local itemsTab = self.build.itemsTab
	for _, itemSetId in ipairs(itemsTab.itemSetOrderList) do
		local itemSet = itemsTab.itemSets[itemSetId]
		if itemSet.title == GUARD_ITEM_SET then
			return itemSet
		end
	end
	local itemSet = itemsTab:NewItemSet()
	itemSet.title = GUARD_ITEM_SET
	t_insert(itemsTab.itemSetOrderList, itemSet.id)
	return itemSet
end

-- Allocates AG's item set for the AG skill gem.
function ImportTabClass:AssignGuardianItemSet(itemSetId)
	local itemsTab = self.build.itemsTab
	for _, socketGroup in ipairs(self.build.skillsTab.socketGroupList) do
		for _, gem in ipairs(socketGroup.gemList) do
			if gem.grantedEffect and gem.grantedEffect.name == "Animate Guardian" then
				for _, suffix in ipairs({ "", "Calcs" }) do
					local current = gem["skillMinionItemSet"..suffix]
					local currentSet = current and itemsTab.itemSets[current]
					if not current or (currentSet and currentSet.title == GUARD_ITEM_SET) then
						gem["skillMinionItemSet"..suffix] = itemSetId
					end
				end
			end
		end
	end
end

--- @class CharacterItemsData : CharacterBasicData
--- @field equipment Item[]
--- @param charData CharacterItemsData
--- @param clearItems boolean
--- @param clearSkills boolean
--- @param ignoreWeaponSwap boolean
--- @return CharacterItemsData, string
function ImportTabClass:ImportItemsAndSkills(charData, clearItems, clearSkills, ignoreWeaponSwap)
	charData = copyTable(charData)
	if clearItems then
		for _, slot in pairs(self.build.itemsTab.slots) do
			if slot.selItemId ~= 0 and not slot.nodeId then
				self.build.itemsTab:DeleteItem(self.build.itemsTab.items[slot.selItemId])
			end
		end
	end

	local mainSkillEmpty = #self.build.skillsTab.socketGroupList == 0
	local skillOrder
	local preservedSocketGroupStateByKey
	if clearSkills then
		skillOrder = { }
		preservedSocketGroupStateByKey = { }
		for _, socketGroup in ipairs(self.build.skillsTab.socketGroupList) do
			for _, gem in ipairs(socketGroup.gemList) do
				if gem.grantedEffect and not gem.grantedEffect.support then
					t_insert(skillOrder, gem.grantedEffect.name)
				end
			end
		end
		for index, socketGroup in ipairs(self.build.skillsTab.socketGroupList) do
			local key = getSocketGroupReimportKey(socketGroup)
			preservedSocketGroupStateByKey[key] = preservedSocketGroupStateByKey[key] or { }
			t_insert(preservedSocketGroupStateByKey[key], snapshotSocketGroupReimportState(socketGroup, index == self.build.mainSocketGroup))
		end
		wipeTable(self.build.skillsTab.socketGroupList)
		self.build.skillsTab:SetDisplayGroup()
		self.build.skillsTab:RebuildImbuedSupportBySlot()
	end
	for _, itemData in ipairs(charData.equipment) do
		self:ImportItem(itemData, nil, ignoreWeaponSwap)
	end
	if charData.guardian and charData.guardian[1] then
		local guardianSet = self:GetOrCreateGuardianItemSet()
		for _, itemData in ipairs(charData.guardian) do
			self:ImportItem(itemData, nil, ignoreWeaponSwap, guardianSet.id)
		end
		self:AssignGuardianItemSet(guardianSet.id)
	end
	if skillOrder then
		local groupOrder = { }
		for index, socketGroup in ipairs(self.build.skillsTab.socketGroupList) do
			groupOrder[socketGroup] = index
		end
		table.sort(self.build.skillsTab.socketGroupList, function(a, b)
			local orderA
			for _, gem in ipairs(a.gemList) do
				if gem.grantedEffect and not gem.grantedEffect.support then
					local i = isValueInArray(skillOrder, gem.grantedEffect.name)
					if i and (not orderA or i < orderA) then
						orderA = i
					end
				end
			end
			local orderB
			for _, gem in ipairs(b.gemList) do
				if gem.grantedEffect and not gem.grantedEffect.support then
					local i = isValueInArray(skillOrder, gem.grantedEffect.name)
					if i and (not orderB or i < orderB) then
						orderB = i
					end
				end
			end
			if orderA and orderB then
				if orderA ~= orderB then
					return orderA < orderB
				else
					return groupOrder[a] < groupOrder[b]
				end
			elseif not orderA and not orderB then
				return groupOrder[a] < groupOrder[b]
			else
				return orderA
			end
		end)
	end
	if preservedSocketGroupStateByKey then
		local restoredMainSocketGroup
		for index, socketGroup in ipairs(self.build.skillsTab.socketGroupList) do
			local stateList = preservedSocketGroupStateByKey[getSocketGroupReimportKey(socketGroup)]
			if stateList and stateList[1] then
				local state = t_remove(stateList, 1)
				applySocketGroupReimportState(socketGroup, state)
				if state.isMainGroup then
					restoredMainSocketGroup = index
				end
			end
		end
		if restoredMainSocketGroup then
			self.build.mainSocketGroup = restoredMainSocketGroup
		end
	end
	if mainSkillEmpty then
		self.build.mainSocketGroup = self:GuessMainSocketGroup()
	end
	self.build.itemsTab:PopulateSlots()
	self.build.itemsTab:AddUndoState()
	self.build.skillsTab:UpdateSocketGroups()
	self.build.skillsTab:AddUndoState()
	self.build.characterLevel = charData.level
	self.build.configTab:UpdateLevel()
	self.build.buildFlag = true
	-- charData for the wrapper
	return charData, colorCodes.POSITIVE .. "Items and skills successfully imported."
end

local rarityMap = { [0] = "NORMAL", "MAGIC", "RARE", "UNIQUE", [9] = "RELIC", [10] = "RELIC" }
local slotMap = { ["Weapon"] = "Weapon 1", ["Offhand"] = "Weapon 2", ["Weapon2"] = "Weapon 1 Swap", ["Offhand2"] = "Weapon 2 Swap", ["Helm"] = "Helmet", ["BodyArmour"] = "Body Armour", ["Gloves"] = "Gloves", ["Boots"] = "Boots",
				  ["Amulet"] = "Amulet", ["Ring"] = "Ring 1", ["Ring2"] = "Ring 2", ["Ring3"] = "Ring 3", ["Belt"] = "Belt",  ["BrequelGrafts"] = "Graft 1", ["BrequelGrafts2"] = "Graft 2", }

function ImportTabClass:ImportItem(itemData, slotName, ignoreWeaponSwap, itemSetId)
	if not slotName then
		if itemData.inventoryId == "PassiveJewels" then
			slotName = "Jewel "..self.build.latestTree.jewelSlots[itemData.x + 1]
		elseif itemData.inventoryId == "Flask" then
			slotName = "Flask "..(itemData.x + 1)
		elseif not (ignoreWeaponSwap and (itemData.inventoryId == "Weapon2" or itemData.inventoryId == "Offhand2")) then
			slotName = slotMap[itemData.inventoryId]
		end
	end
	if not slotName then
		-- Ignore any items that won't go into known slots
		return
	end

	local item = new("Item")

	-- Determine rarity, display name and base type of the item
	item.rarity = rarityMap[itemData.frameType]
	if #itemData.name > 0 then
		item.title = sanitiseText(itemData.name)
		item.baseName = sanitiseText(itemData.typeLine):gsub("Synthesised ", ""):gsub("^Vestigial ", "")
		item.name = item.title .. ", " .. item.baseName
		if item.baseName == "Two-Toned Boots" then
			-- Hack for Two-Toned Boots
			item.baseName = "Two-Toned Boots (Armour/Energy Shield)"
		end
		item.base = self.build.data.itemBases[item.baseName]
		if item.base then
			item.type = item.base.type
		else
			ConPrintf("Unrecognised base in imported item: %s", item.baseName)
		end
	else
		item.name = sanitiseText(itemData.typeLine)
		if item.name:match("Energy Blade") then
			local oneHanded = false
			for _, p in ipairs(itemData.properties) do
				if self.build.data.weaponTypeInfo[p.name] and self.build.data.weaponTypeInfo[p.name].oneHand then
					oneHanded = true
					break
				end
			end
			item.name = oneHanded and "Energy Blade One Handed" or "Energy Blade Two Handed"
			item.rarity = "NORMAL"
			itemData.implicitMods = { }
			itemData.explicitMods = { }
		end
		for baseName, baseData in pairs(self.build.data.itemBases) do
			local s, e = item.name:find(baseName, 1, true)
			if s then
				item.baseName = baseName
				item.namePrefix = item.name:sub(1, s - 1)
				item.nameSuffix = item.name:sub(e + 1)
				item.type = baseData.type
				break
			end
		end
		if not item.baseName then
			local s, e = item.name:find("Two-Toned Boots", 1, true)
			if s then
				-- Hack for Two-Toned Boots
				item.baseName = "Two-Toned Boots (Armour/Energy Shield)"
				item.namePrefix = item.name:sub(1, s - 1)
				item.nameSuffix = item.name:sub(e + 1)
				item.type = "Boots"
			end
		end
		item.base = self.build.data.itemBases[item.baseName]
	end
	if not item.base or not item.rarity then
		return
	end

	-- Import item data
	item.uniqueID = itemData.id
	if itemData.influences then
		for _, curInfluenceInfo in ipairs(influenceInfo) do
			item[curInfluenceInfo.key] = itemData.influences[curInfluenceInfo.display:lower()]
		end
	end
	if itemData.searing then
		item.cleansing = true
	end
	if itemData.tangled then
		item.tangle = true
	end
	if itemData.ilvl > 0 then
		item.itemLevel = itemData.ilvl
	end
	if item.base.weapon or item.base.armour or item.base.flask or item.base.tincture then
		item.quality = 0
	end
	if itemData.properties then
		for _, property in pairs(itemData.properties) do
			if property.name == "Quality" then
				item.quality = tonumber(property.values[1][1]:match("%d+"))
			elseif property.name:match("Quality %(") then
				local catalystMap = {
					["Attack"] = 1,
					["Speed"] = 2,
					["Suffix"] = 3,
					["Life and Mana"] = 4,
					["Caster"] = 5,
					["Attribute"] = 6,
					["Physical and Chaos Damage"] = 7,
					["Resistance"] = 8,
					["Prefix"] = 9,
					["Defense"] = 10,
					["Elemental Damage"] = 11,
					["Critical"] = 12,
				}
				item.catalyst = catalystMap[property.name:match("Quality %((.*) Modifiers%)")]
				item.catalystQuality = tonumber(property.values[1][1]:match("%d+"))
			elseif property.name == "Radius" then
				item.jewelRadiusLabel = property.values[1][1]
			elseif property.name == "Limited to" then
				item.limit = tonumber(property.values[1][1])
			elseif property.name == "Evasion Rating" then
				if item.baseName == "Two-Toned Boots (Armour/Energy Shield)" then
					-- Another hack for Two-Toned Boots
					item.baseName = "Two-Toned Boots (Armour/Evasion)"
					item.base = self.build.data.itemBases[item.baseName]
				end
			elseif property.name == "Energy Shield" then
				if item.baseName == "Two-Toned Boots (Armour/Evasion)" then
					-- Yet another hack for Two-Toned Boots
					item.baseName = "Two-Toned Boots (Evasion/Energy Shield)"
					item.base = self.build.data.itemBases[item.baseName]
				end
			elseif property.name:find("Intangibility") then
				item.intangibility = tonumber(property.values[1][1]:match("%d+"))
			elseif property.name == "Memory Strands" then
				item.memoryStrands = tonumber(property.values[1][1])
			end
			if property.name == "Energy Shield" or property.name == "Ward" or property.name == "Armour" or property.name == "Evasion Rating" then
				item.armourData = item.armourData or { }
				for _, value in ipairs(property.values) do
					item.armourData[property.name:gsub(" Rating", ""):gsub(" ", "")] = (item.armourData[property.name:gsub(" Rating", ""):gsub(" ", "")] or 0) + tonumber(value[1])
				end
			end
		end
	end
	item.split = itemData.split
	item.mirrored = itemData.mirrored
	item.corrupted = itemData.corrupted
	item.fractured = itemData.fractured
	item.synthesised = itemData.synthesised
	item.vestigial = itemData.vestigial
	if itemData.sockets and itemData.sockets[1] then
		item.sockets = { }
		for i, socket in pairs(itemData.sockets) do
			if socket.sColour == "A" then
				item.abyssalSocketCount = item.abyssalSocketCount or 0 + 1
			end
			item.sockets[i] = { group = socket.group, color = socket.sColour }
		end
		if item.abyssalSocketCount and item.abyssalSocketCount > 0 and item.name:match("Energy Blade") then
			t_insert(itemData.explicitMods, "Has " .. item.abyssalSocketCount .. " Abyssal Sockets")
		end
	end
	if itemData.socketedItems then
		self:ImportSocketedItems(item, itemData.socketedItems, slotName)
	end
	if itemData.requirements and (not itemData.socketedItems or not itemData.socketedItems[1]) then
		-- Requirements cannot be trusted if there are socketed gems, as they may override the item's natural requirements
		item.requirements = { }
		for _, req in ipairs(itemData.requirements) do
			if req.name == "Level" then
				item.requirements.level = req.values[1][1]
			elseif req.name == "Class:" then
				item.classRestriction = req.values[1][1]
			end
		end
	end
	item.enchantModLines = { }
	item.scourgeModLines = { }
	item.classRequirementModLines = { }
	item.implicitModLines = { }
	item.explicitModLines = { }
	item.crucibleModLines = { }
	if itemData.enchantMods then
		for _, line in ipairs(itemData.enchantMods) do
			for line in line:gmatch("[^\n]+") do
				local modList, extra = modLib.parseMod(line)
				t_insert(item.enchantModLines, { line = line, extra = extra, mods = modList or { }, crafted = true })
			end
		end
	end
	if itemData.scourgeMods then
		for _, line in ipairs(itemData.scourgeMods) do
			for line in line:gmatch("[^\n]+") do
				local modList, extra = modLib.parseMod(line)
				t_insert(item.scourgeModLines, { line = line, extra = extra, mods = modList or { }, scourge = true })
			end
		end
	end
	if itemData.implicitMods then
		for _, itemMod in ipairs(itemData.implicitMods) do
			local modLine = itemMod.description or itemMod
			local flags = itemMod.flags or itemMod
			for line in modLine:gmatch("[^\n]+") do
				local modList, extra = modLib.parseMod(line)
				t_insert(item.implicitModLines, { line = line, extra = extra, mods = modList or { }, vestigial = flags.vestigial })
			end
		end
	end
	-- TODO: Remove once 3.29 releases https://www.pathofexile.com/developer/docs/changelog#3-29-0
	if itemData.fracturedMods then
		for _, line in ipairs(itemData.fracturedMods) do
			for line in line:gmatch("[^\n]+") do
				local modList, extra = modLib.parseMod(line)
				t_insert(item.explicitModLines, { line = line, extra = extra, mods = modList or { }, fractured = true })
			end
		end
	end
	if itemData.explicitMods then
		for _, itemMod in ipairs(itemData.explicitMods) do
			local modLine = itemMod.description or itemMod
			local flags = itemMod.flags or itemMod
			for line in modLine:gmatch("[^\n]+") do
				local modList, extra = modLib.parseMod(line)
				t_insert(item.explicitModLines, { line = line, extra = extra, mods = modList or { },
					fractured = flags.fractured,
					crafted = flags.crafted,
					mutated = flags.mutated,
					vestigial = flags.vestigial })
			end
		end
	end
	if itemData.crucibleMods then
		for _, line in ipairs(itemData.crucibleMods) do
			for line in line:gmatch("[^\n]+") do
				local modList, extra = modLib.parseMod(line)
				t_insert(item.crucibleModLines, { line = line, extra = extra, mods = modList or { }, crucible = true })
			end
		end
	end
	-- TODO: Remove once 3.29 releases https://www.pathofexile.com/developer/docs/changelog#3-29-0
	if itemData.craftedMods then
		for _, line in ipairs(itemData.craftedMods) do
			for line in line:gmatch("[^\n]+") do
				local modList, extra = modLib.parseMod(line)
				t_insert(item.explicitModLines, { line = line, extra = extra, mods = modList or { }, crafted = true })
			end
		end
	end
	-- TODO: Remove once 3.29 releases https://www.pathofexile.com/developer/docs/changelog#3-29-0
	if itemData.mutatedMods then
		for _, line in ipairs(itemData.mutatedMods) do
			for line in line:gmatch("[^\n]+") do
				local modList, extra = modLib.parseMod(line)
				t_insert(item.explicitModLines, { line = line, extra = extra, mods = modList or { }, mutated = true })
			end
		end
	end
	-- Sometimes flavour text has actual mods that PoB cares about
	-- Right now, the only known one is "This item can be anointed by Cassia"
	if itemData.flavourText then
		for _, line in ipairs(itemData.flavourText) do
			for line in line:gmatch("[^\n]+") do
				-- Remove any text outside of curly braces, if they exist.
				-- This fixes lines such as:
				--   "<default>{This item can be anointed by Cassia}"
				-- To now be:
				--   "This item can be anointed by Cassia"
				local startBracket = line:find("{")
				local endBracket = line:find("}")
				if startBracket and endBracket and endBracket > startBracket then
					line = line:sub(startBracket + 1, endBracket - 1)
				end

				-- If the line parses, then it should be included as an explicit mod
				local modList, extra = modLib.parseMod(line)
				if modList then
					t_insert(item.explicitModLines, { line = line, extra = extra, mods = modList or { } })
				end
			end
		end
	end
	if itemData.foilVariation or itemData.isRelic then
		local foilVariants = {
			"Amethyst",
			"Verdant",
			"Ruby",
			"Cobalt",
			"Sunset",
			"Aureate",
			"Celestial Quartz",
			"Celestial Ruby",
			"Celestial Emerald",
			"Celestial Aureate",
			"Celestial Pearl",
			"Celestial Amethyst",
		}
		item.foilType = foilVariants[itemData.foilVariation] or "Rainbow"
	end

	-- Add and equip the new item
	item:BuildAndParseRaw()
	--ConPrintf("%s", item.raw)
	if item.base then
		local repIndex, repItem
		for index, item in pairs(self.build.itemsTab.items) do
			if item.uniqueID == itemData.id then
				repIndex = index
				repItem = item
				break
			end
		end
		if repIndex then
			-- Item already exists in the build, overwrite it
			item.id = repItem.id
			self.build.itemsTab.items[item.id] = item
			item:BuildModList()
		else
			self.build.itemsTab:AddItem(item, true)
		end
		if itemSetId and itemSetId ~= self.build.itemsTab.activeItemSetId then
			self.build.itemsTab.itemSets[itemSetId][slotName].selItemId = item.id
		else
			self.build.itemsTab.slots[slotName]:SetSelItemId(item.id)
		end
	end
end

function ImportTabClass:ImportSocketedItems(item, socketedItems, slotName)
	-- Build socket group list
	local itemSocketGroupList = { }
	local abyssalSocketId = 1
	for _, socketedItem in ipairs(socketedItems) do
		if socketedItem.abyssJewel then
			self:ImportItem(socketedItem, slotName .. " Abyssal Socket "..abyssalSocketId)
			abyssalSocketId = abyssalSocketId + 1
		else
			local normalizedBasename = sanitiseText(socketedItem.typeLine)
			local gemId = self.build.data.gemForBaseName[normalizedBasename:lower()]
			if socketedItem.hybrid then
				-- Used by transfigured gems and dual-skill gems (currently just Stormbind)
				normalizedBasename = sanitiseText(socketedItem.hybrid.baseTypeName)
				gemId = self.build.data.gemForBaseName[normalizedBasename:lower()]
				if gemId and socketedItem.hybrid.isVaalGem then
					gemId = self.build.data.gemGrantedEffectIdForVaalGemId[self.build.data.gems[gemId].grantedEffectId]
				end
			end
			if gemId then
				local gemInstance = { level = 20, quality = 0, enabled = true, enableGlobal1 = true, gemId = gemId }
				gemInstance.nameSpec = self.build.data.gems[gemId].name
				gemInstance.support = socketedItem.support
				for _, property in pairs(socketedItem.properties) do
					if property.name == "Level" then
						gemInstance.level = tonumber(property.values[1][1]:match("%d+"))
					elseif property.name == "Quality" then
						gemInstance.quality = tonumber(property.values[1][1]:match("%d+"))
					end
				end
				local groupID = item.sockets[socketedItem.socket + 1].group
				if not itemSocketGroupList[groupID] then
					itemSocketGroupList[groupID] = { label = "", enabled = true, gemList = { }, slot = slotName }
				end
				local socketGroup = itemSocketGroupList[groupID]
				t_insert(socketGroup.gemList, gemInstance)
				if socketedItem.builtInSupport then
					socketGroup.imbuedSupport = socketedItem.builtInSupport:gsub("Supported by Level 1 ", "")
					local imbuedGem = data.gems[data.gemForBaseName[socketGroup.imbuedSupport:lower().." support"]]
					if imbuedGem and imbuedGem.grantedEffect then
						self.build.skillsTab.imbuedSupportBySlot[slotName] = imbuedGem.grantedEffect
						self.build.buildFlag = true
					end
				end
			end
		end
	end

	-- Import the socket groups
	for _, itemSocketGroup in pairs(itemSocketGroupList) do
		-- Check if this socket group matches an existing one
		local repGroup
		for index, socketGroup in pairs(self.build.skillsTab.socketGroupList) do
			if #socketGroup.gemList == #itemSocketGroup.gemList and (not socketGroup.slot or socketGroup.slot == slotName) then
				local match = true
				for gemIndex, gem in pairs(socketGroup.gemList) do
					if gem.nameSpec:lower() ~= itemSocketGroup.gemList[gemIndex].nameSpec:lower() then
						match = false
						break
					end
				end
				if match then
					repGroup = socketGroup
					break
				end
			end
		end
		if repGroup then
			-- Update the existing one
			for gemIndex, gem in pairs(repGroup.gemList) do
				local itemGem = itemSocketGroup.gemList[gemIndex]
				gem.level = itemGem.level
				gem.quality = itemGem.quality
			end
		else
			t_insert(self.build.skillsTab.socketGroupList, itemSocketGroup)
		end
		self.build.skillsTab:ProcessSocketGroup(itemSocketGroup)
	end
end

-- Return the index of the group with the most gems
function ImportTabClass:GuessMainSocketGroup()
	local largestGroupSize = 0
	local largestGroupIndex = 1
	for i, socketGroup in ipairs(self.build.skillsTab.socketGroupList) do
		if #socketGroup.gemList > largestGroupSize then
			largestGroupSize = #socketGroup.gemList
			largestGroupIndex = i
		end
	end
	return largestGroupIndex
end

function HexToChar(x)
	return string.char(tonumber(x, 16))
end

function UrlDecode(url)
	if url == nil then
		return
	end
	url = url:gsub("+", " ")
	url = url:gsub("%%(%x%x)", HexToChar)
	return url
end

function ImportTabClass:SetPredefinedBuildName()
	local accountName = self.controls.siteAccountName.buf:gsub('%s+', ''):gsub("#%d+", "")
	local charSelect = self.controls.siteCharSelect
	local charData = charSelect.list[charSelect.selIndex].char
	local charName = charData.name
	main.predefinedBuildName = accountName .. " - " .. charName
end
