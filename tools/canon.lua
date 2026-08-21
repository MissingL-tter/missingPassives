-- Canonical serialiser for parser results, matching modparser/canon.go byte
-- for byte: every table becomes a JSON object with sorted keys, array indices
-- stringified, whole numbers without a decimal point, functions as
-- {"__fn":true}.
local canon = {}

local escapes = {
	["\""] = "\\\"",
	["\\"] = "\\\\",
	["\b"] = "\\b",
	["\f"] = "\\f",
	["\n"] = "\\n",
	["\r"] = "\\r",
	["\t"] = "\\t",
}

local function quote(s)
	return '"' .. s:gsub("[%c\"\\]", function(c)
		return escapes[c] or string.format("\\u%04x", c:byte())
	end) .. '"'
end
canon.quote = quote

local function num(v)
	if v ~= v or v == math.huge or v == -math.huge then
		return quote(tostring(v))
	end
	if v == math.floor(v) and v < 1e15 and v > -1e15 then
		return string.format("%d", v)
	end
	return string.format("%.14g", v)
end

function canon.encode(v)
	local t = type(v)
	if v == nil then
		return "null"
	elseif t == "boolean" then
		return tostring(v)
	elseif t == "number" then
		return num(v)
	elseif t == "string" then
		return quote(v)
	elseif t == "function" then
		return '{"__fn":true}'
	elseif t ~= "table" then
		return quote(tostring(v))
	end
	local keys = {}
	for k in pairs(v) do
		keys[#keys + 1] = tostring(k)
	end
	table.sort(keys)
	local parts = {}
	for _, k in ipairs(keys) do
		-- rawget: pairs() only yields real keys, and __index metamethods
		-- (e.g. data.costs' resource lookup) must not fire here.
		local val = rawget(v, k)
		if val == nil and tonumber(k) then
			val = rawget(v, tonumber(k))
		end
		parts[#parts + 1] = quote(k) .. ":" .. canon.encode(val)
	end
	return "{" .. table.concat(parts, ",") .. "}"
end

return canon
