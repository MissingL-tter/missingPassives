-- RPC host for the headless engine. Spawned by the Electron main process with
-- cwd = <repo>/src; speaks line-delimited JSON over stdio.
--
-- stdin:  {"id":1,"method":"getBuild","params":{...}}
-- stdout: @@RPC@@{"id":1,"result":{...}}   (unprefixed lines are engine logs)
--
-- One build load per process: in-process reloads keep stale engine state, so
-- the Electron side restarts this process for every build load.
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
package.cpath = "../runtime/?.dll;" .. package.cpath

local dkjson = require("dkjson")

dofile("HeadlessWrapper.lua")

-- Engine logs must not interleave with protocol output
io.stdout:flush()
function ConPrintf(fmt, ...)
	io.stderr:write(string.format(fmt, ...), "\n")
	io.stderr:flush()
end

-- HeadlessWrapper stubs Deflate/Inflate to "" (literal TODOs). Timeless-jewel
-- LUTs and build codes need the real thing; bind runtime/zlib1.dll.
do
	local haveFfi, ffi = pcall(require, "ffi")
	if haveFfi then
		pcall(ffi.cdef, [[
int compress2(uint8_t *dest, unsigned long *destLen, const uint8_t *source, unsigned long sourceLen, int level);
int uncompress(uint8_t *dest, unsigned long *destLen, const uint8_t *source, unsigned long sourceLen);
]])
		local haveZlib, zlib = pcall(ffi.load, "../runtime/zlib1.dll")
		if haveZlib then
			function Inflate(data)
				local cap = #data * 6 + 4096
				for _ = 1, 8 do
					local buf = ffi.new("uint8_t[?]", cap)
					local len = ffi.new("unsigned long[1]", cap)
					local res = zlib.uncompress(buf, len, data, #data)
					if res == 0 then return ffi.string(buf, len[0]) end
					if res ~= -5 then return nil end -- only retry Z_BUF_ERROR
					cap = cap * 2
				end
			end
			function Deflate(data)
				local cap = #data + math.floor(#data / 1000) + 128
				local buf = ffi.new("uint8_t[?]", cap)
				local len = ffi.new("unsigned long[1]", cap)
				if zlib.compress2(buf, len, data, #data, 9) == 0 then
					return ffi.string(buf, len[0])
				end
			end
		end
	end
end

local function jsonArray(tbl)
	return setmetatable(tbl, { __jsontype = "array" })
end

local loadedOnce = false

local function serializeStatBox()
	local out = jsonArray({ })
	for _, entry in ipairs(build.statBoxList or { }) do
		local cells = jsonArray({ })
		for i, cell in ipairs(entry) do
			cells[i] = tostring(cell)
		end
		out[#out + 1] = { height = entry.height, align = entry.align, x = entry.x, cells = cells }
	end
	return out
end

local function serializeWarnings()
	local out = jsonArray({ })
	for _, line in ipairs(build.warningLines or { }) do
		out[#out + 1] = line
	end
	return out
end

local function serializeSkills()
	local groups = jsonArray({ })
	for index, socketGroup in ipairs(build.skillsTab.socketGroupList) do
		groups[#groups + 1] = {
			index = index,
			label = socketGroup.displayLabel or socketGroup.label or "?",
			enabled = socketGroup.enabled or false,
			slot = socketGroup.slot,
		}
	end
	return {
		groups = groups,
		mainSocketGroup = build.mainSocketGroup or 1,
	}
end

local function serializeSummary()
	local spec = build.spec
	local pointsUsed, ascUsed = spec:CountAllocNodes()
	return {
		buildName = build.buildName,
		className = spec.curClassName,
		ascendClassName = spec.curAscendClassName,
		level = build.characterLevel,
		pointsUsed = pointsUsed,
		pointsMax = build.pointsMax,
		ascPointsUsed = ascUsed,
		treeVersion = spec.treeVersion,
	}
end

-- Allocation state and identifiers that change as the build changes; nodeCount
-- lets the renderer detect cluster-jewel subgraph changes and refetch geometry
local function serializeTreeState()
	local spec = build.spec
	local alloc = jsonArray({ })
	for nodeId in pairs(spec.allocNodes) do
		alloc[#alloc + 1] = nodeId
	end
	local nodeCount = 0
	for _ in pairs(spec.nodes) do
		nodeCount = nodeCount + 1
	end
	return {
		alloc = alloc,
		nodeCount = nodeCount,
		curClassId = spec.curClassId,
		curClassName = spec.curClassName,
		curAscendClassName = spec.curAscendClassName,
		searchStr = build.treeTab.viewer.searchStr or "",
	}
end

local function fullState()
	return {
		summary = serializeSummary(),
		skills = serializeSkills(),
		statBox = serializeStatBox(),
		warnings = serializeWarnings(),
		treeState = serializeTreeState(),
	}
end

-- Static-ish tree geometry for the active spec: every placed node (including
-- cluster jewel subgraphs) plus the connector list, using the same eligibility
-- rules the old tree view applied
local function serializeTree()
	local spec = build.spec
	local tree = spec.tree
	local nodes = jsonArray({ })
	for nodeId, node in pairs(spec.nodes) do
		if node.x and node.y and not node.isProxy and not (node.group and node.group.isProxy) then
			local entry = {
				id = node.id,
				x = node.x,
				y = node.y,
				type = node.type,
				name = node.dn or node.name,
				ascend = node.ascendancyName,
				g = node.g,
				o = node.o,
				angle = node.angle,
				isJewelSocket = node.isJewelSocket or nil,
			}
			if node.group then
				entry.gx = node.group.x
				entry.gy = node.group.y
			end
			if node.sd and node.sd[1] then
				local sd = jsonArray({ })
				for i, line in ipairs(node.sd) do
					sd[i] = line
				end
				entry.sd = sd
			end
			nodes[#nodes + 1] = entry
		end
	end

	local edges = jsonArray({ })
	local seenEdges = { }
	for _, node in pairs(spec.nodes) do
		for _, otherId in ipairs(node.linkedId or { }) do
			local other = spec.nodes[tonumber(otherId) or otherId]
			if other then
				local key = node.id < other.id and (node.id .. ":" .. other.id) or (other.id .. ":" .. node.id)
				if not seenEdges[key]
					and node.type ~= "ClassStart" and other.type ~= "ClassStart"
					and node.type ~= "Mastery" and other.type ~= "Mastery"
					and node.ascendancyName == other.ascendancyName
					and not node.isProxy and not other.isProxy
					and node.x and node.y and other.x and other.y then
					seenEdges[key] = true
					local edge = { a = node.id, b = other.id }
					if node.g and node.g == other.g and node.o == other.o
						and node.group and node.group == other.group
						and node.angle and other.angle then
						edge.arc = true
					end
					edges[#edges + 1] = edge
				end
			end
		end
	end

	local classes = jsonArray({ })
	for classId, class in pairs(tree.classes) do
		classes[#classes + 1] = { id = classId, name = class.name, startNodeId = class.startNodeId }
	end

	return {
		nodes = nodes,
		edges = edges,
		classes = classes,
		treeVersion = spec.treeVersion,
		nodeCount = #nodes,
	}
end

local methods = { }

function methods.ping()
	return { pong = true, versionNumber = launch.versionNumber, treeVersion = latestTreeVersion }
end

function methods.newBuild()
	assert(not loadedOnce, "engine process already holds a build; restart it to load another")
	loadedOnce = true
	newBuild()
	return fullState()
end

function methods.loadBuildXML(params)
	assert(not loadedOnce, "engine process already holds a build; restart it to load another")
	assert(type(params.xml) == "string", "loadBuildXML needs params.xml")
	loadedOnce = true
	loadBuildFromXML(params.xml, params.name or "Imported build")
	assert(build.spec, "build failed to load")
	return fullState()
end

function methods.getBuild()
	return fullState()
end

function methods.getOutput(params)
	local out = { }
	if params and params.names then
		for _, name in ipairs(params.names) do
			local v = build.calcsTab.mainOutput[name]
			local t = type(v)
			if t == "number" or t == "string" or t == "boolean" then
				out[name] = v
			end
		end
	else
		for k, v in pairs(build.calcsTab.mainOutput) do
			local t = type(v)
			if t == "number" or t == "string" or t == "boolean" then
				out[k] = v
			end
		end
	end
	return out
end

function methods.selectMainSkill(params)
	assert(type(params.index) == "number", "selectMainSkill needs params.index")
	build.mainSocketGroup = params.index
	build.modFlag = true
	build.buildFlag = true
	runCallback("OnFrame")
	return fullState()
end

function methods.saveBuildXML()
	local xml = build:SaveDB("rpc")
	assert(xml, "SaveDB returned nothing")
	return { xml = xml }
end

function methods.getTree()
	return serializeTree()
end

-- Path preview for hovering: the nodes that allocating would add, or the
-- dependents that deallocating would remove
function methods.getNodePath(params)
	local node = build.spec.nodes[params.id]
	assert(node, "unknown node id: " .. tostring(params.id))
	local path = jsonArray({ })
	for _, pathNode in ipairs(node.path or { }) do
		path[#path + 1] = pathNode.id
	end
	local depends = jsonArray({ })
	for _, depNode in ipairs(node.depends or { }) do
		depends[#depends + 1] = depNode.id
	end
	return { path = path, depends = depends, pathDist = node.pathDist, alloc = node.alloc or false }
end

-- Mirrors the old tree view's click handling: dealloc an allocated node (and
-- its dependents), or alloc along the path; masteries go through
-- selectMasteryEffect instead
function methods.toggleNode(params)
	local node = build.spec.nodes[params.id]
	assert(node, "unknown node id: " .. tostring(params.id))
	if node.alloc then
		build.spec:DeallocNode(node)
	elseif node.type == "Mastery" then
		error("mastery nodes need selectMasteryEffect")
	elseif node.path then
		build.spec:AllocNode(node)
	else
		error("node is not reachable")
	end
	build.spec:AddUndoState()
	build.modFlag = true
	build.buildFlag = true
	runCallback("OnFrame")
	return fullState()
end

function methods.getMasteryEffects(params)
	local node = build.spec.nodes[params.id]
	assert(node and node.type == "Mastery", "not a mastery node: " .. tostring(params.id))
	local effects = jsonArray({ })
	for _, effect in pairs(node.masteryEffects or { }) do
		local assignedNodeId = isValueInTable(build.spec.masterySelections, effect.effect)
		if not assignedNodeId or assignedNodeId == node.id then
			effects[#effects + 1] = { id = effect.effect, label = table.concat(effect.stats or { }, " / ") }
		end
	end
	return { effects = effects, selected = build.spec.masterySelections[node.id] }
end

function methods.selectMasteryEffect(params)
	local node = build.spec.nodes[params.id]
	assert(node and node.type == "Mastery", "not a mastery node: " .. tostring(params.id))
	build.treeTab:SelectMasteryEffect(node, params.effectId)
	runCallback("OnFrame")
	return fullState()
end

-- Tree search, ported from the old tree view: quoted phrases and words, Lua
-- patterns with (a|b) alternation via string.matchOrPattern, and "oil:" to
-- search anoint recipes; matches against name, stats, mod names, and type
local function prepSearch(search)
	search = search:lower()
	local searchWords = {}
	for matchstring, v in search:gmatch('"([^"]*)"') do
		searchWords[#searchWords+1] = matchstring
		search = search:gsub('"'..matchstring:gsub("([%(%)])", "%%%1")..'"', "")
	end
	for matchstring, v in search:gmatch("(%S*)") do
		if matchstring:match("%S") ~= nil then
			searchWords[#searchWords+1] = matchstring
		end
	end
	return searchWords
end

local function nodeMatchesSearch(node, searchParams)
	if node.type == "ClassStart" or (node.type == "Mastery" and not node.masteryEffects) then
		return
	end

	local needMatches = copyTable(searchParams)
	local err

	local function search(haystack, need)
		for i=#need, 1, -1 do
			if haystack:matchOrPattern(need[i]) then
				table.remove(need, i)
			end
		end
		return need
	end

	-- Check recipes
	if needMatches[1] == "oil:" then
		if node.recipe then
			for _, recipeName in ipairs(node.recipe) do
				err, needMatches = PCall(search, recipeName:gsub("Oil",""):lower(), needMatches)
				if err then return false end
				if #needMatches == 1 and needMatches[1] == "oil:" then
					return true
				end
			end
		end
		return false
	end

	-- Check node name
	err, needMatches = PCall(search, node.dn:lower(), needMatches)
	if err then return false end
	if #needMatches == 0 then
		return true
	end

	-- Check node description
	for index, line in ipairs(node.sd) do
		-- Check display text first
		err, needMatches = PCall(search, line:lower(), needMatches)
		if err then return false end
		if #needMatches == 0 then
			return true
		end
		if #needMatches > 0 and node.mods[index].list then
			-- Then check modifiers
			for _, mod in ipairs(node.mods[index].list) do
				err, needMatches = PCall(search, mod.name, needMatches)
				if err then return false end
				if #needMatches == 0 then
					return true
				end
			end
		end
	end

	-- Check node type
	err, needMatches = PCall(search, node.type:lower(), needMatches)
	if err then return false end
	if #needMatches == 0 then
		return true
	end
end

function methods.searchNodes(params)
	local query = tostring(params and params.query or "")
	build.treeTab.viewer.searchStr = query
	build.treeTab.searchFlag = query ~= build.treeTab.viewer.searchStrSaved
	local matches = jsonArray({ })
	local searchParams = prepSearch(query)
	if #searchParams > 0 then
		for nodeId, node in pairs(build.spec.nodes) do
			if nodeMatchesSearch(node, searchParams) then
				matches[#matches + 1] = nodeId
			end
		end
	end
	return { matches = matches, query = query }
end

function methods.undoTree()
	build.spec:Undo()
	build.buildFlag = true
	runCallback("OnFrame")
	return fullState()
end

function methods.redoTree()
	build.spec:Redo()
	build.buildFlag = true
	runCallback("OnFrame")
	return fullState()
end

local function respond(payload)
	io.stdout:write("@@RPC@@", dkjson.encode(payload), "\n")
	io.stdout:flush()
end

respond({ ready = true })

while true do
	local line = io.stdin:read("*l")
	if not line then
		break
	end
	if line:match("%S") then
		local request, _, decodeErr = dkjson.decode(line)
		if not request or not request.method then
			respond({ id = request and request.id, error = "bad request: " .. tostring(decodeErr or "no method") })
		else
			local method = methods[request.method]
			if not method then
				respond({ id = request.id, error = "unknown method: " .. tostring(request.method) })
			else
				local ok, result = pcall(method, request.params)
				if ok then
					respond({ id = request.id, result = result })
				else
					respond({ id = request.id, error = tostring(result) })
				end
			end
		end
	end
end
