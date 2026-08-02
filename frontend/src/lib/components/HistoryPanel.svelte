<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Clock } from '@lucide/svelte/icons';
	import { methodColorClass } from '$lib/format';
	import type { HistoryEntry } from '$lib/history';

	let {
		history,
		activeMethod,
		onrestore,
		onclear,
	} = $props<{
		history: HistoryEntry[];
		activeMethod: string;
		onrestore?: (entry: HistoryEntry) => void;
		onclear?: () => void;
	}>();
</script>

<Card class="shadow-lg border-slate-200 dark:border-slate-800">
	<CardHeader class="bg-gradient-to-r from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800">
		<div class="flex items-center justify-between">
			<CardTitle class="flex items-center gap-2 text-lg">
				<Clock class="h-4 w-4 text-blue-600" />
				History
			</CardTitle>
			<Button variant="outline" size="sm" onclick={onclear} class="text-xs">
				Clear
			</Button>
		</div>
	</CardHeader>
	<CardContent class="pt-4">
		<ul class="divide-y divide-slate-200 dark:divide-slate-800">
			{#each history as entry}
				<li>
					<button
						class="w-full flex items-center gap-3 px-2 py-2 text-left hover:bg-slate-50 dark:hover:bg-slate-800/50 rounded transition-colors"
						onclick={() => onrestore?.(entry)}
						title="Restore request"
					>
						<span class={`w-14 shrink-0 text-center text-xs font-bold py-0.5 rounded ${activeMethod === entry.method ? 'ring-2 ring-blue-400' : ''}`}>
							<span class={methodColorClass(entry.method)}>
								{entry.method}
							</span>
						</span>
						<span class="flex-1 truncate font-mono text-sm text-slate-700 dark:text-slate-300">{entry.url}</span>
						<span class="text-xs text-slate-400 shrink-0">{new Date(entry.timestamp).toLocaleTimeString()}</span>
					</button>
				</li>
			{/each}
		</ul>
	</CardContent>
</Card>
