// Canvas passive-tree view. Geometry comes from the engine's getTree RPC
// (spec.nodes including cluster-jewel subgraphs, plus connector eligibility
// already filtered engine-side); allocation state arrives with every
// fullState payload. Clicking mirrors the old tree view: alloc along the
// path, dealloc with dependents, masteries via an effect menu.

const NODE_RADII = {
	Normal: 45,
	Notable: 72,
	Keystone: 95,
	Socket: 68,
	Mastery: 55,
	ClassStart: 130,
	AscendClassStart: 88,
};

const COLOURS = {
	edge: "#33333d",
	edgeAlloc: "#a5813f",
	edgePath: "#4f8f97",
	nodeFill: "#26262e",
	nodeStroke: "#4c4c58",
	nodeAllocFill: "#c9a35a",
	nodeAllocStroke: "#ecd7a4",
	nodePathStroke: "#6fc9d2",
	nodeDependsStroke: "#c05555",
	keystoneStroke: "#7f6fb0",
	socketStroke: "#5f8f5f",
	startStroke: "#8a6a2f",
	startActiveStroke: "#e7c15c",
};

class TreeView {
	constructor(canvas, tooltipEl, masteryMenuEl, callbacks) {
		this.canvas = canvas;
		this.ctx = canvas.getContext("2d");
		this.tooltipEl = tooltipEl;
		this.masteryMenuEl = masteryMenuEl;
		this.callbacks = callbacks; // { onState(state), onError(err) }

		this.nodes = new Map();
		this.edges = [];
		this.classes = [];
		this.nodeCount = 0;
		this.alloc = new Set();
		this.state = null;

		this.scale = 0.04;
		this.tx = 0;
		this.ty = 0;

		this.hoverNode = null;
		this.hoverPath = new Set();
		this.hoverDepends = new Set();
		this.hoverPathLength = 0;
		this.pathRequestSeq = 0;
		this.busy = false;
		this.searchMatches = new Set();

		this.pointer = null;
		this.panning = false;

		new ResizeObserver(() => this.resize()).observe(canvas.parentElement);
		this.bindEvents();
		this.resize();
	}

	async loadTree() {
		const tree = await window.pob.call("getTree");
		this.nodes.clear();
		for (const node of tree.nodes) {
			this.nodes.set(node.id, node);
		}
		this.edges = tree.edges.filter((edge) => this.nodes.has(edge.a) && this.nodes.has(edge.b));
		this.classes = tree.classes;
		this.nodeCount = tree.nodeCount;
		this.clearHover();
		this.fitView();
		this.render();
	}

	setSearchMatches(nodeIds) {
		this.searchMatches = new Set(nodeIds);
		this.render();
	}

	setState(treeState) {
		this.state = treeState;
		this.alloc = new Set(treeState.alloc);
		if (treeState.nodeCount !== this.nodeCount) {
			// Cluster jewel subgraphs changed; refresh geometry
			this.loadTree().catch((err) => this.callbacks.onError(err));
			return;
		}
		this.render();
	}

	resize() {
		const parent = this.canvas.parentElement;
		const dpr = window.devicePixelRatio || 1;
		this.canvas.width = Math.max(1, Math.floor(parent.clientWidth * dpr));
		this.canvas.height = Math.max(1, Math.floor(parent.clientHeight * dpr));
		this.canvas.style.width = parent.clientWidth + "px";
		this.canvas.style.height = parent.clientHeight + "px";
		this.render();
	}

	fitView() {
		let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
		for (const node of this.nodes.values()) {
			if (node.ascend) continue;
			minX = Math.min(minX, node.x);
			minY = Math.min(minY, node.y);
			maxX = Math.max(maxX, node.x);
			maxY = Math.max(maxY, node.y);
		}
		if (minX === Infinity) return;
		const dpr = window.devicePixelRatio || 1;
		const width = this.canvas.width / dpr;
		const height = this.canvas.height / dpr;
		this.scale = Math.min(width / (maxX - minX + 1200), height / (maxY - minY + 1200));
		this.tx = width / 2 - ((minX + maxX) / 2) * this.scale;
		this.ty = height / 2 - ((minY + maxY) / 2) * this.scale;
	}

