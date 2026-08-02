<script lang="ts">
	import './app.css';
	import { onMount } from 'svelte';
	import { ExecuteRequest, ListCollections, GetCollection, SaveRequest, DeleteRequest, DeleteCollection, ExportCollection, ImportCollection, ImportOpenAPI, SaveEnvironment, ListEnvironments, GetEnvironment, DeleteEnvironment } from '$wailsjs/go/main/App';
	import { collection, environment } from '$wailsjs/go/models';
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
	import { Plus, Trash2, Send, Copy, Download, Clock } from '@lucide/svelte/icons';

	// State variables
	let url = $state('https://pokeapi.co/api/v2/pokemon/pikachu');
	let selectedMethod = $state('GET');
	let requestBody = $state('');
	let headers = $state<Array<{key: string, value: string}>>([
		{ key: 'Content-Type', value: 'application/json' }
	]);
	let queryParams = $state<Array<{key: string, value: string, enabled: boolean}>>([]);
	let variables = $state<Array<{key: string, value: string}>>([]);
	let timeoutSec = $state(30);
	let sessionName = $state('');
	let sessionNew = $state(false);
	let response = $state('');
	let responseStatus = $state(0);
	let responseHeaders = $state('');
	let responseTime = $state(0);
	let isLoading = $state(false);
	let error = $state('');
	let authType = $state('none');
	let authUsername = $state('');
	let authPassword = $state('');
	let authToken = $state('');

	// Environments (persisted variable sets, e.g. dev/staging/prod)
	let environmentName = $state(''); // selected environment ('' = none)
	let environments = $state<Array<string>>([]);
	let envName = $state('');
	let envVars = $state<Array<{key: string, value: string}>>([]);
	let envError = $state('');
	let envMessage = $state('');

	// Request history (persisted in localStorage)
	const HISTORY_KEY = 'ilo-pana-request-history';
	const HISTORY_MAX = 10;

	interface HistoryEntry {
		method: string;
		url: string;
		headers: {key: string, value: string}[];
		body: string;
		timestamp: number;
	}

	function loadHistory(): HistoryEntry[] {
		try {
			const raw = localStorage.getItem(HISTORY_KEY);
			if (!raw) return [];
			const parsed = JSON.parse(raw);
			return Array.isArray(parsed) ? parsed : [];
		} catch {
			return [];
		}
	}

	let history = $state<HistoryEntry[]>(loadHistory());

	function saveToHistory() {
		const entry: HistoryEntry = {
			method: selectedMethod,
			url,
			headers: headers.filter(h => h.key?.trim()),
			body: requestBody,
			timestamp: Date.now(),
		};
		// Deduplicate by method + URL (keep the latest)
		const filtered = history.filter(h => !(h.method === entry.method && h.url === entry.url));
		history = [entry, ...filtered].slice(0, HISTORY_MAX);
		try {
			localStorage.setItem(HISTORY_KEY, JSON.stringify(history));
		} catch {
			// localStorage unavailable (private mode etc.) - history just won't persist
		}
	}

	function restoreRequest(entry: HistoryEntry) {
		selectedMethod = entry.method;
		url = entry.url;
		parseQueryParams();
		requestBody = entry.body;
		headers = entry.headers.length ? entry.headers : [{ key: 'Content-Type', value: 'application/json' }];
		response = '';
		responseStatus = 0;
		responseHeaders = '';
		error = '';
	}

	function clearHistory() {
		history = [];
		try {
			localStorage.removeItem(HISTORY_KEY);
		} catch {
			// ignore
		}
	}

	// Keyboard shortcuts
	function handleKeydown(e: KeyboardEvent) {
		if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
			e.preventDefault();
			if (!isLoading && url) sendRequest();
		}
	}

	onMount(() => {
		window.addEventListener('keydown', handleKeydown);
		refreshCollections();
		refreshEnvironments();
		return () => window.removeEventListener('keydown', handleKeydown);
	});

	// Environments (persisted variable sets)
	async function refreshEnvironments() {
		try {
			const names = await ListEnvironments();
			environments = names;
			if (environmentName && !names.includes(environmentName)) {
				environmentName = '';
			}
		} catch {
			// environments just won't load
		}
	}

	async function loadEnvironmentForEdit(name: string) {
		envError = '';
		envMessage = '';
		try {
			const env = await GetEnvironment(name);
			envName = env.name;
			envVars = env.variables
				? Object.entries(env.variables).map(([key, value]) => ({ key, value }))
				: [];
		} catch (e) {
			envError = extractErrorMessage(e);
		}
	}

	async function saveEnvironment() {
		envError = '';
		envMessage = '';
		const name = envName.trim();
		if (!name) {
			envError = 'Enter an environment name.';
			return;
		}
		const varsMap: Record<string, string> = {};
		for (const v of envVars) {
			if (v.key?.trim()) varsMap[v.key.trim()] = v.value ?? '';
		}
		try {
			await SaveEnvironment(name, varsMap);
			envMessage = `Saved environment "${name}".`;
			await refreshEnvironments();
		} catch (e) {
			envError = extractErrorMessage(e);
		}
	}

	async function removeEnvironment(name: string) {
		envError = '';
		envMessage = '';
		try {
			await DeleteEnvironment(name);
			if (envName === name) {
				envName = '';
				envVars = [];
			}
			await refreshEnvironments();
		} catch (e) {
			envError = extractErrorMessage(e);
		}
	}

	function addEnvVar() {
		envVars = [...envVars, { key: '', value: '' }];
	}

	function removeEnvVar(index: number) {
		envVars = envVars.filter((_, i) => i !== index);
	}

	function updateEnvVar(index: number, field: 'key' | 'value', value: string) {
		envVars[index][field] = value;
		envVars = envVars;
	}

	const historyUrls = $derived([...new Set(history.map(h => h.url))]);

	// Collections (saved requests)
	let collectionName = $state('');
	let requestName = $state('');
	let savedCollections = $state<collection.Collection[]>([]);
	let collectionError = $state('');
	let collectionMessage = $state('');

	async function refreshCollections() {
		try {
			const names = await ListCollections();
			const loaded = [];
			for (const name of names) {
				try {
					loaded.push(await GetCollection(name));
				} catch {
					// skip unreadable collection
				}
			}
			savedCollections = loaded;
		} catch (e) {
			collectionError = extractErrorMessage(e);
		}
	}

	async function saveCurrentRequest() {
		collectionError = '';
		collectionMessage = '';
		if (!collectionName.trim()) {
			collectionError = 'Enter a collection name to save this request.';
			return;
		}
		if (!requestName.trim()) {
			collectionError = 'Enter a request name to save this request.';
			return;
		}

		const headersMap: Record<string, string> = {};
		for (const h of headers) {
			if (h.key?.trim()) headersMap[h.key.trim()] = h.value ?? '';
		}
		const variablesMap: Record<string, string> = {};
		for (const v of variables) {
			if (v.key?.trim()) variablesMap[v.key.trim()] = v.value ?? '';
		}

		try {
			await SaveRequest(collectionName.trim(), new collection.SavedRequest({
				name: requestName.trim(),
				method: selectedMethod,
				url,
				headers: headersMap,
				body: requestBody,
				variables: variablesMap,
			}));
			collectionMessage = `Saved "${requestName.trim()}" to "${collectionName.trim()}".`;
			await refreshCollections();
		} catch (e) {
			collectionError = extractErrorMessage(e);
		}
	}

	async function loadSavedRequest(c: collection.Collection, req: collection.SavedRequest) {
		selectedMethod = req.method;
		url = req.url;
		parseQueryParams();
		requestBody = req.body ?? '';
		headers = req.headers && Object.keys(req.headers).length
			? Object.entries(req.headers).map(([key, value]) => ({ key, value }))
			: [{ key: 'Content-Type', value: 'application/json' }];
		variables = req.variables
			? Object.entries(req.variables).map(([key, value]) => ({ key, value }))
			: [];
		response = '';
		responseStatus = 0;
		responseHeaders = '';
		error = '';
		collectionError = '';
		collectionMessage = '';
	}

	async function removeSavedRequest(c: collection.Collection, req: collection.SavedRequest) {
		collectionError = '';
		collectionMessage = '';
		try {
			await DeleteRequest(c.name, req.name);
			if (c.requests.length === 1) {
				await refreshCollections();
			} else {
				const fresh = await GetCollection(c.name);
				const idx = savedCollections.findIndex(s => s.name === c.name);
				if (idx >= 0) savedCollections[idx] = fresh;
				savedCollections = savedCollections;
			}
		} catch (e) {
			collectionError = extractErrorMessage(e);
		}
	}

	async function removeCollection(c: collection.Collection) {
		collectionError = '';
		collectionMessage = '';
		try {
			await DeleteCollection(c.name);
			await refreshCollections();
		} catch (e) {
			collectionError = extractErrorMessage(e);
		}
	}

	async function exportCollection(c: collection.Collection) {
		collectionError = '';
		collectionMessage = '';
		try {
			const json = await ExportCollection(c.name);
			await navigator.clipboard.writeText(json);
			collectionMessage = `Exported "${c.name}" to clipboard.`;
		} catch (e) {
			collectionError = extractErrorMessage(e);
		}
	}

	async function importCollectionFromClipboard() {
		collectionError = '';
		collectionMessage = '';
		try {
			const json = await navigator.clipboard.readText();
			await ImportCollection(json);
			collectionMessage = 'Collection imported from clipboard.';
			await refreshCollections();
		} catch (e) {
			collectionError = extractErrorMessage(e);
		}
	}

	let openapiFileInput: HTMLInputElement | undefined = $state();

	async function importOpenAPIFromFile() {
		collectionError = '';
		collectionMessage = '';
		const file = openapiFileInput?.files?.[0];
		if (!file) return;
		try {
			const content = await file.text();
			const target = collectionName.trim();
			const n = await ImportOpenAPI(content, target);
			collectionMessage = target
				? `Imported ${n} endpoints into "${target}".`
				: `Imported ${n} endpoints.`;
			await refreshCollections();
		} catch (e) {
			collectionError = extractErrorMessage(e);
		} finally {
			if (openapiFileInput) openapiFileInput.value = '';
		}
	}

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

	// Query parameters — parsed from / merged into the URL
	function parseQueryParams() {
		const q = url.split('?')[1];
		const params: Array<{key: string, value: string, enabled: boolean}> = [];
		if (q) {
			for (const part of q.split('&')) {
				if (!part) continue;
				const eq = part.indexOf('=');
				const rawKey = eq >= 0 ? part.slice(0, eq) : part;
				const rawValue = eq >= 0 ? part.slice(eq + 1) : '';
				try {
					params.push({ key: decodeURIComponent(rawKey), value: decodeURIComponent(rawValue), enabled: true });
				} catch {
					params.push({ key: rawKey, value: rawValue, enabled: true });
				}
			}
		}
		queryParams = params;
	}

	function buildUrlFromParams() {
		const base = url.split('?')[0];
		const parts = queryParams
			.filter(p => p.enabled && p.key.trim())
			.map(p => `${encodeURIComponent(p.key.trim())}=${encodeURIComponent(p.value)}`);
		url = parts.length ? `${base}?${parts.join('&')}` : base;
	}

	function addQueryParam() {
		queryParams = [...queryParams, { key: '', value: '', enabled: true }];
	}

	function removeQueryParam(index: number) {
		queryParams = queryParams.filter((_, i) => i !== index);
		buildUrlFromParams();
	}

	function updateQueryParam(index: number, field: 'key' | 'value' | 'enabled', value: string | boolean) {
		if (field === 'enabled') {
			queryParams[index].enabled = value as boolean;
		} else {
			queryParams[index][field] = value as string;
		}
		queryParams = queryParams;
		buildUrlFromParams();
	}

	// Auth helpers
	function btoaUtf8(s: string): string {
		return btoa(unescape(encodeURIComponent(s)));
	}

	function authPreview(): string {
		if (authType === 'basic') {
			return authUsername.trim()
				? 'Authorization: Basic ••••••'
				: 'Enter username and password to generate the Authorization header';
		}
		if (authType === 'bearer') {
			return authToken.trim()
				? 'Authorization: Bearer ••••••'
				: 'Enter a token to generate the Authorization header';
		}
		return '';
	}

	function addVariable() {
		variables = [...variables, { key: '', value: '' }];
	}

	function removeVariable(index: number) {
		variables = variables.filter((_, i) => i !== index);
	}

	function updateVariable(index: number, field: 'key' | 'value', value: string) {
		variables[index][field] = value;
		variables = variables;
	}

	function extractErrorMessage(err: unknown): string {
		if (err instanceof Error) return err.message;
		if (typeof err === 'string') return err;
		if (err && typeof err === 'object' && 'message' in err && typeof (err as { message: unknown }).message === 'string') {
			return (err as { message: string }).message;
		}
		return 'An unknown error occurred';
	}

	async function sendRequest() {
		isLoading = true;
		error = '';
		response = '';
		responseStatus = 0;
		responseHeaders = '';

		const headersMap: Record<string, string> = {};
		for (const h of headers) {
			if (h.key?.trim()) {
				headersMap[h.key.trim()] = h.value ?? '';
			}
		}

		// Auth header (overrides a manually-entered Authorization header)
		if (authType === 'basic' && authUsername.trim()) {
			headersMap['Authorization'] = `Basic ${btoaUtf8(`${authUsername}:${authPassword}`)}`;
		} else if (authType === 'bearer' && authToken.trim()) {
			headersMap['Authorization'] = `Bearer ${authToken}`;
		}

		const variablesMap: Record<string, string> = {};
		for (const v of variables) {
			if (v.key?.trim()) {
				variablesMap[v.key.trim()] = v.value ?? '';
			}
		}

		try {
			const result = await ExecuteRequest({
				Method: selectedMethod,
				URL: url,
				Body: requestBody,
				Headers: headersMap,
				TimeoutMs: timeoutSec * 1000,
				SessionName: sessionName,
				SessionNew: sessionNew,
				Variables: variablesMap,
				Environment: environmentName,
			});

			responseStatus = result.statusCode;
			responseTime = result.elapsedMs;
			response = result.body || '(empty response body)';
			responseHeaders = Object.entries(result.headers)
				.map(([k, v]) => `${k}: ${v}`)
				.join('\n');

			saveToHistory();
		} catch (err) {
			error = extractErrorMessage(err);
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

	function escapeHtml(s: string): string {
		return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
	}

	function formatJson(json: string): string {
		if (!json) return '';
		try {
			const parsed = JSON.parse(json);
			// Standard JSON formatting with 2 spaces
			const formatted = JSON.stringify(parsed, null, 2);

			// Escape HTML first so untrusted response content cannot inject
			// elements/scripts, then apply syntax highlighting on the escaped text.
			// Quotes are intentionally NOT escaped: they are safe in text context
			// and the highlighting regexes rely on them.
			const escaped = escapeHtml(formatted);

			// Simple, clean syntax highlighting
			return escaped
				// Keys - light blue
				.replace(/"([^"]+)":/g, '<span class="text-sky-400">"$1"</span>:')
				// String values - light green  
				.replace(/:\s*"([^"]*)"/g, ': <span class="text-green-400">"$1"</span>')
				// Numbers - yellow
				.replace(/:\s*(-?\d+\.?\d*)/g, ': <span class="text-yellow-400">$1</span>')
				// Booleans - purple
				.replace(/:\s*(true|false)/g, ': <span class="text-purple-400">$1</span>')
				// Null - red
				.replace(/:\s*(null)/g, ': <span class="text-red-400">$1</span>');
		} catch {
			return json;
		}
	}
