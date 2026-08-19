// Fixture test for the renderer's tree search. Run with `npm run test:search`
// (electron doubles as the node runtime; there is no standalone node here).
// This asserts the semantics we chose, not parity with the deleted lua matcher —
// search is a pure view filter, so these cases change when the features do.

const assert = require("assert");
const { buildCorpus, compileQuery, matchCorpus } = require("../renderer/search.js");

const nodes = [
	{
		id: 1,
		name: "Avatar of Fire",
		type: "Keystone",
		sd: ["50% of Physical Damage Converted to Fire Damage", "Deal no Non-Fire Damage"],
	},
	{ id: 2, name: "Precise Technique", type: "Keystone", sd: ["+40% to Critical Strike Multiplier"] },
	{ id: 3, name: "Two Hand Mastery", type: "Mastery", sd: ["Gain 10 Damage"] },
	{ id: 4, name: "Unchosen Mastery", type: "Mastery", sd: [] },
	{ id: 5, name: "WITCH", type: "ClassStart", sd: [] },
	{ id: 6, name: "Elemental Focus", type: "Notable", sd: ["+#% to Fire Resistance"] },
];

const corpus = buildCorpus(nodes);

function find(query, options) {
	const compiled = compileQuery(query, options || {});
	return { ids: matchCorpus(corpus, compiled), error: compiled.error, empty: compiled.empty };
}

const cases = {
	"corpus excludes ClassStart and effectless masteries"() {
		assert.deepStrictEqual(corpus.map((entry) => entry.id), [1, 2, 3, 6]);
	},

	"blank query compiles to empty and matches nothing"() {
		const result = find("");
		assert.strictEqual(result.empty, true);
		assert.deepStrictEqual(result.ids, []);
	},

	"single word matches case-insensitively by default"() {
		assert.deepStrictEqual(find("fire").ids, [1, 6]);
	},

	"multi-word query matches contiguous text only, with no AND split"() {
		assert.deepStrictEqual(find("fire damage").ids, [1]);
		assert.deepStrictEqual(find("damage fire").ids, []);
	},

	"match case narrows to the exact casing"() {
		assert.deepStrictEqual(find("fire", { matchCase: true }).ids, []);
		assert.deepStrictEqual(find("Fire", { matchCase: true }).ids, [1, 6]);
	},

	"node type is searchable"() {
		assert.deepStrictEqual(find("keystone").ids, [1, 2]);
	},

	"regex mode supports character classes and alternation"() {
		assert.deepStrictEqual(find("\\d+ Damage", { regex: true }).ids, [3]);
		assert.deepStrictEqual(find("physical.*fire", { regex: true }).ids, [1]);
		assert.deepStrictEqual(find("technique|focus", { regex: true }).ids, [2, 6]);
	},

	"regex mode honours match case"() {
		assert.deepStrictEqual(find("avatar", { regex: true }).ids, [1]);
		assert.deepStrictEqual(find("avatar", { regex: true, matchCase: true }).ids, []);
	},

	"a dot does not cross a line break without the s flag"() {
		assert.deepStrictEqual(find("Fire Damage.*Deal no", { regex: true }).ids, []);
		assert.deepStrictEqual(find("Fire Damage[\\s\\S]*Deal no", { regex: true }).ids, [1]);
	},

	"metacharacters are literal with regex off"() {
		assert.deepStrictEqual(find("+#% to Fire").ids, [6]);
		assert.deepStrictEqual(find("+40%").ids, [2]);
	},

	"an uncompilable pattern reports an error and matches nothing"() {
		const result = find("+#% to Fire", { regex: true });
		assert.ok(result.error, "expected a compile error");
		assert.deepStrictEqual(result.ids, []);

		const unclosed = find("[unterminated", { regex: true });
		assert.ok(unclosed.error, "expected a compile error");
		assert.deepStrictEqual(unclosed.ids, []);
	},
};

let failed = 0;
for (const [name, run] of Object.entries(cases)) {
	try {
		run();
		console.log("ok   " + name);
	} catch (err) {
		failed++;
		console.error("FAIL " + name + "\n     " + err.message.replace(/\n/g, "\n     "));
	}
}

console.log(`\n${Object.keys(cases).length - failed}/${Object.keys(cases).length} passed`);
process.exit(failed ? 1 : 0);
