-- Mechanical leaf prune, snapshot-safe. Run from src/ once every constraint measures green:
--   luajit ../.claude/skills/cook/tools/prune.lua "Builds/My Build.xml"
--
-- The two hard-won rules this tool encodes (faq.md "Tree surgery"):
--   * DeallocNode drops ALL dependents; restoring only the target node loses branches.
--     Every candidate dealloc is undone by restoring the full snapshotted allocation set.
--   * Guards are not retyped from memory: non-regression is enforced on the whole stat
--     vector below plus keystone-set equality (a hand-typed guard list once let CI get
--     pruned) and absolute attribute sufficiency (gems must still function).
--
-- A leaf is a node whose dealloc frees exactly one point. Passes repeat until a full pass
-- removes nothing. Saves only when at least one point was freed; writes
-- "<build>.preprune.bak" beside the scratchless original first.
local BUILD = arg[1] or os.getenv("BUILDXML")
assert(BUILD, 'usage: luajit prune.lua "Builds/My Build.xml"')
local pob = dofile("../.claude/skills/cook/tools/pob.lua")
local build = pob.load(BUILD)
local spec = build.spec

do -- backup before any destructive pass
	local src = assert(io.open(BUILD, "r"))
	local xml = src:read("*a")
	src:close()
	local dst = assert(io.open(BUILD .. ".preprune.bak", "w"))
	dst:write(xml)
	dst:close()
end

local function snapshot()
	local ids = {}
	for id in pairs(spec.allocNodes) do ids[id] = true end
	return ids
end
local function restore(ids)
	for id, n in pairs(spec.allocNodes) do
		if not ids[id] then n.alloc = false; spec.allocNodes[id] = nil end
	end
	for id in pairs(ids) do
		local n = spec.nodes[id]
		if n and not n.alloc then n.alloc = true; spec.allocNodes[id] = n end
	end
	spec:BuildAllDependsAndPaths()
end
local function keystones()
	local set = {}
	for id, n in pairs(spec.allocNodes) do
		if n.type == "Keystone" then set[#set + 1] = n.name or tostring(id) end
	end
	table.sort(set)
	return table.concat(set, "|")
end
local function meas()
	local o = pob.refresh()
	return {
		dps = (o.WithDotDPS and o.WithDotDPS > 0) and o.WithDotDPS or (o.TotalDPS or 0),
		es = o.EnergyShield or 0,
		phys = o.PhysicalMaximumHitTaken or 0,
		ele = math.min(o.FireMaximumHitTaken or 0, o.ColdMaximumHitTaken or 0,
			o.LightningMaximumHitTaken or 0),
		resMin = math.min(o.FireResist or 0, o.ColdResist or 0, o.LightningResist or 0),
		trig = o.SkillTriggerRate or 0,
		mana = o.ManaUnreserved or 0,
		ks = keystones(),
		strOk = (o.Str or 0) >= (o.ReqStr or 0),
		dexOk = (o.Dex or 0) >= (o.ReqDex or 0),
		intOk = (o.Int or 0) >= (o.ReqInt or 0),
	}
end
local EPS = 1e-6
local function regressed(s, base)
	if s.dps < base.dps - EPS then return "dps" end
	if s.es < base.es - EPS then return "es" end
	if s.phys < base.phys - EPS then return "phys max hit" end
	if s.ele < base.ele - EPS then return "ele max hit" end
	if s.resMin < base.resMin - EPS then return "res" end
	if s.trig < base.trig - EPS then return "trigger rate" end
	if s.mana < base.mana - EPS then return "unreserved mana" end
	if s.ks ~= base.ks then return "keystone set" end
	if not (s.strOk and s.dexOk and s.intOk) then return "attribute requirements" end
	return nil
end

local base = meas()
local startPts = spec:CountAllocNodes()
print(string.format("baseline: pts=%d dps=%.0f es=%.0f phys=%.0f ele=%.0f keystones=[%s]",
	startPts, base.dps, base.es, base.phys, base.ele, base.ks))

local totalRemoved = 0
repeat
	local removed = 0
	local ids = {}
	for id, n in pairs(spec.allocNodes) do
		if n.type ~= "ClassStart" and n.type ~= "AscendClassStart" and not n.isJewelSocket then
			ids[#ids + 1] = id
		end
	end
	table.sort(ids)
	for _, id in ipairs(ids) do
		local n = spec.nodes[id]
		if n and n.alloc then
			local snap = snapshot()
			local before = spec:CountAllocNodes()
			spec:DeallocNode(n)
			local after = spec:CountAllocNodes()
			if before - after == 1 then
				local s = meas()
				if regressed(s, base) then
					restore(snap)
					pob.refresh()
				else
					removed = removed + 1
					base = s
					print(string.format("  pruned %d %s", id, n.name or "?"))
				end
			else
				restore(snap)
				pob.refresh()
			end
		end
	end
	totalRemoved = totalRemoved + removed
	print(string.format("pass: removed %d, points %d", removed, spec:CountAllocNodes()))
until removed == 0

local fin = meas()
print(string.format("freed %d points: pts=%d dps=%.0f es=%.0f phys=%.0f ele=%.0f keystones=[%s]",
	totalRemoved, spec:CountAllocNodes(), fin.dps, fin.es, fin.phys, fin.ele, fin.ks))
if totalRemoved > 0 then
	spec:AddUndoState()
	pob.save(BUILD)
	print("saved " .. BUILD .. "  (backup at .preprune.bak)")
else
	print("nothing pruned; file untouched")
end
