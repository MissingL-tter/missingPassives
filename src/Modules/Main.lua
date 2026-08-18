-- Path of Building
--
-- Module: Main
-- Main module of program.
--
local ipairs = ipairs
local t_insert = table.insert
local t_remove = table.remove
local m_max = math.max
local m_min = math.min

LoadModule("GameVersions")
LoadModule("Modules/Common")
LoadModule("Modules/CalcFormat")
LoadModule("Modules/Data")
LoadModule("Modules/ModTools")
LoadModule("Modules/ItemTools")
LoadModule("Modules/CalcTools")
LoadModule("Modules/PantheonTools")
LoadModule("Modules/BuildSiteTools")

if arg and isValueInTable(arg, "--no-jit") then
	require("jit").off()
	ConPrintf("JIT Disabled")
end

if arg and isValueInTable(arg, "--no-ssl") then
	launch.noSSL = true
	ConPrintf("SSL verification disabled")
end

main = { }

function main:Init()
	self:DetectUnicodeSupport()
	self.popups = { }
	self.modes = { }
	-- Placeholder mode for when no build is open; holds the build list location state
	self.modes["LIST"] = {
		subPath = "",
		Init = function(mode, selBuildName, subPath)
			mode.selBuildName = selBuildName
			mode.subPath = subPath or ""
		end,
		GetArgs = function(mode)
			return mode.selBuildName, mode.subPath
		end,
	}
	self.modes["BUILD"] = LoadModule("Modules/Build")

	self.sharedItemList = { }
	self.sharedItemSetList = { }
	self.gameAccounts = { }

	local ignoreBuild
	if arg[1] then
		local importLink = buildSites.ParseImportLinkFromURI(arg[1])
		buildSites.DownloadBuild(arg[1], nil, function(isSuccess, data, importLink)
			if not isSuccess then
				self:SetMode("BUILD", false, data)
			else
				local xmlText = Inflate(common.base64.decode(data:gsub("-","+"):gsub("_","/")))
				self:SetMode("BUILD", false, "Imported Build", xmlText, false, importLink)
				self.newModeChangeToTree = true
			end
		end)
		arg[1] = nil -- Protect against downloading again this session.
		ignoreBuild = true
	end

	if not ignoreBuild then
		self:SetMode("BUILD", false, "Unnamed build")
	end
	-- Always put user data in the script path
	self.userPath = GetScriptPath().."/"

	self.buildSortMode = "NAME"
	self.connectionProtocol = 0
	self.nodePowerTheme = "RED/BLUE"
	self.colorPositive = defaultColorCodes.POSITIVE
	self.colorNegative = defaultColorCodes.NEGATIVE
	self.colorHighlight = defaultColorCodes.HIGHLIGHT
	self.showThousandsSeparators = true
	self.useCompactValues = false
	self.edgeSearchHighlight = true
	self.thousandsSeparator = ","
	self.decimalSeparator = "."
	self.defaultItemAffixQuality = 0.5
	self.showTitlebarName = true
	self.showWarnings = true
	self.slotOnlyTooltips = true
	self.migrateEldritchImplicits = true
	self.notSupportedModTooltips = true
	self.notSupportedTooltipText = " ^8(Not supported in PoB yet)"
	self.showPublicBuilds = true
	self.showFlavourText = true
	self.showAnimations = true
	self.showAllItemAffixes = true
	self.disableScrollControlInteraction = false
	self.errorReadingSettings = false

	if launch.devMode and IsKeyDown("CTRL") or os.getenv("REGENERATE_MOD_CACHE") == "1" then
		-- If modLib.parseMod doesn't find a cache entry it generates it.
		-- Not loading pre-generated cache causes it to be rebuilt
		self.saveNewModCache = true
	else
		-- Load mod cache
		LoadModule("Data/ModCache", modLib.parseModCache)
	end

	self.tree = { }
	self:LoadTree(latestTreeVersion)

	if self.userPath then
		self:ChangeUserPath(self.userPath, ignoreBuild)
	end

	self.uniqueDB = { list = { }, loading = true }
	self.rareDB = { list = { }, loading = true }

	local function loadItemDBs()
		for type, typeList in pairsYield(data.uniques) do
			for _, raw in pairs(typeList) do
				local newItem = new("Item", raw, "UNIQUE", true)
				if newItem.base then
					self.uniqueDB.list[newItem.name] = newItem
				elseif launch.devMode then
					ConPrintf("Unique DB unrecognised item of type '%s':\n%s", type, raw)
				end
			end
		end

		self.uniqueDB.loading = nil
		ConPrintf("Uniques loaded")

		for _, raw in pairsYield(data.rares) do
			local newItem = new("Item", raw, "RARE", true)
			if newItem.base then
				if newItem.crafted then
					if newItem.base.implicit and #newItem.implicitModLines == 0 then
						-- Automatically add implicit
						local implicitIndex = 1
						for line in newItem.base.implicit:gmatch("[^\n]+") do
							t_insert(newItem.implicitModLines, { line = line, modTags = newItem.base.implicitModTypes and newItem.base.implicitModTypes[implicitIndex] or { } })
							implicitIndex = implicitIndex + 1
						end
					end
					newItem:Craft()
				end
				self.rareDB.list[newItem.name] = newItem
			elseif launch.devMode then
				ConPrintf("Rare DB unrecognised item:\n%s", raw)
			end
		end

		self.rareDB.loading = nil
		ConPrintf("Rares loaded")
	end

	if self.saveNewModCache then
		local saved = self.defaultItemAffixQuality
		self.defaultItemAffixQuality = 0.5
		loadItemDBs()
		self:SaveModCache()
		self.defaultItemAffixQuality = saved
	end

	self.onFrameFuncs = {
		["FirstFrame"] = function()
			self.onFrameFuncs["FirstFrame"] = nil
			if launch.devMode then
				data.printMissingMinionSkills()
			end
			ConPrintf("Startup time: %d ms", GetTime() - launch.startTime)
		end
	}

	if not self.saveNewModCache then
		local itemsCoroutine = coroutine.create(loadItemDBs)

		self.onFrameFuncs["LoadItems"] = function()
			local res, errMsg = coroutine.resume(itemsCoroutine)
			if coroutine.status(itemsCoroutine) == "dead" then
				self.onFrameFuncs["LoadItems"] = nil
			end
			if not res then
				error(errMsg)
			end
		end
	end
