-- Uploads a build to pobb.in and prints the link. Run from src/:
--   luajit ../.claude/skills/cook/tools/export.lua "Builds/My Build.xml"
--
-- HeadlessWrapper stubs Deflate to return "" (a literal "TODO: Might need this"), so PoB's
-- own export path silently yields an empty code headless. Deflate through runtime/zlib1.dll
-- via ffi instead, then base64 with PoB's own encoder - byte-identical to the UI's
-- "export build code".
local BUILD = arg[1] or os.getenv("BUILDXML")
assert(BUILD, 'usage: luajit export.lua "Builds/My Build.xml"')

local ffi = require("ffi")
ffi.cdef([[
int compress2(uint8_t *dest, unsigned long *destLen,
              const uint8_t *source, unsigned long sourceLen, int level);
]])
local zlib = ffi.load("../runtime/zlib1.dll")

local pob = dofile("../.claude/skills/cook/tools/pob.lua")
local build = pob.load(BUILD)
local xml = build:SaveDB("code")
assert(xml and #xml > 1000, "SaveDB returned nothing")

local cap = #xml + math.floor(#xml / 1000) + 128
local buf = ffi.new("uint8_t[?]", cap)
local len = ffi.new("unsigned long[1]", cap)
assert(zlib.compress2(buf, len, xml, #xml, 9) == 0, "zlib compress2 failed")
local code = common.base64.encode(ffi.string(buf, len[0])):gsub("+", "-"):gsub("/", "_")

local codePath = (os.getenv("TEMP") or ".") .. "\\pobcode.txt"
local f = assert(io.open(codePath, "w"))
f:write(code)
f:close()

-- curl ships with Windows 10+; -f makes HTTP errors exit non-zero
local p = assert(io.popen('curl -sf -X POST "https://pobb.in/pob/" '
	.. '-H "Content-Type: text/plain" --data-binary "@' .. codePath .. '"'))
local id = p:read("*a"):gsub("%s+", "")
p:close()
assert(id ~= "" and not id:find("[^%w%-_]"), "pobb.in upload failed: " .. tostring(id))
print("https://pobb.in/" .. id)