	toWorld(screenX, screenY) {
		return { x: (screenX - this.tx) / this.scale, y: (screenY - this.ty) / this.scale };
	}

	nodeAt(worldX, worldY) {
		let best = null;
		let bestDist = Infinity;
		const slack = Math.max(30, 12 / this.scale);
		for (const node of this.nodes.values()) {
			const radius = (NODE_RADII[node.type] || 45) + slack;
			const dx = node.x - worldX;
			const dy = node.y - worldY;
			const dist = dx * dx + dy * dy;
			if (dist < radius * radius && dist < bestDist) {
				best = node;
				bestDist = dist;
			}
		}
		return best;
	}

	// -- Rendering ----------------------------------------------------------

	render() {
		const ctx = this.ctx;
		const dpr = window.devicePixelRatio || 1;
		ctx.setTransform(1, 0, 0, 1, 0, 0);
		ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
		ctx.setTransform(dpr * this.scale, 0, 0, dpr * this.scale, dpr * this.tx, dpr * this.ty);

		const activeAscend = this.state && this.state.curAscendClassName;

		for (const pass of ["base", "alloc", "path"]) {
			for (const edge of this.edges) {
				const a = this.nodes.get(edge.a);
				const b = this.nodes.get(edge.b);
				const isAlloc = this.alloc.has(a.id) && this.alloc.has(b.id);
				const inPath = (this.hoverPath.has(a.id) || this.alloc.has(a.id)) && this.hoverPath.has(b.id)
					|| (this.hoverPath.has(b.id) || this.alloc.has(b.id)) && this.hoverPath.has(a.id);
				const passWanted = inPath ? "path" : isAlloc ? "alloc" : "base";
				if (passWanted !== pass) continue;

				ctx.globalAlpha = a.ascend && a.ascend !== activeAscend ? 0.25 : 1;
				ctx.strokeStyle = pass === "path" ? COLOURS.edgePath : pass === "alloc" ? COLOURS.edgeAlloc : COLOURS.edge;
				ctx.lineWidth = pass === "base" ? 14 : 22;
				ctx.beginPath();
				if (edge.arc) {
					const r = Math.hypot(a.x - a.gx, a.y - a.gy);
					let start = a.angle - Math.PI / 2;
					let end = b.angle - Math.PI / 2;
					let delta = (end - start) % (Math.PI * 2);
					if (delta < 0) delta += Math.PI * 2;
					ctx.arc(a.gx, a.gy, r, start, end, delta > Math.PI);
				} else {
					ctx.moveTo(a.x, a.y);
					ctx.lineTo(b.x, b.y);
				}
				ctx.stroke();
			}
		}

		const startNodeIds = new Map();
		for (const cls of this.classes) {
			startNodeIds.set(cls.startNodeId, cls);
		}

		for (const node of this.nodes.values()) {
			const radius = NODE_RADII[node.type] || 45;
			const isAlloc = this.alloc.has(node.id);
			const inPath = this.hoverPath.has(node.id) && !isAlloc;
			const inDepends = this.hoverDepends.has(node.id);
			ctx.globalAlpha = node.ascend && node.ascend !== activeAscend ? 0.25 : 1;

			ctx.beginPath();
			if (node.type === "Mastery") {
				ctx.moveTo(node.x, node.y - radius);
				ctx.lineTo(node.x + radius, node.y);
				ctx.lineTo(node.x, node.y + radius);
				ctx.lineTo(node.x - radius, node.y);
				ctx.closePath();
			} else {
				ctx.arc(node.x, node.y, radius, 0, Math.PI * 2);
			}

			if (node.type === "ClassStart" || node.type === "AscendClassStart") {
				const cls = startNodeIds.get(node.id);
				const isActive = cls ? this.state && cls.name === this.state.curClassName : isAlloc;
				ctx.strokeStyle = isActive ? COLOURS.startActiveStroke : COLOURS.startStroke;
				ctx.lineWidth = 16;
				ctx.stroke();
				continue;
			}

			ctx.fillStyle = isAlloc ? COLOURS.nodeAllocFill : COLOURS.nodeFill;
			ctx.fill();
			ctx.lineWidth = node === this.hoverNode ? 18 : 10;
			ctx.strokeStyle =
				inDepends ? COLOURS.nodeDependsStroke :
				inPath ? COLOURS.nodePathStroke :
				isAlloc ? COLOURS.nodeAllocStroke :
				node.type === "Keystone" ? COLOURS.keystoneStroke :
				node.type === "Socket" ? COLOURS.socketStroke :
				COLOURS.nodeStroke;
			ctx.stroke();

			if (this.searchMatches.has(node.id)) {
				ctx.beginPath();
				ctx.arc(node.x, node.y, radius + 26, 0, Math.PI * 2);
				ctx.strokeStyle = "#e04444";
				ctx.lineWidth = 14;
				ctx.stroke();
			}
		}
		ctx.globalAlpha = 1;
	}