end

function main:DetectUnicodeSupport()
	-- PoeCharm has utf8 global that normal PoB doesn't have
	self.unicode = type(_G.utf8) == "table"
	if self.unicode then
		ConPrintf("Unicode support detected")
	end
end

function main:SaveModCache()
	-- Update mod cache
	local out = io.open("Data/ModCache.lua", "w")
	out:write('local c=...')
	for line, dat in pairsSortByKey(modLib.parseModCache) do
		if not dat[1] or not dat[1][1] or (dat[1][1].name ~= "JewelFunc" and dat[1][1].name ~= "ExtraJewelFunc") then
			out:write('c["', line:gsub("\n","\\n"), '"]={')
			if dat[1] then
				writeLuaTable(out, dat[1])
			else
				out:write('nil')
			end
			if dat[2] then
				out:write(',"', dat[2]:gsub("\n","\\n"), '"}\n')
			else
				out:write(',nil}\n')
			end
		end
	end
	out:close()
end

function main:LoadTree(treeVersion)
	if self.tree[treeVersion] then
		data.setJewelRadiiGlobally(treeVersion)
		return self.tree[treeVersion]
	elseif isValueInTable(treeVersionList, treeVersion) then
		data.setJewelRadiiGlobally(treeVersion)
		self.tree[treeVersion] = new("PassiveTree", treeVersion)
		return self.tree[treeVersion]
	end
	return nil
end

function main:CanExit()
	local ret = self:CallMode("CanExit", "EXIT")
	if ret ~= nil then
		return ret
	else
		return true
	end
end

function main:Shutdown()
	self:CallMode("Shutdown")
	self:SaveSettings()
end

