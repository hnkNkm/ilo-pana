<script lang="ts">
	import { ChevronDown, ChevronRight, Search, X } from '@lucide/svelte/icons';

	interface Entry {
		path: string;
		key: string;
		value: unknown;
		depth: number;
		isLeaf: boolean;
	}

	let { value }: { value: unknown } = $props();

	let collapsed = $state<Set<string>>(new Set());
	let searchInput = $state('');
	let matchIndex = $state(0);

	function flatten(v: unknown, path = '', depth = 0): Entry[] {
		const out: Entry[] = [];
		if (v === null || typeof v !== 'object') {
			out.push({ path, key: '', value: v, depth, isLeaf: true });
			return out;
		}
		if (Array.isArray(v)) {
			v.forEach((item, i) => {
				const p = path ? `${path}.${i}` : String(i);
				const leaf = item === null || typeof item !== 'object';
				out.push({ path: p, key: String(i), value: item, depth, isLeaf: leaf });
				if (!leaf) out.push(...flatten(item, p, depth + 1));
			});
		} else {
			for (const key of Object.keys(v as Record<string, unknown>)) {
				const item = (v as Record<string, unknown>)[key];
				const p = path ? `${path}.${key}` : key;
				const leaf = item === null || typeof item !== 'object';
				out.push({ path: p, key, value: item, depth, isLeaf: leaf });
				if (!leaf) out.push(...flatten(item, p, depth + 1));
			}
		}
		return out;
	}

	const entries = $derived(flatten(value));

	function stringifyValue(v: unknown): string {
		if (v === null) return 'null';
		if (typeof v === 'string') return JSON.stringify(v);
		if (typeof v === 'number' || typeof v === 'boolean') return String(v);
		if (Array.isArray(v)) return `[${v.length} items]`;
		return `{${Object.keys(v as object).length} keys}`;
	}

	const searching = $derived(searchInput.trim().length > 0);
	const matches = $derived.by(() => {
		const q = searchInput.trim().toLowerCase();
		if (!q) return [];
		return entries.filter(
			(e) =>
				e.key.toLowerCase().includes(q) ||
				stringifyValue(e.value).toLowerCase().includes(q)
		);
	});
	const matchPaths = $derived(new Set(matches.map((m) => m.path)));
	const openPaths = $derived.by(() => {
		if (!matches.length) return new Set<string>();
		const set = new Set<string>();
		for (const m of matches) {
			const parts = m.path.split('.');
			let p = '';
			for (let i = 0; i < parts.length - 1; i++) {
				p = p ? `${p}.${parts[i]}` : parts[i];
				set.add(p);
			}
		}
		return set;
	});
	const currentMatchPath = $derived(matches.length ? matches[matchIndex]?.path : '');

	$effect(() => {
		if (!matches.length) {
			matchIndex = 0;
			return;
		}
		matchIndex = Math.min(matchIndex, matches.length - 1);
		const el = document.getElementById(`json-tree-match-${matchIndex}`);
		el?.scrollIntoView({ block: 'nearest' });
	});

	// During a search only paths leading to matches stay open; otherwise the
	// user's collapse state applies.
	function parentOf(entry: Entry): string {
		const i = entry.path.lastIndexOf('.');
		return i >= 0 ? entry.path.slice(0, i) : '';
	}

	function isParentOpen(entry: Entry): boolean {
		if (entry.depth === 0) return true;
		const p = parentOf(entry);
		return searching ? openPaths.has(p) : !collapsed.has(p);
	}

	function isExpanded(entry: Entry): boolean {
		if (entry.isLeaf) return false;
		return searching ? openPaths.has(entry.path) : !collapsed.has(entry.path);
	}

	function toggle(entry: Entry) {
		if (entry.isLeaf) return;
		const next = new Set(collapsed);
		if (next.has(entry.path)) {
			next.delete(entry.path);
		} else {
			next.add(entry.path);
		}
		collapsed = next;
	}

	function goToMatch(delta: number) {
		if (!matches.length) return;
		matchIndex = (matchIndex + delta + matches.length) % matches.length;
	}

	function highlightClass(entry: Entry): string {
		if (!searching) return '';
		if (entry.path === currentMatchPath) return 'bg-yellow-400/30';
		if (matchPaths.has(entry.path)) return 'bg-yellow-400/10';
		return '';
	}
</script>

<div class="flex flex-col h-full">
	<div class="flex items-center gap-2 border-b border-slate-700 px-3 py-1.5">
		<Search class="h-3.5 w-3.5 text-slate-400" />
		<input
			type="text"
			placeholder="Search keys & values"
			value={searchInput}
			oninput={(e) => (searchInput = e.currentTarget.value)}
			class="flex-1 bg-transparent text-xs font-mono text-slate-200 placeholder:text-slate-500 outline-none"
		/>
		{#if searching}
			<span class="text-[10px] font-mono text-slate-400">
				{matches.length ? `${matchIndex + 1}/${matches.length}` : '0/0'}
			</span>
			<button onclick={() => goToMatch(-1)} class="text-slate-400 hover:text-white" title="Previous match">
				<ChevronDown class="h-3.5 w-3.5 rotate-180" />
			</button>
			<button onclick={() => goToMatch(1)} class="text-slate-400 hover:text-white" title="Next match">
				<ChevronDown class="h-3.5 w-3.5" />
			</button>
			<button onclick={() => (searchInput = '')} class="text-slate-400 hover:text-white" title="Clear search">
				<X class="h-3.5 w-3.5" />
			</button>
		{/if}
	</div>
	<div class="flex-1 overflow-auto p-2 font-mono text-[12px] leading-[1.5]">
		{#if entries.length === 0}
			<p class="p-3 text-xs text-slate-500">(empty)</p>
		{:else}
			{#each entries as entry (entry.path)}
				{#if isParentOpen(entry)}
					<button
						id={entry.path === currentMatchPath ? `json-tree-match-${matchIndex}` : undefined}
						onclick={() => toggle(entry)}
						class="flex w-full items-center gap-1 rounded px-1 text-left hover:bg-slate-800/60 {entry.isLeaf ? 'cursor-default' : 'cursor-pointer'} {highlightClass(entry)}"
						style="padding-left: {entry.depth * 16 + 4}px"
					>
						<span class="w-4 shrink-0 text-slate-500">
							{#if !entry.isLeaf}
								{#if isExpanded(entry)}
									<ChevronDown class="h-3.5 w-3.5" />
								{:else}
									<ChevronRight class="h-3.5 w-3.5" />
								{/if}
							{/if}
						</span>
						{#if entry.key !== ''}
							<span class="text-sky-300">{entry.key}</span>
							<span class="text-slate-500">:</span>
						{/if}
						<span
							class={entry.isLeaf
								? typeof entry.value === 'string'
									? 'text-emerald-300'
									: entry.value === null
										? 'text-slate-500'
										: 'text-amber-300'
								: 'text-slate-400'}
						>{stringifyValue(entry.value)}</span>
					</button>
				{/if}
			{/each}
		{/if}
	</div>
</div>
