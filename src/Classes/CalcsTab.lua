-- Path of Building
--
-- Module: Calcs Tab
-- Calculations controller for the current build.
--
local pairs = pairs
local ipairs = ipairs
local t_insert = table.insert
local m_max = math.max
local m_floor = math.floor

local CalcsTabClass = newClass("CalcsTab", "UndoHandler", function(self, build)
	self.UndoHandler()

	self.build = build

	self.calcs = LoadModule("Modules/Calcs")

	self.input = { }
	self.input.skill_number = 1
	self.input.misc_buffMode = "EFFECTIVE"

	-- Collapsed-state of the calcs view sections, preserved for the build file
	self.sectionStateList = { }

	self.powerBuilderInitialized = nil
end)

function CalcsTabClass:Load(xml, dbFileName)
	for _, node in ipairs(xml) do
		if type(node) == "table" then
			if node.elem == "Input" then
				if not node.attrib.name then
					launch:ShowErrMsg("^1Error parsing '%s': 'Input' element missing name attribute", fileName)
					return true
				end
				if node.attrib.number then
					self.input[node.attrib.name] = tonumber(node.attrib.number)
				elseif node.attrib.string then
					self.input[node.attrib.name] = node.attrib.string
				elseif node.attrib.boolean then
					self.input[node.attrib.name] = node.attrib.boolean == "true"
				else
					launch:ShowErrMsg("^1Error parsing '%s': 'Input' element missing number, string or boolean attribute", fileName)
					return true
				end
			elseif node.elem == "Section" then
				if not node.attrib.id then
					launch:ShowErrMsg("^1Error parsing '%s': 'Section' element missing id attribute", fileName)
					return true
				end
				if node.attrib.subsection then
					t_insert(self.sectionStateList, {
						id = node.attrib.id,
						subsection = node.attrib.subsection,
						collapsed = node.attrib.collapsed,
					})
				end
			end
		end
	end
	self:ResetUndo()
end

function CalcsTabClass:Save(xml)
	for k, v in pairs(self.input) do
		local child = { elem = "Input", attrib = {name = k} }
		if type(v) == "number" then
			child.attrib.number = tostring(v)
		elseif type(v) == "boolean" then
			child.attrib.boolean = tostring(v)
		else
			child.attrib.string = tostring(v)
		end
		t_insert(xml, child)
	end
	for _, sectionState in ipairs(self.sectionStateList) do
		t_insert(xml, { elem = "Section", attrib = {
			id = sectionState.id,
			subsection = sectionState.subsection,
			collapsed = sectionState.collapsed,
		} })
	end
end

function CalcsTabClass:CheckFlag(obj, actor)
	actor = actor or (self.input.showMinion and self.calcsEnv.minion or self.calcsEnv.player)
	local skillFlags = actor.mainSkill.skillFlags
	if obj.flag and not skillFlags[obj.flag] then
		return
	end
	if obj.flagList then
		for _, flag in ipairs(obj.flagList) do
			if not skillFlags[flag] then
				return
			end
		end
	end
	if obj.playerFlag and not self.calcsEnv.player.mainSkill.skillFlags[obj.playerFlag] then
		return
	end
	if obj.notFlag and skillFlags[obj.notFlag] then
		return
	end
	if obj.notFlagList then
		for _, flag in ipairs(obj.notFlagList) do
			if skillFlags[flag] then
				return
			end
		end
	end
	if obj.haveOutput then
		local ns, var = obj.haveOutput:match("^(%a+)%.(%a+)$")
		if ns then
			if not actor.output[ns] or not actor.output[ns][var] or actor.output[ns][var] == 0 then
				return
			end
		elseif not actor.output[obj.haveOutput] or actor.output[obj.haveOutput] == 0 then
			return
		end
	end
	return true
end

-- Build the calculation output tables
function CalcsTabClass:BuildOutput()
	self.powerBuildFlag = true

	for _, node in pairs(self.build.spec.nodes) do
		-- Set default final mod list for all nodes; some may not be set during the main pass
		node.finalModList = node.modList
	end

	self.mainEnv = self.calcs.buildOutput(self.build, "MAIN")
	self.mainOutput = self.mainEnv.player.output
	self.calcsEnv = self.calcs.buildOutput(self.build, "CALCS")
	self.calcsOutput = self.calcsEnv.player.output

	-- Retrieve calculator functions
	self.nodeCalculator = { self.calcs.getNodeCalculator(self.build) }
	self.miscCalculator = { self.calcs.getMiscCalculator(self.build) }