</script>

<main class="min-h-screen bg-gradient-to-br from-slate-50 to-slate-100 dark:from-slate-950 dark:to-slate-900 p-4">
	<div class="space-y-4">
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
						placeholder="Enter request URL (e.g., https://pokeapi.co/api/v2/pokemon/pikachu)"
						bind:value={url}
						oninput={parseQueryParams}
						list="url-history"
						class="flex-1 border-2 hover:border-blue-300 focus:border-blue-500 transition-colors font-mono"
					/>
					<datalist id="url-history">
						{#each historyUrls as u}
							<option value={u}></option>
						{/each}
					</datalist>
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

				<!-- Options row: timeout, session -->
				<div class="flex flex-wrap items-end gap-3">
					<div class="flex items-center gap-2">
						<Label for="timeout-input" class="whitespace-nowrap text-sm">Timeout (s)</Label>
						<Input
							id="timeout-input"
							type="number"
							min="1"
							max="300"
							bind:value={timeoutSec}
							class="w-24 text-right font-mono"
						/>
					</div>
					<div class="flex items-center gap-2">
						<Label for="session-name-input" class="whitespace-nowrap text-sm">Session</Label>
						<Input
							id="session-name-input"
							type="text"
							placeholder="Session name (optional)"
							bind:value={sessionName}
							class="w-48 font-mono"
						/>
						<label class="flex items-center gap-2 text-sm cursor-pointer">
							<input type="checkbox" bind:checked={sessionNew} class="h-4 w-4" />
							New session
						</label>
					</div>
					<div class="flex items-center gap-2">
						<Label for="environment-select" class="whitespace-nowrap text-sm">Environment</Label>
						<Select type="single" bind:value={environmentName}>
							<SelectTrigger class="w-44 font-mono text-sm" id="environment-select-trigger">
								<span class={environmentName ? 'text-emerald-600' : 'text-slate-400'}>
									{environmentName || 'No Environment'}
								</span>
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="">No Environment</SelectItem>
								{#each environments as name}
									<SelectItem value={name} class="font-mono">{name}</SelectItem>
								{/each}
							</SelectContent>
						</Select>
						{#if environmentName}
							<Badge variant="outline" class="text-xs bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400">
								Using {environmentName} variables
							</Badge>
						{/if}
					</div>
				</div>

				<!-- Save to collection row -->
				<div class="flex flex-wrap items-end gap-3 rounded-lg border border-slate-200 dark:border-slate-800 p-3 bg-slate-50/50 dark:bg-slate-900/50">
					<div class="flex items-center gap-2">
						<Label for="collection-name-input" class="whitespace-nowrap text-sm">Collection</Label>
						<Input
							id="collection-name-input"
							type="text"
							placeholder="Collection name"
							bind:value={collectionName}
							class="w-40 font-mono"
						/>
					</div>
					<div class="flex items-center gap-2">
						<Label for="request-name-input" class="whitespace-nowrap text-sm">Request name</Label>
						<Input
							id="request-name-input"
							type="text"
							placeholder="e.g. get-pikachu"
							bind:value={requestName}
							class="w-48 font-mono"
						/>
					</div>
					<Button
						variant="outline"
						onclick={saveCurrentRequest}
						class="text-sm"
					>
						Save Request
					</Button>
					<Button
						variant="outline"
						size="sm"
						onclick={importCollectionFromClipboard}
						class="text-xs"
						title="Import a collection JSON from the clipboard"
					>
						Import from clipboard
					</Button>
					<Button
						variant="outline"
						size="sm"
						onclick={() => openapiFileInput?.click()}
						class="text-xs"
						title="Import endpoints from an OpenAPI (YAML/JSON) spec into this collection"
					>
						Import OpenAPI
					</Button>
					<input
						type="file"
						accept=".json,.yaml,.yml"
						class="hidden"
						bind:this={openapiFileInput}
						onchange={importOpenAPIFromFile}
					/>
					{#if collectionError}
						<span class="text-xs text-red-600 dark:text-red-400">{collectionError}</span>
					{/if}
					{#if collectionMessage}
						<span class="text-xs text-green-600 dark:text-green-400">{collectionMessage}</span>
					{/if}
				</div>

				<!-- Request Configuration Tabs -->
				<Tabs value="params" class="w-full">
					<TabsList class="grid w-full grid-cols-5">
						<TabsTrigger value="params">Params</TabsTrigger>
						<TabsTrigger value="headers">Headers</TabsTrigger>
						<TabsTrigger value="auth">Auth</TabsTrigger>
						<TabsTrigger value="variables">Variables</TabsTrigger>
						<TabsTrigger value="body" disabled={selectedMethod === 'GET' || selectedMethod === 'HEAD'}>Body</TabsTrigger>
					</TabsList>

					<!-- Params Tab -->
					<TabsContent value="params" class="space-y-2">
						<p class="text-xs text-slate-500 dark:text-slate-400">
							Query parameters are merged into the URL automatically.
						</p>
						<div class="space-y-2">
							{#each queryParams as param, i}
								<div class="flex gap-2 items-center">
									<input
										type="checkbox"
										checked={param.enabled}
										onchange={(e) => updateQueryParam(i, 'enabled', e.currentTarget.checked)}
										class="h-4 w-4 shrink-0"
										title="Enable / disable this parameter"
									/>
									<Input
										placeholder="Parameter name"
										value={param.key}
										oninput={(e) => updateQueryParam(i, 'key', e.currentTarget.value)}
										class="flex-1"
									/>
									<Input
										placeholder="Parameter value"
										value={param.value}
										oninput={(e) => updateQueryParam(i, 'value', e.currentTarget.value)}
										class="flex-1"
									/>
									<Button
										variant="outline"
										size="icon"
										onclick={() => removeQueryParam(i)}
									>
										<Trash2 class="h-4 w-4" />
									</Button>
								</div>
							{/each}
						</div>
						<Button
							variant="outline"
							onclick={addQueryParam}
							class="w-full"
						>
							<Plus class="mr-2 h-4 w-4" />
							Add Parameter
						</Button>
					</TabsContent>

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

					<!-- Auth Tab -->
					<TabsContent value="auth" class="space-y-3">
						<div class="space-y-2">
							<Label for="auth-type" class="text-sm">Auth Type</Label>
							<Select type="single" bind:value={authType}>
								<SelectTrigger id="auth-type" class="w-full">
									<span>{authType === 'none' ? 'No Auth' : authType === 'basic' ? 'Basic Auth' : 'Bearer Token'}</span>
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="none">No Auth</SelectItem>
									<SelectItem value="basic">Basic Auth</SelectItem>
									<SelectItem value="bearer">Bearer Token</SelectItem>
								</SelectContent>
							</Select>
							{#if authType === 'basic'}
								<Input type="text" placeholder="Username" bind:value={authUsername} class="font-mono" />
								<Input type="password" placeholder="Password" bind:value={authPassword} class="font-mono" />
							{:else if authType === 'bearer'}
								<Input type="password" placeholder="Token" bind:value={authToken} class="font-mono" />
							{/if}
							{#if authType !== 'none'}
								<div class="rounded-md bg-slate-100 dark:bg-slate-800 px-3 py-2 font-mono text-xs text-slate-600 dark:text-slate-300 break-all">
									{authPreview()}
								</div>
							{/if}
						</div>
					</TabsContent>

					<!-- Variables Tab -->
					<TabsContent value="variables" class="space-y-2">
						<p class="text-xs text-slate-500 dark:text-slate-400">
							{'Variables are referenced with {{NAME}} syntax in the URL, headers, and body.'}
						</p>
						<div class="space-y-2">
							{#each variables as variable, i}
								<div class="flex gap-2">
									<Input
										placeholder="Variable name"
										bind:value={variable.key}
										onchange={(e) => updateVariable(i, 'key', e.currentTarget.value)}
										class="flex-1"
									/>
									<Input
										placeholder="Variable value"
										bind:value={variable.value}
										onchange={(e) => updateVariable(i, 'value', e.currentTarget.value)}
										class="flex-1"
									/>
									<Button
										variant="outline"
										size="icon"
										onclick={() => removeVariable(i)}
									>
										<Trash2 class="h-4 w-4" />
									</Button>
								</div>
							{/each}
						</div>
						<Button
							variant="outline"
							onclick={addVariable}
							class="w-full"
						>
							<Plus class="mr-2 h-4 w-4" />
							Add Variable
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

		<!-- History Section -->
		{#if history.length > 0}
			<Card class="shadow-lg border-slate-200 dark:border-slate-800">
				<CardHeader class="bg-gradient-to-r from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800">
					<div class="flex items-center justify-between">
						<CardTitle class="flex items-center gap-2 text-lg">
							<Clock class="h-4 w-4 text-blue-600" />
							History
						</CardTitle>
						<Button variant="outline" size="sm" onclick={clearHistory} class="text-xs">
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
									onclick={() => restoreRequest(entry)}
									title="Restore request"
								>
									<span class={`w-14 shrink-0 text-center text-xs font-bold py-0.5 rounded ${selectedMethod === entry.method ? 'ring-2 ring-blue-400' : ''}`}>
										<span class={entry.method === 'GET' ? 'text-green-600' : entry.method === 'POST' ? 'text-blue-600' : entry.method === 'PUT' ? 'text-yellow-600' : entry.method === 'DELETE' ? 'text-red-600' : 'text-purple-600'}>
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
		{/if}

		<!-- Collections Section -->
		{#if savedCollections.length > 0}
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
					{#each savedCollections as c}
						<div class="rounded-lg border border-slate-200 dark:border-slate-800 overflow-hidden">
							<div class="flex items-center justify-between px-3 py-2 bg-slate-100 dark:bg-slate-800/60">
								<span class="text-sm font-semibold text-slate-700 dark:text-slate-200">{c.name}</span>
								<div class="flex items-center gap-1">
									<Button variant="ghost" size="sm" class="text-xs" onclick={() => exportCollection(c)} title="Copy collection JSON to clipboard">
										Export
									</Button>
									<Button variant="ghost" size="sm" class="text-xs text-red-600 hover:text-red-700" onclick={() => removeCollection(c)} title="Delete collection">
										Delete
									</Button>
								</div>
							</div>
							<ul class="divide-y divide-slate-100 dark:divide-slate-800">
								{#each c.requests as req}
									<li class="flex items-center gap-2 px-3 py-1.5 hover:bg-slate-50 dark:hover:bg-slate-800/50">
										<button
											class="flex-1 flex items-center gap-2 text-left font-mono text-sm text-slate-700 dark:text-slate-300"
											onclick={() => loadSavedRequest(c, req)}
											title="Load request"
										>
											<span class="text-xs font-bold w-12 shrink-0 text-right">
												<span class={req.method === 'GET' ? 'text-green-600' : req.method === 'POST' ? 'text-blue-600' : req.method === 'PUT' ? 'text-yellow-600' : req.method === 'DELETE' ? 'text-red-600' : 'text-purple-600'}>
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
											onclick={() => removeSavedRequest(c, req)}
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
		{/if}

		<!-- Environments Section -->
		<Card class="shadow-lg border-slate-200 dark:border-slate-800">
				<CardHeader class="bg-gradient-to-r from-slate-50 to-slate-100 dark:from-slate-900 dark:to-slate-800">
					<CardTitle class="flex items-center gap-2 text-lg">
						<svg class="h-4 w-4 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3.75 21h16.5M4.5 3h15M5.25 3v18m13.5-18v18M9 6.75h1.5m-1.5 3h1.5m-1.5 3h1.5m3-6H15m-1.5 3H15m-1.5 3H15M9 21v-3.375c0-.621.504-1.125 1.125-1.125h3.75c.621 0 1.125.504 1.125 1.125V21" />
						</svg>
						Environments
					</CardTitle>
					<CardDescription>
						{'Environment variables are merged into {{VAR}} expansion when a request is sent.'}
					</CardDescription>
				</CardHeader>
				<CardContent class="pt-4 space-y-3">
					<!-- Editor -->
					<div class="rounded-lg border border-slate-200 dark:border-slate-800 p-3 space-y-2">
						<div class="flex items-center gap-2">
							<Label for="env-name-input" class="whitespace-nowrap text-sm">Name</Label>
							<Input
								id="env-name-input"
								type="text"
								placeholder="e.g. dev"
								bind:value={envName}
								class="w-40 font-mono"
							/>
							<Button variant="outline" size="sm" class="text-xs" onclick={saveEnvironment} title="Save or update this environment">
								Save Environment
							</Button>
							{#if envName}
								<Button variant="outline" size="sm" class="text-xs text-red-600 hover:text-red-700" onclick={() => removeEnvironment(envName.trim())} title="Delete this environment">
									Delete
								</Button>
							{/if}
						</div>
						<div class="space-y-2">
							{#each envVars as v, i}
								<div class="flex gap-2 items-center">
									<Input
										placeholder="Variable name"
										value={v.key}
										oninput={(e) => updateEnvVar(i, 'key', e.currentTarget.value)}
										class="flex-1 font-mono"
									/>
									<Input
										placeholder={'Value ({{NAME}} is replaced with this)'}
										value={v.value}
										oninput={(e) => updateEnvVar(i, 'value', e.currentTarget.value)}
										class="flex-1 font-mono"
									/>
									<Button variant="outline" size="icon" onclick={() => removeEnvVar(i)} title="Remove variable">
										<Trash2 class="h-4 w-4" />
									</Button>
								</div>
							{/each}
						</div>
						<Button variant="ghost" size="sm" class="text-xs" onclick={addEnvVar}>
							+ Add variable
						</Button>
						{#if envError}
							<p class="text-xs text-red-600 dark:text-red-400">{envError}</p>
						{/if}
						{#if envMessage}
							<p class="text-xs text-green-600 dark:text-green-400">{envMessage}</p>
						{/if}
					</div>

					<!-- List -->
					{#if environments.length > 0}
						<ul class="divide-y divide-slate-200 dark:divide-slate-800 rounded-lg border border-slate-200 dark:border-slate-800">
							{#each environments as name}
								<li class="flex items-center gap-2 px-3 py-2">
									<span class="font-mono text-sm text-slate-700 dark:text-slate-200 flex-1">
										{name}
										{#if name === environmentName}
											<Badge variant="outline" class="ml-2 text-[10px] bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400">active</Badge>
										{/if}
									</span>
									<Button variant="ghost" size="sm" class="text-xs" onclick={() => loadEnvironmentForEdit(name)} title="Edit environment">
										Edit
									</Button>
									<Button variant="ghost" size="sm" class="text-xs text-emerald-600 hover:text-emerald-700" onclick={() => { environmentName = name; }} title="Use this environment for requests">
										Use
									</Button>
								</li>
							{/each}
						</ul>
					{/if}
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
						
						<TabsContent value="body" class="mt-4">
							<div class="rounded-lg border border-slate-700 bg-slate-900 shadow-xl overflow-hidden">
								<div class="flex items-center justify-between px-4 py-2 bg-slate-800 border-b border-slate-700">
									<div class="flex items-center gap-2">
										<div class="w-3 h-3 rounded-full bg-red-500"></div>
										<div class="w-3 h-3 rounded-full bg-yellow-500"></div>
										<div class="w-3 h-3 rounded-full bg-green-500"></div>
									</div>
									<span class="text-xs text-slate-400 font-mono">response.json</span>
									<button onclick={copyResponse} class="text-slate-400 hover:text-white transition-colors p-1" title="Copy to clipboard">
										<Copy class="h-4 w-4" />
									</button>
								</div>
								<ScrollArea class="h-[600px] w-full bg-gray-900" orientation="both">
									<pre class="block w-full min-w-max p-4 text-left text-[12px] font-mono leading-[1.4] text-gray-200 whitespace-pre">{@html formatJson(response)}</pre>
								</ScrollArea>
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
	</div>
</main>

<style>
	:global(body) {
		margin: 0;
		padding: 0;
	}
</style>