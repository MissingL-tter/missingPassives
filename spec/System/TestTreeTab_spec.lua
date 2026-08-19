describe("TreeTab", function()
	local originalClusterNodeMap
	local originalMasteryEffects

	before_each(function()
		newBuild()
		originalClusterNodeMap = build.spec.tree.clusterNodeMap
		originalMasteryEffects = build.spec.tree.masteryEffects
	end)

	after_each(function()
		build.spec.tree.clusterNodeMap = originalClusterNodeMap
		build.spec.tree.masteryEffects = originalMasteryEffects
	end)

	it("adds separate power report entries for mastery effects", function()
		local treeTab = build.treeTab
		local parentNode = { id = 2 }
		local masteryNode = {
			id = 1,
			type = "Mastery",
			dn = "Two Hand Mastery",
			power = {
				masteryEffects = {
					[101] = { singleStat = 10, pathPower = 10 },
					[102] = { singleStat = 20, pathPower = 20 },
				},
			},
			masteryEffects = {
				{ effect = 101 },
				{ effect = 102 },
			},
			path = { parentNode, false },
			x = 10,
			y = 20,
		}
		masteryNode.path[2] = masteryNode

		treeTab.build.displayStats = {
			{ stat = "Damage", label = "Damage", fmt = ".1f" },
		}
		treeTab.build.spec.nodes = {
			[masteryNode.id] = masteryNode,
		}
		treeTab.build.spec.masterySelections = { }
		treeTab.build.spec.tree.clusterNodeMap = { }
		treeTab.build.spec.tree.masteryEffects = {
			[101] = { id = 101, sd = { "Gain 10 Damage" }, stats = { "Gain 10 Damage" } },
			[102] = { id = 102, sd = { "Gain 20 Damage" }, stats = { "Gain 20 Damage" } },
		}
		treeTab.build.calcsTab.mainEnv = { grantedPassives = { } }

		local report = treeTab:BuildPowerReportList({ stat = "Damage", label = "Damage" })

		assert.are.same(2, #report)
		assert.are.same("Mastery", report[1].type)
		assert.are.same("Two Hand Mastery: Gain 20 Damage", report[1].name)
		assert.are.same(20, report[1].power)
		assert.are.same(2, report[1].pathDist)
		assert.are.same(10, report[2].power)
		assert.are.same("Two Hand Mastery: Gain 10 Damage", report[2].name)
	end)

	describe("GetMasteryEffectOptions", function()
		local masteryNode

		before_each(function()
			masteryNode = {
				id = 1,
				type = "Mastery",
				dn = "Two Hand Mastery",
				masteryEffects = {
					{ effect = 101, stats = { "Gain 10 Damage" } },
					{ effect = 102, stats = { "Gain 20 Damage" } },
				},
			}
			build.spec.nodes = { [masteryNode.id] = masteryNode }
			build.spec.masterySelections = { }
		end)

		it("offers every effect when none are assigned", function()
			local options, selected = build.treeTab:GetMasteryEffectOptions(masteryNode)

			assert.are.same(2, #options)
			assert.are.same(101, options[1].id)
			assert.are.same({ "Gain 10 Damage" }, options[1].stats)
			assert.are.same(102, options[2].id)
			assert.is_nil(selected)
		end)

		it("excludes an effect held by another mastery node", function()
			build.spec.masterySelections = { [7] = 101 }

			local options = build.treeTab:GetMasteryEffectOptions(masteryNode)

			assert.are.same(1, #options)
			assert.are.same(102, options[1].id)
		end)

		it("keeps this node's own selection and reports it", function()
			build.spec.masterySelections = { [masteryNode.id] = 101 }

			local options, selected = build.treeTab:GetMasteryEffectOptions(masteryNode)

			assert.are.same(2, #options)
			assert.are.same(101, options[1].id)
			assert.are.same(101, selected)
		end)
	end)
end)
