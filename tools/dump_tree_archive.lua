-- Dumps a freshly built PassiveTree — before any calc can stamp item
-- sources onto shared node mod lists — for the tree port's archive
-- comparison. Run from .archive/src/:
--
--   luajit ../../tools/dump_tree_archive.lua [version]
--
-- version defaults to 3_29. Writes test/testdata/tree_archive.jsonl:
--   meta        - version, counts, class table
--   node.<id>   - one record per processed tree node
--   legion.<id> - one record per legion (timeless) passive
--   masteryEffects, keystoneMap, notableMap, ascendancyMap,
--   clusterNodeMap, sockets (nodesInRadius per radius index)

local version = arg and arg[1] or "3_29"

package.cpath = "../runtime/?.dll;" .. package.cpath
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
dofile("HeadlessWrapper.lua")

local canon = dofile("../../tools/canon.lua")

data.setJewelRadiiGlobally(version)
local tree = new("PassiveTree", version)

local out = assert(io.open("../../test/testdata/tree_archive.jsonl", "w"))
local function emit(name, value)
	out:write('{"k":', canon.quote(name), ',"c":', canon.quote(canon.encode(value)), "}\n")
end

local function modArray(list)
	local o = {}
	for i, mod in ipairs(list or {}) do
		o[i] = mod
	end
	return o
end

local function stateOf(node)
	local mods = {}
	for i, m in ipairs(node.mods or {}) do
		mods[i] = { extra = m.extra, count = m.list and #m.list or nil }
	end
	return {
		id = node.id,
		type = node.type,
		dn = node.dn,
		ascendancyName = node.ascendancyName,
		group = node.g,
		x = node.x,
		y = node.y,
		linkedId = node.linkedId,
		passivePointsGranted = node.passivePointsGranted,
		modKey = node.modKey,
		mods = mods,
		modList = modArray(node.modList),
		keystoneMod = node.keystoneMod,
		unknown = node.unknown,
		extra = node.extra,
		sd = node.sd,
		isProxy = node.isProxy or nil,
		isBlighted = node.isBlighted or nil,
	}
end

local classes = {}
for classId, class in pairs(tree.classes) do
	local ascends = {}
	for ascendClassId, ascendClass in pairs(class.classes) do
		ascends[tostring(ascendClassId)] = {
			id = ascendClass.id,
			name = ascendClass.name,
			startNodeId = ascendClass.startNodeId,
		}
	end
	classes[tostring(classId)] = {
		name = class.name,
		base_str = class.base_str,
		base_dex = class.base_dex,
		base_int = class.base_int,
		startNodeId = class.startNodeId,
		classes = ascends,
	}
end

local nodeCount = 0
for _ in pairs(tree.nodes) do
	nodeCount = nodeCount + 1
end
emit("meta", { version = version, nodeCount = nodeCount, classes = classes })

local nodeIds = {}
for id in pairs(tree.nodes) do
	nodeIds[#nodeIds + 1] = id
end
table.sort(nodeIds)
for _, id in ipairs(nodeIds) do
	emit("node." .. id, stateOf(tree.nodes[id]))
end

-- legion.nodes is an array; key the records by the node's own string id.
local legionNodes = {}
for _, node in pairs(tree.legion.nodes) do
	legionNodes[#legionNodes + 1] = node
end
table.sort(legionNodes, function(a, b) return a.id < b.id end)
for _, node in ipairs(legionNodes) do
	emit("legion." .. node.id, {
		id = node.id,
		type = node.type,
		dn = node.dn,
		modKey = node.modKey,
		modList = modArray(node.modList),
		unknown = node.unknown,
		extra = node.extra,
		sd = node.sd,
	})
end

local effects = {}
for id, effect in pairs(tree.masteryEffects) do
	effects[tostring(id)] = { sd = effect.sd, modKey = effect.modKey, modList = modArray(effect.modList) }
end
emit("masteryEffects", effects)

local function nameToId(map)
	local o = {}
	for name, node in pairs(map) do
		o[name] = node.id
	end
	return o
end
emit("keystoneMap", nameToId(tree.keystoneMap))
emit("notableMap", nameToId(tree.notableMap))
emit("ascendancyMap", nameToId(tree.ascendancyMap))
emit("clusterNodeMap", nameToId(tree.clusterNodeMap))

local sockets = {}
for id, socket in pairs(tree.sockets) do
	local radii
	if socket.nodesInRadius then
		radii = {}
		for radiusIndex, nodes in ipairs(socket.nodesInRadius) do
			local ids = {}
			for nodeId in pairs(nodes) do
				ids[tostring(nodeId)] = true
			end
			radii[radiusIndex] = ids
		end
	end
	sockets[tostring(id)] = { charmSocket = socket.charmSocket or nil, nodesInRadius = radii }
end
emit("sockets", sockets)

out:close()
print(string.format("tree archive %s: %d nodes, %d legion", version, nodeCount, #legionNodes))
