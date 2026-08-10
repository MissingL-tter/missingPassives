-- Headless PoB bootstrap. Every tool here starts with:
--     local pob = dofile("../.claude/skills/cook/tools/pob.lua")
-- Requires cwd == src/ (HeadlessWrapper.lua resolves relatively).
-- No cpath line -> "module 'lua-utf8' not found"; no path line -> "module 'xml' not found".
-- Pure-Lua deps live in runtime/lua/, native ones are .dll files loose in runtime/.
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
package.cpath = "../runtime/?.dll;" .. package.cpath

dofile("HeadlessWrapper.lua")

-- HeadlessWrapper stubs Deflate/Inflate to "" (literal TODOs). Timeless-jewel LUTs and build
-- codes need the real thing; bind runtime/zlib1.dll. The zip files are raw zlib streams, not
-- PK archives. pcall the cdef: export.lua declares compress2 first when it runs.
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