function main:OnFrame()
	while self.newMode do
		if self.mode then
			self:CallMode("Shutdown")
		end
		self.mode = self.newMode
		self.newMode = nil
		self:CallMode("Init", unpack(self.newModeArgs))
		if self.newModeChangeToTree then
			self.modes[self.mode].viewMode = "TREE"
		end
		self.newModeChangeToTree = false
	end

	self:CallMode("OnFrame")

	if launch.updateErrMsg then
		ConPrintf("Update check failed!\n%s", launch.updateErrMsg)
		launch.updateErrMsg = nil
	end
	if launch.updateAvailable then
		if launch.updateAvailable == "none" then
			launch.updateAvailable = nil
		elseif not self.updateAvailableShown then
			ConPrintf("Update Available\nAn update has been downloaded and is ready to be applied.")
			self.updateAvailableShown = true
		end
	end

	-- TODO: this pattern may pose memory management issues for classes that don't exist for the lifetime of the program
	for _, onFrameFunc in pairs(self.onFrameFuncs) do
		onFrameFunc()
	end
end

function main:SetMode(newMode, ...)
	self.newMode = newMode
	self.newModeArgs = {...}
	self.predefinedBuildName = nil
end

function main:CallMode(func, ...)
	local modeTbl = self.modes[self.mode]
	if modeTbl and modeTbl[func] then
		return modeTbl[func](modeTbl, ...)
	end
end

