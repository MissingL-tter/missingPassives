-- Headless PoB bootstrap. Every tool in this directory starts with:
--     local pob = dofile("../.claude/skills/cook/tools/pob.lua")
-- Requires cwd == src/ (HeadlessWrapper.lua resolves relatively).
--
-- Without the cpath line you get "module 'lua-utf8' not found"; without the path
-- line, "module 'xml' not found". Pure-Lua deps live in runtime/lua/, native ones
-- are .dll files loose in runtime/.
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

return pob
