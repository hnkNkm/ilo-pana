<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Trash2 } from '@lucide/svelte/icons';
	import { methodColorClass } from '$lib/format';
	import type { Collection, SavedRequest } from '$lib/app-client';

	let {
		collections,
		onload,
		ondeleterequest,
		ondeletecollection,
		onexport,
	} = $props<{
		collections: Collection[];
		onload?: (c: Collection, req: SavedRequest) => void;
		ondeleterequest?: (c: Collection, req: SavedRequest) => void;
		ondeletecollection?: (c: Collection) => void;
		onexport?: (c: Collection) => void;
	}>();
</script>

<Card class="shadow-lg border-slate-200 dark:border-slate-800">
	<CardHeader class="bg-gradient-to-r from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800">
		<CardTitle class="flex items-center gap-2 text-lg">
			<svg class="h-4 w-4 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V7z" />
			</svg>
			Collections
		</CardTitle>
	</CardHeader>
	<CardContent class="pt-4 space-y-3">
		{#each collections as c}
			<div class="rounded-lg border border-slate-200 dark:border-slate-800 overflow-hidden">
				<div class="flex items-center justify-between px-3 py-2 bg-slate-100 dark:bg-slate-800/60">
					<span class="text-sm font-semibold text-slate-700 dark:text-slate-200">{c.name}</span>
					<div class="flex items-center gap-1">
						<Button variant="ghost" size="sm" class="text-xs" onclick={() => onexport?.(c)} title="Copy collection JSON to clipboard">
							Export
						</Button>
						<Button variant="ghost" size="sm" class="text-xs text-red-600 hover:text-red-700" onclick={() => ondeletecollection?.(c)} title="Delete collection">
							Delete
						</Button>
					</div>
				</div>
				<ul class="divide-y divide-slate-100 dark:divide-slate-800">
					{#each c.requests as req}
						<li class="flex items-center gap-2 px-3 py-1.5 hover:bg-slate-50 dark:hover:bg-slate-800/50">
							<button
								class="flex-1 flex items-center gap-2 text-left font-mono text-sm text-slate-700 dark:text-slate-300"
								onclick={() => onload?.(c, req)}
								title="Load request"
							>
								<span class="text-xs font-bold w-12 shrink-0 text-right">
									<span class={methodColorClass(req.method)}>
										{req.method}
									</span>
								</span>
								<span class="text-slate-500 dark:text-slate-400 shrink-0">{req.name}</span>
								<span class="truncate text-xs text-slate-400">{req.url}</span>
							</button>
							<Button
								variant="ghost"
								size="icon"
								class="h-6 w-6"
								onclick={() => ondeleterequest?.(c, req)}
								title="Delete request"
							>
								<Trash2 class="h-3.5 w-3.5" />
							</Button>
						</li>
					{/each}
				</ul>
			</div>
		{/each}
	</CardContent>
</Card>