function main:LoadSettings(ignoreBuild)
	if self.errorReadingSettings then
		return true
	end
	local setXML, errMsg = common.xml.LoadXMLFile(self.userPath.."Settings.xml")
	if errMsg and errMsg:match(".*file returns nil") then
		self.errorReadingSettings = true
		self:OpenCloudErrorPopup(self.userPath.."Settings.xml")
		return true
	elseif errMsg and not errMsg:match(".*No such file or directory") then
		self.errorReadingSettings = true
		launch:ShowErrMsg("^1"..errMsg)
		return true
	end
	if not setXML then
		return true
	elseif setXML[1].elem ~= "PathOfBuilding" then
		launch:ShowErrMsg("^1Error parsing 'Settings.xml': 'PathOfBuilding' root element missing")
		return true
	end
	for _, node in ipairs(setXML[1]) do
		if type(node) == "table" then
			if not ignoreBuild and node.elem == "Mode" then
				if not node.attrib.mode or not self.modes[node.attrib.mode] then
					launch:ShowErrMsg("^1Error parsing 'Settings.xml': Invalid mode attribute in 'Mode' element")
					return true
				end
				local args = { }
				for _, child in ipairs(node) do
					if type(child) == "table" then
						if child.elem == "Arg" then
							if child.attrib.number then
								t_insert(args, tonumber(child.attrib.number))
							elseif child.attrib.string then
								t_insert(args, child.attrib.string)
							elseif child.attrib.boolean then
								t_insert(args, child.attrib.boolean == "true")
							end
						end
					end
				end
				self:SetMode(node.attrib.mode, unpack(args))
			elseif node.elem == "Accounts" then
				self.lastAccountName = node.attrib.lastAccountName
				self.lastRealm = node.attrib.lastRealm
				self.lastLeague = node.attrib.lastLeague
				self.lastToken = node.attrib.lastToken
				self.lastRefreshToken = node.attrib.lastRefreshToken
				self.tokenExpiry = tonumber(node.attrib.tokenExpiry)
				for _, child in ipairs(node) do
					if child.elem == "Account" then
						self.gameAccounts[child.attrib.accountName] = {
							sessionID = child.attrib.sessionID,
						}
					end
				end
			elseif node.elem == "Misc" then
				if node.attrib.buildSortMode then
					self.buildSortMode = node.attrib.buildSortMode
				end
				launch.connectionProtocol = tonumber(node.attrib.connectionProtocol)
				launch.proxyURL = node.attrib.proxyURL
				if node.attrib.buildPath then
					self.buildPath = node.attrib.buildPath
				end
				if node.attrib.nodePowerTheme then
					self.nodePowerTheme = node.attrib.nodePowerTheme
				end
				if node.attrib.colorPositive then
					updateColorCode("POSITIVE", node.attrib.colorPositive)
					self.colorPositive = node.attrib.colorPositive
				end
				if node.attrib.colorNegative then
					updateColorCode("NEGATIVE", node.attrib.colorNegative)
					self.colorNegative = node.attrib.colorNegative
				end
				if node.attrib.colorHighlight then
					updateColorCode("HIGHLIGHT", node.attrib.colorHighlight)
					self.colorHighlight = node.attrib.colorHighlight
				end

				-- In order to preserve users' settings through renaming/merging this variable, we have this if statement to use the first found setting
				-- Once the user has closed PoB once, they will be using the new `showThousandsSeparator` variable name, so after some time, this statement may be removed
				if node.attrib.showThousandsCalcs then
					self.showThousandsSeparators = node.attrib.showThousandsCalcs == "true"
				elseif node.attrib.showThousandsSidebar then
					self.showThousandsSeparators = node.attrib.showThousandsSidebar == "true"
				end
				if node.attrib.showThousandsSeparators then
					self.showThousandsSeparators = node.attrib.showThousandsSeparators == "true"
				end
				if node.attrib.thousandsSeparator then
					self.thousandsSeparator = node.attrib.thousandsSeparator
				end
				if node.attrib.useCompactValues then
					self.useCompactValues = node.attrib.useCompactValues == "true"
				end
				if node.attrib.decimalSeparator then
					self.decimalSeparator = node.attrib.decimalSeparator
				end
				if node.attrib.showTitlebarName then
					self.showTitlebarName = node.attrib.showTitlebarName == "true"
				end
				if node.attrib.betaTest then
					self.betaTest = node.attrib.betaTest == "true"
				end
				if node.attrib.edgeSearchHighlight then
					self.edgeSearchHighlight = node.attrib.edgeSearchHighlight == "true"
				end
				if node.attrib.defaultGemQuality then
					self.defaultGemQuality = m_min(tonumber(node.attrib.defaultGemQuality) or 0, 23)
				end
				if node.attrib.defaultCharLevel then
					self.defaultCharLevel = m_min(m_max(tonumber(node.attrib.defaultCharLevel) or 1, 1), 100)
				end
				if node.attrib.defaultItemAffixQuality then
					self.defaultItemAffixQuality = m_min(tonumber(node.attrib.defaultItemAffixQuality) or 0.5, 1)
				end
				if node.attrib.lastExportedWebsite then
					self.lastExportedWebsite = node.attrib.lastExportedWebsite
				end
				if node.attrib.showWarnings then
					self.showWarnings = node.attrib.showWarnings == "true"
				end
				if node.attrib.slotOnlyTooltips then
					self.slotOnlyTooltips = node.attrib.slotOnlyTooltips == "true"
				end
				if node.attrib.migrateEldritchImplicits then
					self.migrateEldritchImplicits = node.attrib.migrateEldritchImplicits == "true"
				end
				if node.attrib.notSupportedModTooltips then
					self.notSupportedModTooltips = node.attrib.notSupportedModTooltips == "true"
				end
				if node.attrib.invertSliderScrollDirection then
					self.invertSliderScrollDirection = node.attrib.invertSliderScrollDirection == "true"
				end
				if node.attrib.disableDevAutoSave then
					self.disableDevAutoSave = node.attrib.disableDevAutoSave == "true"
				end
				if node.attrib.showPublicBuilds then
					self.showPublicBuilds = node.attrib.showPublicBuilds == "true"
				end
				if node.attrib.showFlavourText then
					self.showFlavourText = node.attrib.showFlavourText == "true"
				end
				if node.attrib.showAnimations then
					self.showAnimations = node.attrib.showAnimations == "true"
				end
				if node.attrib.showAllItemAffixes then
					self.showAllItemAffixes = node.attrib.showAllItemAffixes == "true"
				end
				if node.attrib.disableScrollControlInteraction then
					self.disableScrollControlInteraction = node.attrib.disableScrollControlInteraction == "true"
				end
			end
		end
	end
end

