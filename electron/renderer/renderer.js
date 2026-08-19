// Renders the engine state: build list, summary, main-skill selector, and the
// sidebar stat list (statBoxList) with PoB ^-colour codes translated to spans.

// SimpleGraphic's ^0-^9 palette
const PALETTE = [
	"#000000", "#ff0000", "#00ff00", "#0000ff", "#ffff00",
	"#ff00ff", "#00ffff", "#ffffff", "#b3b3b3", "#666666",
];

const buildListEl = document.getElementById("buildList");
const engineStatusEl = document.getElementById("engineStatus");
const summaryEl = document.getElementById("summary");
const skillSelectEl = document.getElementById("skillSelect");
const statBoxEl = document.getElementById("statBox");
const warningsEl = document.getElementById("warnings");
const engineLogEl = document.getElementById("engineLog");

let activeBuild = null;
let firstRender = true;

const treeView = new TreeView(
	document.getElementById("treeCanvas"),
	document.getElementById("treeTooltip"),
	document.getElementById("masteryMenu"),
	{
		onState: (state) => renderState(state),
		onError: (err) => setStatus(String(err), "statusError"),
		onTreeLoaded: (nodes) => rebuildSearchCorpus(nodes),
	}
);

function setStatus(text, cls) {
	engineStatusEl.textContent = "engine: " + text;
	engineStatusEl.className = cls;
}

// Converts a PoB-coloured string ("^7Label", "^xFF7700text") to DOM spans
function colouredSpans(text, defaultColour) {
	const fragment = document.createDocumentFragment();
	let colour = defaultColour || "#d0d0d0";
	let index = 0;
	const pattern = /\^(?:x([0-9a-fA-F]{6})|(\d))/g;
	let match;
	while ((match = pattern.exec(text)) !== null) {
		if (match.index > index) {
			const span = document.createElement("span");
			span.style.color = colour;
			span.textContent = text.slice(index, match.index);
			fragment.appendChild(span);
		}
		colour = match[1] ? "#" + match[1] : PALETTE[Number(match[2])];
		index = pattern.lastIndex;
	}
	if (index < text.length) {
		const span = document.createElement("span");
		span.style.color = colour;
		span.textContent = text.slice(index);
		fragment.appendChild(span);
	}
	return fragment;
}

function renderSummary(summary) {
	summaryEl.replaceChildren();
	const name = document.createElement("div");
	name.className = "buildName";
	name.textContent = summary.buildName || "Unnamed build";
	const meta = document.createElement("div");
	meta.className = "buildMeta";
	const cls = summary.ascendClassName && summary.ascendClassName !== "None"
		? summary.ascendClassName
		: summary.className;
	meta.textContent = `Level ${summary.level} ${cls} — ${summary.pointsUsed} points, ${summary.ascPointsUsed} ascendancy`;
	summaryEl.append(name, meta);
}

function renderSkills(skills) {
	skillSelectEl.replaceChildren();
	if (!skills.groups.length) {
		const option = document.createElement("option");
		option.textContent = "<no skills>";
		skillSelectEl.appendChild(option);
		skillSelectEl.disabled = true;
		return;
	}
	skillSelectEl.disabled = false;
	for (const group of skills.groups) {
		const option = document.createElement("option");
		option.value = group.index;
		option.textContent = group.label.replace(/\^x[0-9a-fA-F]{6}|\^\d/g, "");
		skillSelectEl.appendChild(option);
	}
	skillSelectEl.value = skills.mainSocketGroup;
}

function renderStatBox(statBox) {
	statBoxEl.replaceChildren();
	for (const entry of statBox) {
		const cells = Array.isArray(entry.cells) ? entry.cells : [];
		if (!cells.length) {
			const spacer = document.createElement("div");
			spacer.className = "statSpacer";
			statBoxEl.appendChild(spacer);
			continue;
		}
		const row = document.createElement("div");
		if (cells.length === 1) {
			row.className = entry.align ? "statCenter" : "statRow";
			row.appendChild(colouredSpans(cells[0]));
		} else {
			row.className = "statRow";
			const label = document.createElement("span");
			label.className = "statLabel";
			label.appendChild(colouredSpans(cells[0]));
			const value = document.createElement("span");
			value.className = "statValue";
			value.appendChild(colouredSpans(cells[1]));
			row.append(label, value);
		}
		statBoxEl.appendChild(row);
	}
}

function renderWarnings(warnings) {
	warningsEl.replaceChildren();
	for (const warning of warnings) {
		const line = document.createElement("div");
		line.textContent = "⚠ " + warning;
		warningsEl.appendChild(line);
	}
}