end

-- Controls the coroutine that calculates node power
function CalcsTabClass:BuildPower()
	if self.powerBuildFlag then
		self.powerBuildFlag = false
		self.powerMax = nil
		self.powerBuilder = coroutine.create(self.PowerBuilder)
	end
	if self.powerBuilder then
		local res, errMsg = coroutine.resume(self.powerBuilder, self)
		if launch.devMode and not res then
			error(errMsg)
		end
		if coroutine.status(self.powerBuilder) == "dead" then
			self.powerBuilder = nil
			if self.build.powerBuilderCallback then
				self.build.powerBuilderCallback()
			end
		end
	end
end

-- Estimate the offensive and defensive power of all unallocated nodes
function CalcsTabClass:PowerBuilder()
	-- local timer_start = GetTime()
	local useFullDPS = self.powerStat and self.powerStat.stat == "FullDPS"
	local calcFunc, calcBase = self:GetMiscCalculator()
	local anointOnly = self.nodePowerAnointOnly
	-- Same test the anoint node list uses: a node is anointable if it has an oil recipe
	local function isNodeInScope(node)
		return not anointOnly or (node.recipe and #node.recipe >= 1)
	end
	local cache = { }
	local distanceMap = { }
	local distanceList = { }
	local masteryNodeList = { }
	local newPowerMax = {
		singleStat = 0,
		offence = 0,
		offencePerPoint = 0,
		defence = 0,
		defencePerPoint = 0
	}
	if not self.powerMax then
		self.powerMax = newPowerMax
	end
	if coroutine.running() then
		coroutine.yield()
	end

	local function buildMasteryEffectNode(node, effect)
		local effectNode = {
			id = node.id,
			type = node.type,
			name = node.name,
			sd = { },
		}
		for i, sd in ipairs(effect.sd or { }) do
			effectNode.sd[i] = sd
		end
		self.build.spec.tree:ProcessStats(effectNode)
		return effectNode
	end

	local function masteryEffectCanBeAssignedToNode(node, masteryEffect)
		local assignedNodeId = isValueInTable(self.build.spec.masterySelections, masteryEffect.effect)
		return not assignedNodeId or assignedNodeId == node.id
	end

	local function calculateAddNodePower(power, distance, node, output, buildPathNodes)
		if self.powerStat and self.powerStat.stat and not self.powerStat.ignoreForNodes then
			power.singleStat = self:CalculatePowerStat(self.powerStat, output, calcBase)
			if node.path and not node.ascendancyName then
				newPowerMax.singleStat = m_max(newPowerMax.singleStat, power.singleStat)
				power.pathPower = power.singleStat
				if distance > 1 then
					power.pathPower = self:CalculatePowerStat(self.powerStat, calcFunc({ addNodes = buildPathNodes() }, useFullDPS), calcBase)
				end
			end
		elseif not self.powerStat or not self.powerStat.ignoreForNodes then
			power.offence, power.defence = self:CalculateCombinedOffDefStat(output, calcBase)
			power.singleStat = power.offence
			if node.path and not node.ascendancyName then
				newPowerMax.offence = m_max(newPowerMax.offence, power.offence)
				newPowerMax.defence = m_max(newPowerMax.defence, power.defence)
				newPowerMax.offencePerPoint = m_max(newPowerMax.offencePerPoint, power.offence / distance)
				newPowerMax.defencePerPoint = m_max(newPowerMax.defencePerPoint, power.defence / distance)
			end
		end
	end

	local start = GetTime()
	local nodeIndex = 0
	local total = 0

	for nodeId, node in pairs(self.build.spec.nodes) do
		wipeTable(node.power)
		if node.type == "Mastery" then
			node.power.masteryEffects = { }
		end
		if node.modKey ~= "" and not self.mainEnv.grantedPassives[nodeId] and isNodeInScope(node) then
			if node.type == "Mastery" and node.allMasteryOptions then
				if not (self.nodePowerMaxDepth and self.nodePowerMaxDepth < node.pathDist) then
					t_insert(masteryNodeList, node)
					for _, masteryEffect in ipairs(node.masteryEffects or { }) do
						if masteryEffectCanBeAssignedToNode(node, masteryEffect) then
							total = total + 1
						end
					end
				end
			else
				local dist = node.pathDist or 1000
				for _, leap in ipairs(node.intuitiveLeapLikesAffecting or {}) do
					if leap.alloc then
						dist = math.max(math.min(leap.pathDist or 1000, dist), 1)
					end
				end
				distanceMap[dist] = distanceMap[dist] or {}
				distanceMap[dist][nodeId] = node
				node.power.distance = dist
				if (not self.nodePowerMaxDepth) or dist <= self.nodePowerMaxDepth then
					total = total + 1
				end
			end
		end
	end
	for distance, nodes in pairs(distanceMap) do
		t_insert(distanceList, { distance, nodes })
	end
	distanceMap = nil
	table.sort(distanceList, function(a, b) return a[1] < b[1] end)
	-- Count eligible cluster nodes
	for _, node in pairs(self.build.spec.tree.clusterNodeMap) do
		if not node.alloc and node.modKey ~= "" and not self.mainEnv.grantedPassives[node.id] and isNodeInScope(node) then
			total = total + 1
		end
	end

	for _, data in ipairs(distanceList) do
		local distance, nodes = data[1], data[2]
		if self.nodePowerMaxDepth and self.nodePowerMaxDepth < distance then
			break
		end
		for nodeId, node in pairs(nodes) do
			if not node.alloc and node.modKey ~= "" and not self.mainEnv.grantedPassives[nodeId] then
				if not cache[node.modKey] then
					cache[node.modKey] = calcFunc({ addNodes = { [node] = true } }, useFullDPS)
				end
				local output = cache[node.modKey]
				calculateAddNodePower(node.power, distance, node, output, function()
					local pathNodes = { }
					for _, pathNode in pairs(node.path) do
						pathNodes[pathNode] = true
					end
					return pathNodes
				end)
			elseif node.alloc and node.modKey ~= "" and not self.mainEnv.grantedPassives[nodeId] then
				if not cache[node.modKey.."_remove"] then
					cache[node.modKey.."_remove"] = calcFunc({ removeNodes = { [node] = true } }, useFullDPS)
				end
				local output = cache[node.modKey.."_remove"]
				if self.powerStat and self.powerStat.stat and not self.powerStat.ignoreForNodes then
					node.power.singleStat = self:CalculatePowerStat(self.powerStat, output, calcBase)
					if node.depends and not node.ascendancyName then
						node.power.pathPower = node.power.singleStat
						local pathNodes = { }
						for _, node in pairs(node.depends) do
							pathNodes[node] = true
						end
						if #node.depends > 1 then
							node.power.pathPower = self:CalculatePowerStat(self.powerStat, calcFunc({ removeNodes = pathNodes }, useFullDPS), calcBase)
						end
					end
				end
			end
			if node.type == "Mastery" then
				local selectedEffectId = self.build.spec.masterySelections[node.id]
				if selectedEffectId then
					node.power.masteryEffects[selectedEffectId] = {
						singleStat = node.power.singleStat,
						pathPower = node.power.pathPower,
						offence = node.power.offence,
						defence = node.power.defence,
					}
				end
			end
			nodeIndex = nodeIndex + 1
			if coroutine.running() and GetTime() - start > 100 then
				if self.build.powerBuilderProgressCallback then
					self.build.powerBuilderProgressCallback(m_floor(nodeIndex/total*100))
				end
				coroutine.yield()
				start = GetTime()
			end
		end
	end

	for _, node in ipairs(masteryNodeList) do
		for _, masteryEffect in ipairs(node.masteryEffects or { }) do
			if masteryEffectCanBeAssignedToNode(node, masteryEffect) then
				local effect = self.build.spec.tree.masteryEffects[masteryEffect.effect]
				if effect then
					local effectNode = buildMasteryEffectNode(node, effect)
					if effectNode.modKey ~= "" then
						if not cache[effectNode.modKey] then
							cache[effectNode.modKey] = calcFunc({ addNodes = { [effectNode] = true } }, useFullDPS)
						end
						local output = cache[effectNode.modKey]
						node.power.masteryEffects[effect.id] = { }
						local effectPower = node.power.masteryEffects[effect.id]
						calculateAddNodePower(effectPower, node.pathDist, node, output, function()
							local pathNodes = {
								[effectNode] = true
							}
							for _, pathNode in pairs(node.path) do
								if pathNode ~= node then
									pathNodes[pathNode] = true
								end
							end
							return pathNodes
						end)
						if self.powerStat and self.powerStat.stat and not self.powerStat.ignoreForNodes then
							effectPower.pathPower = effectPower.pathPower or effectPower.singleStat
							node.power.singleStat = m_max(node.power.singleStat or 0, effectPower.singleStat)
							node.power.pathPower = m_max(node.power.pathPower or 0, effectPower.pathPower)
						elseif not self.powerStat or not self.powerStat.ignoreForNodes then
							node.power.offence = m_max(node.power.offence or 0, effectPower.offence)
							node.power.defence = m_max(node.power.defence or 0, effectPower.defence)
							node.power.singleStat = m_max(node.power.singleStat or 0, effectPower.singleStat)
						end
					end
					nodeIndex = nodeIndex + 1
					if coroutine.running() and GetTime() - start > 100 then
						if self.build.powerBuilderProgressCallback then
							self.build.powerBuilderProgressCallback(m_floor(nodeIndex/total*100))
						end
						coroutine.yield()
						start = GetTime()
					end
				end
			end
		end
	end

	-- Calculate the impact of every cluster notable
	-- used for the power report screen
	for nodeName, node in pairs(self.build.spec.tree.clusterNodeMap) do
		if not node.power then
			node.power = {}
		end
		wipeTable(node.power)
		if not node.alloc and node.modKey ~= "" and not self.mainEnv.grantedPassives[node.id] and isNodeInScope(node) then
			if not cache[node.modKey] then
				cache[node.modKey] = calcFunc({ addNodes = { [node] = true } }, useFullDPS)
			end
			local output = cache[node.modKey]
			if self.powerStat and self.powerStat.stat and not self.powerStat.ignoreForNodes then
				node.power.singleStat = self:CalculatePowerStat(self.powerStat, output, calcBase)
			end
			nodeIndex = nodeIndex + 1
			if coroutine.running() and GetTime() - start > 100 then
				if self.build.powerBuilderProgressCallback then
					self.build.powerBuilderProgressCallback(m_floor(nodeIndex/total*100))
				end
				coroutine.yield()
				start = GetTime()
			end
		end
	end
	self.powerMax = newPowerMax
	self.powerBuilderInitialized = true
	-- ConPrintf("Power Build time: %d ms", GetTime() - timer_start)
end

function CalcsTabClass:CalculatePowerStat(selection, original, modified)
	local originalValue = data.powerStatList.GetFromOutput(original, selection)
	local modifiedValue = data.powerStatList.GetFromOutput(modified, selection)
	return originalValue - modifiedValue
end

function CalcsTabClass:CalculateCombinedOffDefStat(original, modified)
	local defence = (original.LifeUnreserved - modified.LifeUnreserved) / m_max(3000, modified.Life) +
					(original.Armour - modified.Armour) / m_max(10000, modified.Armour) +
					((original.EnergyShieldRecoveryCap or original.EnergyShield) - (modified.EnergyShieldRecoveryCap or modified.EnergyShield)) / m_max(3000, (modified.EnergyShieldRecoveryCap or modified.EnergyShield)) +
					(original.Evasion - modified.Evasion) / m_max(10000, modified.Evasion) +
					(original.LifeRegenRecovery - modified.LifeRegenRecovery) / 500 +
					(original.EnergyShieldRegenRecovery - modified.EnergyShieldRegenRecovery) / 1000
	local modifiedDps = modified.CombinedDPS + (modified.Minion and modified.Minion.CombinedDPS or 0)
	local dpsIncr = original.CombinedDPS + (original.Minion and original.Minion.CombinedDPS or 0) - modifiedDps
	return dpsIncr / modifiedDps, defence
end

function CalcsTabClass:GetNodeCalculator()
	return unpack(self.nodeCalculator)
end

function CalcsTabClass:GetMiscCalculator()
	return unpack(self.miscCalculator)
end

function CalcsTabClass:CreateUndoState()
	return copyTable(self.input)
end

function CalcsTabClass:RestoreUndoState(state)
	wipeTable(self.input)
	for k, v in pairs(state) do
		self.input[k] = v
	end
end
