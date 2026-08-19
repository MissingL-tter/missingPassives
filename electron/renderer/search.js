// Client-side passive tree search. Runs entirely in the renderer over the
// cached getTree payload: no RPC per keystroke, and nothing here can touch
// build state. Semantics are ours, not reference PoB's — the search box owns
// a match-case and a regex toggle, and the toggles decide the mode outright
// rather than being inferred from the query.

// One corpus entry per searchable node. nodeText is the node's name, its stat
// lines, and its type newline-joined; nodeLower is the same prelowered, since
// String.includes has no case-insensitive form and lowering 1.5k strings per
// keystroke would be wasted work.
function buildCorpus(treeNodes) {
	const corpus = [];
	for (const node of treeNodes) {
		// Class start nodes are not allocatable and only ever match their class name
		if (node.type === "ClassStart") continue;
		const sd = node.sd || [];
		// A mastery holding effects always has stat lines (every option before a
		// pick, the chosen effect after), so empty sd means there is nothing to find
		if (node.type === "Mastery" && !sd.length) continue;
		const parts = [node.name || ""];
		for (const line of sd) {
			parts.push(line);
		}
		parts.push(node.type || "");
		const nodeText = parts.join("\n");
		corpus.push({ id: node.id, nodeText, nodeLower: nodeText.toLowerCase() });
	}
	return corpus;
}

// Returns { test(entry) } for a usable query, { error } for a regex that will
// not compile, or { empty } for a blank query. Never throws.
function compileQuery(query, options) {
	const opts = options || {};
	if (!query) {
		return { empty: true };
	}
	if (opts.regex) {
		let pattern;
		try {
			pattern = new RegExp(query, opts.matchCase ? "" : "i");
		} catch (err) {
			return { error: err.message };
		}
		return { test: (entry) => pattern.test(entry.nodeText) };
	}
	// Literal path: nothing is compiled, so metacharacters need no escaping and
	// a query like "+#% to Fire Damage" just works
	if (opts.matchCase) {
		return { test: (entry) => entry.nodeText.includes(query) };
	}
	const lowered = query.toLowerCase();
	return { test: (entry) => entry.nodeLower.includes(lowered) };
}

function matchCorpus(corpus, compiled) {
	const ids = [];
	if (!compiled || !compiled.test) return ids;
	for (let i = 0; i < corpus.length; i++) {
		if (compiled.test(corpus[i])) {
			ids.push(corpus[i].id);
		}
	}
	return ids;
}

const api = { buildCorpus, compileQuery, matchCorpus };

if (typeof window !== "undefined") {
	window.PobSearch = api;
}
if (typeof module !== "undefined" && module.exports) {
	module.exports = api;
}