function main:LoadSharedItems()
	if self.errorReadingSettings then
		return true
	end
	local setXML, errMsg = common.xml.LoadXMLFile(self.userPath.."Settings.xml")
	if errMsg and errMsg:match(".*file returns nil") then
		self.errorReadingSettings = true
		self:OpenCloudErrorPopup(self.userPath.."Settings.xml")
		return true
	elseif errMsg and not errMsg:match(".*No such file or directory") then
		self.errorReadingSettings = true
		launch:ShowErrMsg("^1"..errMsg)
		return true
	end
	if not setXML then
		return true
	elseif setXML[1].elem ~= "PathOfBuilding" then
		launch:ShowErrMsg("^1Error parsing 'Settings.xml': 'PathOfBuilding' root element missing")
		return true
	end
	for _, node in ipairs(setXML[1]) do
		if type(node) == "table" then
			if node.elem == "SharedItems" then
				for _, child in ipairs(node) do
					if child.elem == "Item" then
						local rawItem = { raw = "" }
						for _, subChild in ipairs(child) do
							if type(subChild) == "string" then
								rawItem.raw = subChild
							end
						end
						local newItem = new("Item", rawItem.raw)
						t_insert(self.sharedItemList, newItem)
					elseif child.elem == "ItemSet" then
						local sharedItemSet = { title = child.attrib.title, slots = { } }
						for _, grandChild in ipairs(child) do
							if grandChild.elem == "Item" then
								local rawItem = { raw = "" }
								for _, subChild in ipairs(grandChild) do
									if type(subChild) == "string" then
										rawItem.raw = subChild
									end
								end
								local newItem = new("Item", rawItem.raw)
								sharedItemSet.slots[grandChild.attrib.slotName] = newItem
							end
						end
						t_insert(self.sharedItemSetList, sharedItemSet)
					end
				end
			end
		end
	end
end

function main:SaveSettings()
	if self.errorReadingSettings then
		return
	end
	local setXML = { elem = "PathOfBuilding" }
	local mode = { elem = "Mode", attrib = { mode = self.mode } }
	for _, val in ipairs({ self:CallMode("GetArgs") }) do
		local child = { elem = "Arg", attrib = { } }
		if type(val) == "number" then
			child.attrib.number = tostring(val)
		elseif type(val) == "boolean" then
			child.attrib.boolean = tostring(val)
		else
			child.attrib.string = tostring(val)
		end
		t_insert(mode, child)
	end

	-- if setting save is attempted and mode is nil something has gone very wrong
	if not mode.attrib.mode or not mode[1] then
		launch:ShowErrMsg("^1Error saving 'Settings.xml': mode element is invalid")
		return true
	end
	t_insert(setXML, mode)
	local accounts = { elem = "Accounts", attrib = { lastAccountName = self.lastAccountName, lastRealm = self.lastRealm, lastLeague = self.lastLeague, lastToken = self.lastToken, lastRefreshToken = self.lastRefreshToken, tokenExpiry = tostring(self.tokenExpiry) } }
	for accountName, account in pairs(self.gameAccounts) do
		t_insert(accounts, { elem = "Account", attrib = { accountName = accountName, sessionID = account.sessionID } })
	end
	t_insert(setXML, accounts)
	local sharedItemList = { elem = "SharedItems" }
	for _, verItem in ipairs(self.sharedItemList) do
		t_insert(sharedItemList, { elem = "Item", [1] = verItem.raw })
	end
	for _, sharedItemSet in ipairs(self.sharedItemSetList) do
		local set = { elem = "ItemSet", attrib = { title = sharedItemSet.title } }
		for slotName, verItem in pairs(sharedItemSet.slots) do
			t_insert(set, { elem = "Item", attrib = { slotName = slotName }, [1] = verItem.raw })
		end
		t_insert(sharedItemList, set)
	end
	t_insert(setXML, sharedItemList)
	t_insert(setXML, { elem = "Misc", attrib = {
		buildSortMode = self.buildSortMode,
		connectionProtocol = tostring(launch.connectionProtocol),
		proxyURL = launch.proxyURL,
		buildPath = (self.buildPath ~= self.defaultBuildPath and self.buildPath or nil),
		nodePowerTheme = self.nodePowerTheme,
		colorPositive = self.colorPositive,
		colorNegative = self.colorNegative,
		colorHighlight = self.colorHighlight,
		showThousandsSeparators = tostring(self.showThousandsSeparators),
		thousandsSeparator = self.thousandsSeparator,
		useCompactValues = tostring(self.useCompactValues),
		decimalSeparator = self.decimalSeparator,
		showTitlebarName = tostring(self.showTitlebarName),
		betaTest = tostring(self.betaTest),
		edgeSearchHighlight = tostring(self.edgeSearchHighlight),
		defaultGemQuality = tostring(self.defaultGemQuality or 0),
		defaultCharLevel = tostring(self.defaultCharLevel or 1),
		defaultItemAffixQuality = tostring(self.defaultItemAffixQuality or 0.5),
		lastExportedWebsite = self.lastExportedWebsite,
		showWarnings = tostring(self.showWarnings),
		slotOnlyTooltips = tostring(self.slotOnlyTooltips),
		migrateEldritchImplicits = tostring(self.migrateEldritchImplicits),
		notSupportedModTooltips = tostring(self.notSupportedModTooltips),
		invertSliderScrollDirection = tostring(self.invertSliderScrollDirection),
		disableDevAutoSave = tostring(self.disableDevAutoSave),
		showPublicBuilds = tostring(self.showPublicBuilds),
		showFlavourText = tostring(self.showFlavourText),
		showAnimations = tostring(self.showAnimations),
		showAllItemAffixes = tostring(self.showAllItemAffixes),
		disableScrollControlInteraction = tostring(self.disableScrollControlInteraction),
	} })
	local res, errMsg = common.xml.SaveXMLFile(setXML, self.userPath.."Settings.xml")
	if not res then
		launch:ShowErrMsg("Error saving 'Settings.xml': %s", errMsg)
		return true
	end
