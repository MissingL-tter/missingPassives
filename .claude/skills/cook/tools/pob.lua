-- Headless PoB bootstrap. Every tool here starts with:
--     local pob = dofile("../.claude/skills/cook/tools/pob.lua")
-- Requires cwd == src/ (HeadlessWrapper.lua resolves relatively).
-- No cpath line -> "module 'lua-utf8' not found"; no path line -> "module 'xml' not found".
-- Pure-Lua deps live in runtime/lua/, native ones are .dll files loose in runtime/.
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
package.cpath = "../runtime/?.dll;" .. package.cpath

dofile("HeadlessWrapper.lua")

local pob = {}

-- data is a global populated during build construction, not by the wrapper.
function pob.data()
	if not _G.data then newBuild() end
	return _G.data
end

-- Load a build .xml and settle the calcs. Returns the global `build`.
function pob.load(path)
	local f = assert(io.open(path, "r"), "cannot open " .. tostring(path))
	local xml = f:read("*a")
	f:close()
	loadBuildFromXML(xml, "tool")
	runCallback("OnFrame")
	return _G.build
end

-- Recalculate after mutating the build.
function pob.refresh()
	_G.build.buildFlag = true
	runCallback("OnFrame")
	return _G.build.calcsTab.mainOutput or {}
end

-- Serialize the loaded build to disk through PoB's own writer. Items save from
-- itemsTab.items (+ itemOrderList for anything added at runtime); slot assignments save
-- from the item SETS, so mutating slot.selItemId alone is lost.
function pob.save(path)
	local xml = _G.build:SaveDB(path)
	assert(xml and #xml > 1000, "SaveDB returned nothing")
	local f = assert(io.open(path, "w"))
	f:write(xml)
	f:close()
end

return pob