	// -- Interaction --------------------------------------------------------

	bindEvents() {
		this.canvas.addEventListener("wheel", (event) => {
			event.preventDefault();
			const rect = this.canvas.getBoundingClientRect();
			const mx = event.clientX - rect.left;
			const my = event.clientY - rect.top;
			const factor = event.deltaY < 0 ? 1.2 : 1 / 1.2;
			const next = Math.min(0.7, Math.max(0.015, this.scale * factor));
			this.tx = mx - ((mx - this.tx) / this.scale) * next;
			this.ty = my - ((my - this.ty) / this.scale) * next;
			this.scale = next;
			this.render();
		}, { passive: false });

		this.canvas.addEventListener("mousedown", (event) => {
			this.pointer = { x: event.clientX, y: event.clientY, moved: false };
		});

		window.addEventListener("mousemove", (event) => {
			if (this.pointer && (this.pointer.moved
				|| Math.abs(event.clientX - this.pointer.x) > 4
				|| Math.abs(event.clientY - this.pointer.y) > 4)) {
				this.pointer.moved = true;
				this.tx += event.movementX;
				this.ty += event.movementY;
				this.render();
				return;
			}
			if (event.target === this.canvas) {
				const rect = this.canvas.getBoundingClientRect();
				const world = this.toWorld(event.clientX - rect.left, event.clientY - rect.top);
				this.setHover(this.nodeAt(world.x, world.y), event.clientX, event.clientY);
			}
		});

		window.addEventListener("mouseup", (event) => {
			const pointer = this.pointer;
			this.pointer = null;
			if (!pointer || pointer.moved || event.target !== this.canvas) return;
			if (this.hoverNode) {
				this.clickNode(this.hoverNode, event.clientX, event.clientY);
			} else {
				this.hideMasteryMenu();
			}
		});

		this.canvas.addEventListener("mouseleave", () => this.setHover(null));
	}

	setHover(node, clientX, clientY) {
		if (node !== this.hoverNode) {
			this.hoverNode = node;
			this.hoverPath.clear();
			this.hoverDepends.clear();
			this.hoverPathLength = 0;
			if (node && node.type !== "ClassStart") {
				const seq = ++this.pathRequestSeq;
				window.pob.call("getNodePath", { id: node.id }).then((info) => {
					if (seq !== this.pathRequestSeq) return;
					if (info.alloc) {
						this.hoverDepends = new Set(info.depends);
					} else {
						this.hoverPath = new Set(info.path);
						this.hoverPathLength = info.path.length;
					}
					this.updateTooltip(clientX, clientY);
					this.render();
				}).catch(() => {});
			}
			this.render();
		}
		this.updateTooltip(clientX, clientY);
	}