// Tree search is entirely local: the corpus is rebuilt whenever tree geometry
// is (re)fetched, and matching runs on the next frame after any input, so no
// keystroke ever reaches the engine
const treeSearchBoxEl = document.getElementById("treeSearchBox");
const treeSearchEl = document.getElementById("treeSearch");
const searchCaseEl = document.getElementById("treeSearchCase");
const searchRegexEl = document.getElementById("treeSearchRegex");

let searchCorpus = [];
let searchFrame = null;

function loadSearchToggle(el, key) {
	if (localStorage.getItem(key) === "1") {
		el.setAttribute("aria-pressed", "true");
	}
	el.addEventListener("click", () => {
		const next = el.getAttribute("aria-pressed") !== "true";
		el.setAttribute("aria-pressed", String(next));
		localStorage.setItem(key, next ? "1" : "0");
		applySearch();
	});
}

loadSearchToggle(searchCaseEl, "search.matchCase");
loadSearchToggle(searchRegexEl, "search.regex");

function setSearchError(message) {
	treeSearchBoxEl.classList.toggle("searchInvalid", Boolean(message));
	treeSearchEl.title = message || "";
}

// Called explicitly rather than only from the input event, so a preset search
// (SMOKE_SEARCH assigns .value, which fires nothing) and corpus rebuilds match
function applySearch() {
	const compiled = PobSearch.compileQuery(treeSearchEl.value, {
		regex: searchRegexEl.getAttribute("aria-pressed") === "true",
		matchCase: searchCaseEl.getAttribute("aria-pressed") === "true",
	});
	setSearchError(compiled.error);
	treeView.setSearchMatches(compiled.test ? PobSearch.matchCorpus(searchCorpus, compiled) : []);
}

function scheduleSearch() {
	if (searchFrame !== null) return;
	searchFrame = requestAnimationFrame(() => {
		searchFrame = null;
		applySearch();
	});
}

treeSearchEl.addEventListener("input", scheduleSearch);

function rebuildSearchCorpus(nodes) {
	searchCorpus = PobSearch.buildCorpus(nodes.values());
	applySearch();
}

function renderState(state) {
	renderSummary(state.summary);
	renderSkills(state.skills);
	renderStatBox(state.statBox);
	renderWarnings(state.warnings);
	if (state.treeState) {
		// Node text only changes when geometry is refetched, and that path
		// rebuilds the corpus and re-matches on its own
		treeView.setState(state.treeState);
	}
	if (firstRender) {
		firstRender = false;
		window.pob.statsRendered();
	}
}

async function loadBuild(fileName) {
	setStatus(`loading ${fileName}…`, "statusBooting");
	for (const item of buildListEl.children) {
		item.classList.toggle("active", item.dataset.file === fileName);
	}
	try {
		const state = await window.pob.loadBuildFile(fileName);
		activeBuild = fileName;
		await treeView.loadTree();
		renderState(state);
		setStatus("ready", "statusReady");
	} catch (err) {
		setStatus(String(err), "statusError");
	}
}

window.addEventListener("keydown", async (event) => {
	if (!event.ctrlKey) return;
	if (event.target instanceof HTMLInputElement || event.target instanceof HTMLSelectElement) return;
	const method = event.key === "z" ? "undoTree" : event.key === "y" ? "redoTree" : null;
	if (!method) return;
	event.preventDefault();
	try {
		renderState(await window.pob.call(method));
	} catch (err) {
		setStatus(String(err), "statusError");
	}
});

skillSelectEl.addEventListener("change", async () => {
	try {
		const state = await window.pob.call("selectMainSkill", { index: Number(skillSelectEl.value) });
		renderState(state);
	} catch (err) {
		setStatus(String(err), "statusError");
	}
});

document.getElementById("newBuildButton").addEventListener("click", async () => {
	setStatus("starting new build…", "statusBooting");
	try {
		const state = await window.pob.newBuild();
		activeBuild = null;
		for (const item of buildListEl.children) {
			item.classList.remove("active");
		}
		await treeView.loadTree();
		renderState(state);
		setStatus("ready", "statusReady");
	} catch (err) {
		setStatus(String(err), "statusError");
	}
});

window.pob.onEngineLog((line) => {
	engineLogEl.textContent += line + "\n";
	engineLogEl.scrollTop = engineLogEl.scrollHeight;
});

(async () => {
	const presetSearch = new URLSearchParams(location.search).get("search");
	if (presetSearch) {
		treeSearchEl.value = presetSearch;
	}
	const builds = await window.pob.listBuilds();
	for (const fileName of builds) {
		const item = document.createElement("li");
		item.textContent = fileName.replace(/\.xml$/i, "");
		item.dataset.file = fileName;
		item.addEventListener("click", () => loadBuild(fileName));
		buildListEl.appendChild(item);
	}
	if (builds.length) {
		loadBuild(builds[0]);
	} else {
		document.getElementById("newBuildButton").click();
	}
})();
