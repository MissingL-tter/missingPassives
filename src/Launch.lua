-- Path of Building
--
-- Module: Launch
-- Program entry point; loads and runs the Main module within a protected environment
--

local startTime = GetTime()
APP_NAME = "Path of Building"

launch = { }
SetMainObject(launch)
jit.opt.start('maxtrace=4000','maxmcode=8192')
collectgarbage("setpause", 400)

function launch:OnInit()
	self.devMode = false
	self.installedMode = false
	self.versionNumber = "?"
	self.versionBranch = "?"
	self.versionPlatform = "?"
	self.lastUpdateCheck = GetTime()
	self.subScripts = { }
	self.startTime = startTime
	local firstRunFile = io.open("first.run", "r")
	if firstRunFile then
		firstRunFile:close()
		os.remove("first.run")
		-- This is a fresh installation
		-- Perform an immediate update to download the latest version
		ConPrintf("Please wait while we complete installation...\n")
		local updateMode, errMsg = LoadModule("UpdateCheck")
		if not updateMode then
			self.updateErrMsg = errMsg
		elseif updateMode ~= "none" then
			self:ApplyUpdate(updateMode)
			return
		end
	end
	local xml = require("xml")
	local localManXML = xml.LoadXMLFile("manifest.xml") or xml.LoadXMLFile("../manifest.xml")
	if localManXML and localManXML[1].elem == "PoBVersion" then
		for _, node in ipairs(localManXML[1]) do
			if type(node) == "table" then
				if node.elem == "Version" then
					self.versionNumber = node.attrib.number
					self.versionBranch = node.attrib.branch
					self.versionPlatform = node.attrib.platform
				end
			end
		end
	end
	local installedFile = io.open("installed.cfg", "r")
	if installedFile then
		self.installedMode = true
		installedFile:close()
	end
	ConPrintf("Loading main script...")
	local errMsg
	errMsg, self.main = PLoadModule("Modules/Main")
	if errMsg then
		self:ShowErrMsg("Error loading main script: %s", errMsg)
	elseif not self.main then
		self:ShowErrMsg("Error loading main script: no object returned")
	elseif self.main.Init then
		errMsg = PCall(self.main.Init, self.main)
		if errMsg then
			self:ShowErrMsg("In 'Init': %s", errMsg)
		end
	end
end

function launch:CanExit()
	if self.main and self.main.CanExit and not self.promptMsg then
		local errMsg, ret = PCall(self.main.CanExit, self.main)
		if errMsg then
			self:ShowErrMsg("In 'CanExit': %s", errMsg)
			return false
		else
			return ret
		end
	end
	return true
end

function launch:OnExit()
	if self.main and self.main.Shutdown then
		PCall(self.main.Shutdown, self.main)
	end
end

function launch:OnFrame()
	if self.main then
		if self.main.OnFrame then
			local errMsg = PCall(self.main.OnFrame, self.main)
			if errMsg then
				self:ShowErrMsg("In 'OnFrame': %s", errMsg)
			end
		end
	end
	if self.doRestart then
		Restart()
	end
end

function launch:OnSubCall(func, ...)
	if func == "UpdateProgress" then
		self.updateProgress = string.format(...)
	end
	if _G[func] then
		return _G[func](...)
	end
end

function launch:OnSubError(id, errMsg)
	if self.subScripts[id].type == "UPDATE" then
		self:ShowErrMsg("In update thread: %s", errMsg)
		self.updateCheckRunning = false
	elseif self.subScripts[id].type == "DOWNLOAD" then
		local errMsg = PCall(self.subScripts[id].callback, nil, errMsg)
		if errMsg then
			self:ShowErrMsg("In download callback: %s", errMsg)
		end
	end
	self.subScripts[id] = nil
end

function launch:OnSubFinished(id, ...)
	if self.subScripts[id].type == "UPDATE" then
		self.updateAvailable, self.updateErrMsg = ...
		self.updateCheckRunning = false
		if self.updateCheckBackground and self.updateAvailable == "none" then
			self.updateAvailable = nil
		end
	elseif self.subScripts[id].type == "DOWNLOAD" then
		local errMsg = PCall(self.subScripts[id].callback, ...)
		if errMsg then
			self:ShowErrMsg("In download callback: %s", errMsg)
		end
	elseif self.subScripts[id].type == "CUSTOM" then
		if self.subScripts[id].callback then
			local errMsg = PCall(self.subScripts[id].callback, ...)
			if errMsg then
				self:ShowErrMsg("In subscript callback: %s", errMsg)
			end
		end
	end
	self.subScripts[id] = nil
end