	clearHover() {
		this.hoverNode = null;
		this.hoverPath.clear();
		this.hoverDepends.clear();
		this.tooltipEl.style.display = "none";
	}

	updateTooltip(clientX, clientY) {
		const node = this.hoverNode;
		if (!node || clientX === undefined) {
			this.tooltipEl.style.display = "none";
			return;
		}
		this.tooltipEl.replaceChildren();
		const title = document.createElement("div");
		title.className = "ttTitle";
		title.textContent = node.name || node.type;
		this.tooltipEl.appendChild(title);
		if (node.ascend) {
			const meta = document.createElement("div");
			meta.className = "ttMeta";
			meta.textContent = node.ascend;
			this.tooltipEl.appendChild(meta);
		}
		for (const line of node.sd || []) {
			const stat = document.createElement("div");
			stat.className = "ttStat";
			stat.textContent = line;
			this.tooltipEl.appendChild(stat);
		}
		const isAlloc = this.alloc.has(node.id);
		if (!isAlloc && this.hoverPathLength > 0) {
			const cost = document.createElement("div");
			cost.className = "ttMeta";
			cost.textContent = `Allocate: ${this.hoverPathLength} point${this.hoverPathLength === 1 ? "" : "s"}`;
			this.tooltipEl.appendChild(cost);
		} else if (isAlloc && this.hoverDepends.size > 0) {
			const cost = document.createElement("div");
			cost.className = "ttMeta";
			cost.textContent = `Deallocate: removes ${this.hoverDepends.size} node${this.hoverDepends.size === 1 ? "" : "s"}`;
			this.tooltipEl.appendChild(cost);
		}
		this.tooltipEl.style.display = "block";
		const pad = 16;
		const maxX = window.innerWidth - this.tooltipEl.offsetWidth - 8;
		const maxY = window.innerHeight - this.tooltipEl.offsetHeight - 8;
		this.tooltipEl.style.left = Math.min(clientX + pad, maxX) + "px";
		this.tooltipEl.style.top = Math.min(clientY + pad, maxY) + "px";
	}

	async clickNode(node, clientX, clientY) {
		if (this.busy) return;
		this.hideMasteryMenu();
		if (node.type === "ClassStart" || node.type === "AscendClassStart") return;
		try {
			if (node.type === "Mastery" && !this.alloc.has(node.id)) {
				const info = await window.pob.call("getMasteryEffects", { id: node.id });
				this.showMasteryMenu(node, info.effects, clientX, clientY);
				return;
			}
			this.busy = true;
			const state = await window.pob.call("toggleNode", { id: node.id });
			this.callbacks.onState(state);
			this.clearHover();
		} catch (err) {
			this.callbacks.onError(err);
		} finally {
			this.busy = false;
		}
	}

	showMasteryMenu(node, effects, clientX, clientY) {
		this.masteryMenuEl.replaceChildren();
		if (!effects.length) return;
		for (const effect of effects) {
			const item = document.createElement("div");
			item.className = "masteryItem";
			item.textContent = effect.label;
			item.addEventListener("click", async () => {
				this.hideMasteryMenu();
				try {
					this.busy = true;
					const state = await window.pob.call("selectMasteryEffect", { id: node.id, effectId: effect.id });
					this.callbacks.onState(state);
				} catch (err) {
					this.callbacks.onError(err);
				} finally {
					this.busy = false;
				}
			});
			this.masteryMenuEl.appendChild(item);
		}
		this.masteryMenuEl.style.display = "block";
		this.masteryMenuEl.style.left = Math.min(clientX, window.innerWidth - 420) + "px";
		this.masteryMenuEl.style.top = Math.min(clientY, window.innerHeight - this.masteryMenuEl.offsetHeight - 8) + "px";
	}

	hideMasteryMenu() {
		this.masteryMenuEl.style.display = "none";
		this.masteryMenuEl.replaceChildren();
	}
}

window.TreeView = TreeView;
