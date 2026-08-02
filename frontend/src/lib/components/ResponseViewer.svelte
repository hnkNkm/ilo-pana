<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { ScrollArea } from '$lib/components/ui/scroll-area';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Copy, Download } from '@lucide/svelte/icons';
	import JsonTree from '$lib/components/JsonTree.svelte';
	import { formatBytes, formatJson, getStatusColor } from '$lib/format';
	import type { AssertionResult } from '$lib/app-client';

	let {
		response,
		responseStatus,
		responseTime,
		responseHeaders,
		error,
		assertionResults,
	} = $props<{
		response: string;
		responseStatus: number;
		responseTime: number;
		responseHeaders: string;
		error: string;
		assertionResults: AssertionResult[];
	}>();

	let responseViewMode = $state<'raw' | 'pretty' | 'tree'>('pretty');

	const parsedResponse = $derived.by(() => {
		try {
			return JSON.parse(response);
		} catch {
			return null;
		}
	});
	const responseIsJson = $derived(parsedResponse !== null);
	const assertionPassedCount = $derived(assertionResults.filter((r: AssertionResult) => r.passed).length);

	function copyResponse() {
		navigator.clipboard.writeText(response);
	}

	function downloadResponse() {
		const blob = new Blob([response], { type: 'application/json' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `response-${Date.now()}.json`;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		URL.revokeObjectURL(url);
	}
</script>

<Card class="shadow-lg border-slate-200 dark:border-slate-800">
	<CardHeader class="bg-gradient-to-r from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800">
		<div class="flex items-center justify-between">
			<CardTitle class="flex items-center gap-2">
				<svg class="h-5 w-5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
				Response
			</CardTitle>
			{#if response}
				<div class="flex items-center gap-2">
					<Badge class={responseStatus ? getStatusColor(responseStatus) : ''}>
						{responseStatus || 'N/A'}
					</Badge>
					<Badge variant="outline">{responseTime}ms</Badge>
					<Badge variant="outline">{formatBytes(response.length)}</Badge>
					{#if assertionResults.length > 0}
						<Badge
							class={assertionPassedCount === assertionResults.length
								? 'bg-emerald-600 text-white'
								: 'bg-red-600 text-white'}
						>
							{assertionPassedCount}/{assertionResults.length} assertions
						</Badge>
					{/if}
					<Button
						variant="outline"
						size="icon"
						onclick={copyResponse}
						title="Copy response body to clipboard"
					>
						<Copy class="h-4 w-4" />
					</Button>
					<Button
						variant="outline"
						size="icon"
						onclick={downloadResponse}
						title="Download response body as JSON"
					>
						<Download class="h-4 w-4" />
					</Button>
				</div>
			{/if}
		</div>
	</CardHeader>
	<CardContent>
		{#if error}
			<Alert variant="destructive">
				<AlertDescription>{error}</AlertDescription>
			</Alert>
		{:else if response}
			<Tabs value="body" class="w-full">
				<TabsList class="grid w-full grid-cols-3">
					<TabsTrigger value="body">Body</TabsTrigger>
					<TabsTrigger value="headers">Headers</TabsTrigger>
					<TabsTrigger value="assertions">Assertions</TabsTrigger>
				</TabsList>

				<TabsContent value="body" class="mt-4">
					<div class="rounded-lg border border-slate-700 bg-slate-900 shadow-xl overflow-hidden">
						<div class="flex items-center justify-between px-4 py-2 bg-slate-800 border-b border-slate-700">
							<div class="flex items-center gap-2">
								<div class="w-3 h-3 rounded-full bg-red-500"></div>
								<div class="w-3 h-3 rounded-full bg-yellow-500"></div>
								<div class="w-3 h-3 rounded-full bg-green-500"></div>
								<span class="text-xs text-slate-400 font-mono">response.json</span>
							</div>
							<div class="flex items-center gap-2">
								{#if responseIsJson}
									<div class="flex rounded-md bg-slate-900 border border-slate-700 p-0.5 text-[11px] font-mono">
										<button
											class="rounded px-2 py-0.5 {responseViewMode === 'raw' ? 'bg-slate-700 text-white' : 'text-slate-400 hover:text-white'}"
											onclick={() => (responseViewMode = 'raw')}
											title="Raw response text"
										>Raw</button>
										<button
											class="rounded px-2 py-0.5 {responseViewMode === 'pretty' ? 'bg-slate-700 text-white' : 'text-slate-400 hover:text-white'}"
											onclick={() => (responseViewMode = 'pretty')}
											title="Pretty-printed JSON"
										>Pretty</button>
										<button
											class="rounded px-2 py-0.5 {responseViewMode === 'tree' ? 'bg-slate-700 text-white' : 'text-slate-400 hover:text-white'}"
											onclick={() => (responseViewMode = 'tree')}
											title="Collapsible JSON tree with search"
										>Tree</button>
									</div>
								{/if}
								<button onclick={copyResponse} class="text-slate-400 hover:text-white transition-colors p-1" title="Copy to clipboard">
									<Copy class="h-4 w-4" />
								</button>
							</div>
						</div>
						{#if responseIsJson && responseViewMode === 'tree'}
							<div class="h-[600px] w-full bg-gray-900">
								<JsonTree value={parsedResponse} />
							</div>
						{:else if responseIsJson && responseViewMode === 'raw'}
							<ScrollArea class="h-[600px] w-full bg-gray-900" orientation="both">
								<pre class="block w-full min-w-max p-4 text-left text-[12px] font-mono leading-[1.4] text-gray-200 whitespace-pre">{response}</pre>
							</ScrollArea>
						{:else}
							<ScrollArea class="h-[600px] w-full bg-gray-900" orientation="both">
								<pre class="block w-full min-w-max p-4 text-left text-[12px] font-mono leading-[1.4] text-gray-200 whitespace-pre">{@html formatJson(response)}</pre>
							</ScrollArea>
						{/if}
					</div>
				</TabsContent>

				<TabsContent value="headers" class="mt-4">
					<div class="rounded-lg border border-slate-700 bg-slate-900 shadow-xl overflow-hidden">
						<div class="flex items-center justify-between px-4 py-2 bg-slate-800 border-b border-slate-700">
							<div class="flex items-center gap-2">
								<div class="w-3 h-3 rounded-full bg-red-500"></div>
								<div class="w-3 h-3 rounded-full bg-yellow-500"></div>
								<div class="w-3 h-3 rounded-full bg-green-500"></div>
							</div>
							<span class="text-xs text-slate-400 font-mono">headers</span>
						</div>
						<ScrollArea class="h-[600px] w-full bg-gray-900" orientation="both">
							<pre class="block w-full min-w-max p-6 text-left text-[14px] font-mono text-cyan-400 leading-[1.65] whitespace-pre">{responseHeaders}</pre>
						</ScrollArea>
					</div>
				</TabsContent>

				<TabsContent value="assertions" class="mt-4">
					{#if assertionResults.length > 0}
						<div class="rounded-lg border border-slate-700 bg-slate-900 shadow-xl overflow-hidden">
							<div class="flex items-center justify-between px-4 py-2 bg-slate-800 border-b border-slate-700">
								<span class="text-xs text-slate-400 font-mono">assertions</span>
								<span class="text-xs font-mono {assertionPassedCount === assertionResults.length ? 'text-emerald-400' : 'text-red-400'}">
									{assertionPassedCount}/{assertionResults.length} passed
								</span>
							</div>
							<ul class="p-3 space-y-2">
								{#each assertionResults as r}
									<li
										class="flex items-start gap-2 rounded-md border px-3 py-2 text-sm font-mono
											{r.passed
												? 'border-emerald-700/60 bg-emerald-950/40 text-emerald-300'
												: 'border-red-700/60 bg-red-950/40 text-red-300'}"
									>
										<span>{r.passed ? 'PASS' : 'FAIL'}</span>
										<span class="flex-1">
											{r.rule.name ? `${r.rule.name}: ` : ''}{r.message}
										</span>
									</li>
								{/each}
							</ul>
						</div>
					{:else}
						<div class="flex h-[300px] items-center justify-center rounded-lg border-2 border-dashed border-slate-300 dark:border-slate-700 bg-slate-50/50 dark:bg-slate-900/50">
							<p class="text-sm text-slate-500">No assertions configured. Add rules in the Assertions request tab.</p>
						</div>
					{/if}
				</TabsContent>
			</Tabs>
		{:else}
			<div class="flex h-[400px] items-center justify-center rounded-lg border-2 border-dashed border-slate-300 dark:border-slate-700 bg-slate-50/50 dark:bg-slate-900/50">
				<div class="text-center">
					<svg class="mx-auto h-12 w-12 text-slate-400 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
					</svg>
					<p class="text-lg font-medium text-slate-600 dark:text-slate-400">No response yet</p>
					<p class="text-sm text-slate-500 dark:text-slate-500 mt-1">Send a request to see the result</p>
				</div>
			</div>
		{/if}
	</CardContent>
</Card>
