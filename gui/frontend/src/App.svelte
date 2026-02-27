<script lang="ts">
	import './app.css';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Select, SelectContent, SelectItem, SelectTrigger } from '$lib/components/ui/select';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import { Badge } from '$lib/components/ui/badge';
	import { ScrollArea } from '$lib/components/ui/scroll-area';
	import { Separator } from '$lib/components/ui/separator';
	import { Alert, AlertDescription } from '$lib/components/ui/alert';
	import { Plus, Trash2, Send, Copy, Download } from '@lucide/svelte/icons';

	// State variables
	let url = $state('https://api.example.com/data');
	let selectedMethod = $state('GET');
	let requestBody = $state('');
	let headers = $state<Array<{key: string, value: string}>>([
		{ key: 'Content-Type', value: 'application/json' }
	]);
	let response = $state('');
	let responseStatus = $state(0);
	let responseHeaders = $state('');
	let responseTime = $state(0);
	let isLoading = $state(false);
	let error = $state('');

	// Methods
	const httpMethods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'];

	function addHeader() {
		headers = [...headers, { key: '', value: '' }];
	}

	function removeHeader(index: number) {
		headers = headers.filter((_, i) => i !== index);
	}

	function updateHeader(index: number, field: 'key' | 'value', value: string) {
		headers[index][field] = value;
		headers = headers;
	}

	async function sendRequest() {
		isLoading = true;
		error = '';
		response = '';
		responseStatus = 0;
		responseHeaders = '';
		
		const startTime = performance.now();
		
		try {
			// TODO: Call Wails backend function here
			// For now, just simulate a response
			await new Promise(resolve => setTimeout(resolve, 1000));
			
			responseTime = Math.round(performance.now() - startTime);
			responseStatus = 200;
			response = JSON.stringify({
				message: "This is a mock response",
				timestamp: new Date().toISOString(),
				method: selectedMethod,
				url,
				headers: headers.filter(h => h.key && h.value)
			}, null, 2);
			responseHeaders = `HTTP/1.1 200 OK
Content-Type: application/json
Date: ${new Date().toUTCString()}
Content-Length: ${response.length}`;
			
		} catch (err) {
			error = err instanceof Error ? err.message : 'An unknown error occurred';
			responseTime = Math.round(performance.now() - startTime);
		} finally {
			isLoading = false;
		}
	}

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

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	}

	function getStatusColor(status: number): string {
		if (status >= 200 && status < 300) return 'bg-gradient-to-r from-green-500 to-green-600 text-white font-bold';
		if (status >= 300 && status < 400) return 'bg-gradient-to-r from-yellow-500 to-yellow-600 text-white font-bold';
		if (status >= 400 && status < 500) return 'bg-gradient-to-r from-orange-500 to-orange-600 text-white font-bold';
		if (status >= 500) return 'bg-gradient-to-r from-red-500 to-red-600 text-white font-bold';
		return 'bg-gradient-to-r from-gray-500 to-gray-600 text-white font-bold';
	}
</script>