end

function main:ChangeUserPath(newUserPath, ignoreBuild)
	self.userPath = newUserPath
	MakeDir(self.userPath)
	self.defaultBuildPath = self.userPath.."Builds/"
	self.buildPath = self.defaultBuildPath
	MakeDir(self.buildPath)
	self:LoadSettings(ignoreBuild)
	self:LoadSharedItems()
end

function main:SetManifestBranch(branchName)
	local xml = require("xml")
	local manifestLocation = "manifest.xml"
	local localManXML = xml.LoadXMLFile(manifestLocation)
	if not localManXML then
		manifestLocation = "../manifest.xml"
		localManXML = xml.LoadXMLFile(manifestLocation)
	end
	if localManXML and localManXML[1].elem == "PoBVersion" then
		for _, node in ipairs(localManXML[1]) do
			if type(node) == "table" then
				if node.elem == "Version" then
					node.attrib.branch = branchName
				end
			end
		end
	end
	xml.SaveXMLFile(localManXML[1], manifestLocation)
end

-- Register a dialog state record; the record holds the model objects that drive it
function main:OpenPopup(width, height, title, controls, enterControl, defaultControl, escapeControl, scrollBarFunc, resizeFunc)
	local popup = { width = width, height = height, title = title, controls = controls }
	t_insert(self.popups, 1, popup)
	return popup
end

function main:ClosePopup()
	t_remove(self.popups, 1)
end

-- Report a message that the UI would have shown as a popup
function main:OpenMessagePopup(title, msg)
	ConPrintf("%s: %s", title, msg)
end

-- Report a file that cannot be read due to cloud provider unavailability
function main:OpenCloudErrorPopup(fileName)
	local provider, _, status = GetCloudProvider(fileName)
	ConPrintf('^1Error: file offline "%s" provider: "%s" status: "%s"', fileName or "?", provider, status)
end

function main:StatColor(stat, base, limit)
	if limit and stat > limit then
		return colorCodes.NEGATIVE
	elseif base and stat ~= base then
		return colorCodes.MAGIC
	else
		return "^7"
	end
end

return main