function launch:RegisterSubScript(id, callback)
	if id then
		self.subScripts[id] = {
			type = "CUSTOM",
			callback = callback,
		}
	end
end

---Download the given page in the background, and calls the provided callback function when done:
---@param url string
---@param callback fun(response:table, errMsg:string) @ response = { header, body }
---@param params? table @ params = { header, body }
function launch:DownloadPage(url, callback, params)
	params = params or {}
	local script = [[
		local url, requestHeader, requestBody, connectionProtocol, proxyURL, noSSL = ...
		local responseHeader = ""
		local responseBody = ""
		ConPrintf("Downloading page at: %s", url)
		local curl = require("lcurl.safe")
		local easy = curl.easy()
		if requestHeader then
			local header = {}
			for s in requestHeader:gmatch("[^\r\n]+") do
    			table.insert(header, s)
			end
			easy:setopt(curl.OPT_HTTPHEADER, header)
		end
		easy:setopt_url(url)
		easy:setopt(curl.OPT_USERAGENT, "Path of Building/]]..self.versionNumber..[[")
		easy:setopt(curl.OPT_ACCEPT_ENCODING, "")
		easy:setopt(curl.OPT_FOLLOWLOCATION, 1)
		if requestBody then
			easy:setopt(curl.OPT_POST, true)
			easy:setopt(curl.OPT_POSTFIELDS, requestBody)
		end
		if connectionProtocol then
			easy:setopt(curl.OPT_IPRESOLVE, connectionProtocol)
		end
		if proxyURL then
			easy:setopt(curl.OPT_PROXY, proxyURL)
		end
		if noSSL then
			easy:setopt(curl.OPT_SSL_VERIFYPEER, 0)
			easy:setopt(curl.OPT_SSL_VERIFYHOST, 0)
		end
		easy:setopt_headerfunction(function(data)
			responseHeader = responseHeader .. data
			return true
		end)
		easy:setopt_writefunction(function(data)
			responseBody = responseBody .. data
			return true
		end)
		local _, error = easy:perform()
		local code = easy:getinfo(curl.INFO_RESPONSE_CODE)
		easy:close()
		local errMsg
		if error then
			errMsg = error:msg()
		elseif code ~= 200 then
			errMsg = "Response code: "..code
		elseif #responseBody == 0 then
			errMsg = "No data returned"
		end
		ConPrintf("Download complete. Status: %s", errMsg or "OK")
		return responseBody, errMsg, responseHeader
	]]
	local id = LaunchSubScript(script, "", "ConPrintf", url, params.header, params.body, self.connectionProtocol, self.proxyURL, self.noSSL or false)
	if id then
		self.subScripts[id] = {
			type = "DOWNLOAD",
			callback = function(responseBody, errMsg, responseHeader)
				callback({header=responseHeader, body=responseBody}, errMsg)
			end
		}
	end
end

function launch:ApplyUpdate(mode)
	if mode == "basic" then
		-- Need to revert to the basic environment to fully apply the update
		LoadModule("UpdateApply", "Update/opFile.txt")
		SpawnProcess(GetRuntimePath()..'/Update', 'UpdateApply.lua Update/opFileRuntime.txt')
		Exit()
	elseif mode == "normal" then
		-- Update can be applied while normal environment is running
		LoadModule("UpdateApply", "Update/opFile.txt")
		Restart()
		self.doRestart = "Updating..."
	end
end

function launch:CheckForUpdate(inBackground)
	if self.updateCheckRunning then
		return
	end
	self.updateCheckBackground = inBackground
	self.updateMsg = "Initialising..."
	self.updateProgress = "Checking..."
	self.lastUpdateCheck = GetTime()
	local update = io.open("UpdateCheck.lua", "r")
	local id = LaunchSubScript(update:read("*a"), "GetScriptPath,GetRuntimePath,GetWorkDir,MakeDir", "ConPrintf,UpdateProgress", self.connectionProtocol, self.proxyURL, self.noSSL or false)
	if id then
		self.subScripts[id] = {
			type = "UPDATE"
		}
		self.updateCheckRunning = true
	end
	update:close()
end

function launch:ShowPrompt(r, g, b, str, func)
	self.promptMsg = str
	self.promptCol = {r, g, b}
	self.promptFunc = func
end

function launch:ShowErrMsg(fmt, ...)
	if not self.promptMsg then
		local version = self.versionNumber and
			"^8v"..self.versionNumber..(self.versionBranch and " "..self.versionBranch or "")
			or ""
		self:ShowPrompt(1, 0, 0, "^1Error:\n\n^0"..string.format(fmt, ...).."\n"..version)
	end
end