<main class="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-950 dark:to-slate-900 p-6">
	<div class="mx-auto max-w-6xl space-y-6">
		<!-- Header -->
		<Card class="border-0 bg-gradient-to-r from-blue-600 to-purple-600 text-white shadow-xl">
			<CardHeader class="pb-3">
				<div class="flex items-center gap-3">
					<div class="rounded-lg bg-white/20 p-2 backdrop-blur">
						<svg class="h-8 w-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
						</svg>
					</div>
					<div>
						<CardTitle class="text-3xl font-bold text-white">API Tester</CardTitle>
						<CardDescription class="text-blue-50">Test HTTP APIs with a modern, easy-to-use interface</CardDescription>
					</div>
				</div>
			</CardHeader>
		</Card>

		<!-- Request Section -->
		<Card class="shadow-lg border-slate-200 dark:border-slate-800">
			<CardHeader class="bg-gradient-to-r from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800">
				<CardTitle class="flex items-center gap-2">
					<svg class="h-5 w-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
					</svg>
					Request Configuration
				</CardTitle>
			</CardHeader>
			<CardContent class="space-y-4 pt-6">
				<!-- URL and Method -->
				<div class="flex gap-3">
					<Select type="single" bind:value={selectedMethod}>
						<SelectTrigger class="w-32 font-semibold border-2 hover:border-blue-300 focus:border-blue-500 transition-colors">
							<span class={selectedMethod === 'GET' ? 'text-green-600' : selectedMethod === 'POST' ? 'text-blue-600' : selectedMethod === 'PUT' ? 'text-yellow-600' : selectedMethod === 'DELETE' ? 'text-red-600' : 'text-purple-600'}>
								{selectedMethod || 'GET'}
							</span>
						</SelectTrigger>
						<SelectContent>
							{#each httpMethods as m}
								<SelectItem value={m} class="font-medium">
									<span class={m === 'GET' ? 'text-green-600' : m === 'POST' ? 'text-blue-600' : m === 'PUT' ? 'text-yellow-600' : m === 'DELETE' ? 'text-red-600' : 'text-purple-600'}>
										{m}
									</span>
								</SelectItem>
							{/each}
						</SelectContent>
					</Select>
					<Input
						type="url"
						placeholder="Enter request URL (e.g., https://api.example.com/users)"
						bind:value={url}
						class="flex-1 border-2 hover:border-blue-300 focus:border-blue-500 transition-colors font-mono"
					/>
					<Button 
						onclick={sendRequest} 
						disabled={isLoading || !url}
						class="min-w-32 bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 text-white font-semibold shadow-md transition-all transform hover:scale-105"
					>
						{#if isLoading}
							<span class="animate-spin mr-2">⟳</span>
							Sending...
						{:else}
							<Send class="mr-2 h-4 w-4" />
							Send Request
						{/if}
					</Button>
				</div>

				<!-- Request Configuration Tabs -->
				<Tabs value="headers" class="w-full">
					<TabsList class="grid w-full grid-cols-2">
						<TabsTrigger value="headers">Headers</TabsTrigger>
						<TabsTrigger value="body" disabled={selectedMethod === 'GET' || selectedMethod === 'HEAD'}>Body</TabsTrigger>
					</TabsList>
					
					<!-- Headers Tab -->
					<TabsContent value="headers" class="space-y-2">
						<div class="space-y-2">
							{#each headers as header, i}
								<div class="flex gap-2">
									<Input
										placeholder="Header name"
										bind:value={header.key}
										onchange={(e) => updateHeader(i, 'key', e.currentTarget.value)}
										class="flex-1"
									/>
									<Input
										placeholder="Header value"
										bind:value={header.value}
										onchange={(e) => updateHeader(i, 'value', e.currentTarget.value)}
										class="flex-1"
									/>
									<Button
										variant="outline"
										size="icon"
										onclick={() => removeHeader(i)}
									>
										<Trash2 class="h-4 w-4" />
									</Button>
								</div>
							{/each}
						</div>
						<Button
							variant="outline"
							onclick={addHeader}
							class="w-full"
						>
							<Plus class="mr-2 h-4 w-4" />
							Add Header
						</Button>
					</TabsContent>
					
					<!-- Body Tab -->
					<TabsContent value="body">
						<Textarea
							placeholder="Request body (JSON, XML, etc.)"
							bind:value={requestBody}
							class="min-h-[200px] font-mono text-sm"
						/>
					</TabsContent>
				</Tabs>
			</CardContent>
		</Card>

		<!-- Response Section -->
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
							<Button
								variant="outline"
								size="icon"
								onclick={copyResponse}
							>
								<Copy class="h-4 w-4" />
							</Button>
							<Button
								variant="outline"
								size="icon"
								onclick={downloadResponse}
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
						<TabsList class="grid w-full grid-cols-2">
							<TabsTrigger value="body">Body</TabsTrigger>
							<TabsTrigger value="headers">Headers</TabsTrigger>
						</TabsList>
						
						<TabsContent value="body">
							<ScrollArea class="h-[400px] w-full rounded-lg border-2 border-slate-200 dark:border-slate-700 bg-slate-950 shadow-inner">
								<pre class="p-6 text-sm font-mono text-green-400 leading-relaxed">{response}</pre>
							</ScrollArea>
						</TabsContent>
						
						<TabsContent value="headers">
							<ScrollArea class="h-[400px] w-full rounded-lg border-2 border-slate-200 dark:border-slate-700 bg-slate-950 shadow-inner">
								<pre class="p-6 text-sm font-mono text-blue-400 leading-relaxed">{responseHeaders}</pre>
							</ScrollArea>
						</TabsContent>
					</Tabs>
				{:else}
					<div class="flex h-[200px] items-center justify-center rounded-lg border-2 border-dashed border-slate-300 dark:border-slate-700 bg-slate-50/50 dark:bg-slate-900/50">
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
	</div>
</main>

<style>
	:global(body) {
		margin: 0;
		padding: 0;
	}
</style>