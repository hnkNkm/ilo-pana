<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Select, SelectContent, SelectItem, SelectTrigger } from '$lib/components/ui/select';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import { Badge } from '$lib/components/ui/badge';
	import { Plus, Trash2, Send, Copy, X } from '@lucide/svelte/icons';
	import { buildUrlFromParams, parseQueryParams as parseQueryString } from '$lib/url-utils';
	import { extractErrorMessage, methodColorClass } from '$lib/format';
	import { DRAFT_VERSION, fromSavedRequest, toSavedRequest, type RequestDraft } from '$lib/request-draft';
	import {
		evaluateAssertions,
		generateCurl,
		importCollection,
		importCurl,
		importOpenAPI,
		newAssertionRule,
		newSavedRequest,
		saveRequest,
		sendRequest as sendRequestToBackend,
		type AssertionResult,
		type ResponseData,
	} from '$lib/app-client';

	export type RequestEditorHandle = {
		snapshotDraft: () => RequestDraft;
		applyDraft: (draft: RequestDraft) => void;
		send: () => Promise<void>;
	};

	// ---- Props (shared state owned by App) ---------------------------------
	let {
		historyUrls,
		environments,
		environmentName,
		isLoading,
		collectionError,
		collectionMessage,
		onresponse,
		onerror,
		onrequeststate,
		oncollectionchanged,
		oncollectionstatus,
		onenvironmentchange,
		onmethodchange,
	} = $props<{
		historyUrls: string[];
		environments: string[];
		environmentName: string;
		isLoading: boolean;
		collectionError: string;
		collectionMessage: string;
		onresponse?: (result: ResponseData, assertionResults: AssertionResult[]) => void;
		onerror?: (message: string) => void;
		onrequeststate?: (loading: boolean) => void;
		oncollectionchanged?: () => void;
		oncollectionstatus?: (error: string, message: string) => void;
		onenvironmentchange?: (name: string) => void;
		onmethodchange?: (method: string) => void;
	}>();

	// Notify App so shared UI (e.g. the history highlight) stays in sync.
	$effect(() => onmethodchange?.(selectedMethod));

	// ---- Editor state ------------------------------------------------------
	let url = $state('https://pokeapi.co/api/v2/pokemon/pikachu');
	let selectedMethod = $state('GET');
	let requestBody = $state('');
	let bodyFormat = $state<'raw' | 'form-data' | 'urlencoded'>('raw');

	function newRowId(): string {
		return crypto.randomUUID();
	}

	function ensureRowIds<T extends object>(rows: T[]): Array<T & { id: string }> {
		for (const row of rows) {
			const r = row as T & { id?: string };
			if (!r.id) r.id = newRowId();
		}
		return rows as Array<T & { id: string }>;
	}

	interface FormFieldRow {
		id: string;
		key: string;
		value: string;
		type: 'text' | 'file';
		fileName: string;
		fileContent: number[];
		contentType: string;
	}
	let formFields = $state<FormFieldRow[]>([]);

	function addFormField() {
		formFields = [...formFields, { id: newRowId(), key: '', value: '', type: 'text', fileName: '', fileContent: [], contentType: '' }];
	}

	function removeFormField(id: string) {
		formFields = formFields.filter((f) => f.id !== id);
	}

	function updateFormField(id: string, field: 'key' | 'value' | 'type', value: string) {
		const f = formFields.find((row) => row.id === id);
		if (!f) return;
		if (field === 'type') f.type = value as 'text' | 'file';
		else f[field] = value;
		formFields = formFields;
	}

	function fileInputId(id: string) {
		return `form-file-input-${id}`;
	}

	function handleFormFileSelect(id: string, input: HTMLInputElement) {
		const file = input.files?.[0];
		if (!file) return;
		file.arrayBuffer().then((buf) => {
			const row = formFields.find((f) => f.id === id);
			if (!row) return;
			row.fileName = file.name;
			row.fileContent = Array.from(new Uint8Array(buf));
			row.contentType = file.type || 'application/octet-stream';
			formFields = formFields;
		});
	}

	let headers = $state<Array<{ id: string; key: string; value: string }>>([
		{ id: newRowId(), key: 'Content-Type', value: 'application/json' },
	]);
	let queryParams = $state<Array<{ id: string; key: string; value: string; enabled: boolean }>>([]);
	let variables = $state<Array<{ id: string; key: string; value: string }>>([]);
	let timeoutSec = $state(30);
	let sessionName = $state('');
	let sessionNew = $state(false);
	let authType = $state('none');
	let authUsername = $state('');
	let authPassword = $state('');
	let authToken = $state('');

	// Assertions (response validation rules)
	type AssertionRuleRow = { id: string; name: string; kind: string; target?: string; expected?: string };
	let assertionRules = $state<AssertionRuleRow[]>([]);
	let assertionError = $state('');

	function addAssertionRule() {
		assertionRules = [...assertionRules, { ...newAssertionRule({ kind: 'status_equals', target: '200' }), id: newRowId() }];
	}

	function removeAssertionRule(id: string) {
		assertionRules = assertionRules.filter((r) => r.id !== id);
	}

	function updateAssertionRule(id: string, field: 'name' | 'kind' | 'target' | 'expected', value: string) {
		const rule = assertionRules.find((r) => r.id === id);
		if (!rule) return;
		rule[field] = value;
		assertionRules = assertionRules;
	}

	function assertionKindLabel(kind: string): string {
		const labels: Record<string, string> = {
			status_equals: 'Status equals',
			status_range: 'Status in range (2xx)',
			body_contains: 'Body contains',
			body_not_contains: 'Body not contains',
			json_path_exists: 'JSON path exists',
			json_path_equals: 'JSON path equals',
			json_path_contains: 'JSON path contains',
		};
		return labels[kind] ?? kind;
	}

	function ruleNeedsExpected(kind: string): boolean {
		return kind === 'json_path_equals' || kind === 'json_path_contains';
	}

	function assertionTargetPlaceholder(kind: string): string {
		if (kind === 'status_equals' || kind === 'status_range') return 'e.g. 200, 2xx, 200-299';
		if (kind.startsWith('json_path')) return 'e.g. data.items[0].name';
		return 'e.g. hello world';
	}

	// ---- Draft snapshot / apply (exported for App.svelte) --------------------
	export function snapshotDraft(): RequestDraft {
		return {
			version: DRAFT_VERSION,
			method: selectedMethod,
			url,
			headers: headers.filter((h) => h.key?.trim()).map((h) => ({ key: h.key, value: h.value })),
			body: requestBody,
			bodyFormat,
			formFields: formFields.map((f) => ({
				key: f.key,
				value: f.value,
				type: f.type,
				fileName: f.fileName,
				contentType: f.contentType,
				hasFile: f.type === 'file',
			})),
			variables: variables.filter((v) => v.key?.trim()).map((v) => ({ key: v.key, value: v.value })),
			timeoutSec,
			sessionName,
			sessionNew,
			environment: environmentName,
			assertions: assertionRules.map((r) => ({
				name: r.name,
				kind: r.kind,
				target: r.target ?? '',
				expected: r.expected ?? '',
			})),
		};
	}

	export function applyDraft(draft: RequestDraft) {
		selectedMethod = draft.method;
		url = draft.url;
		parseQueryParams();
		requestBody = draft.body;
		bodyFormat = draft.bodyFormat;
		formFields = ensureRowIds(
			draft.formFields.map((f) => ({
				key: f.key,
				value: f.value,
				type: f.type,
				fileName: f.fileName,
				fileContent: [],
				contentType: f.contentType,
			}))
		);
		headers = ensureRowIds(
			draft.headers.length
				? draft.headers.map((h) => ({ key: h.key, value: h.value }))
				: [{ key: 'Content-Type', value: 'application/json' }]
		);
		variables = ensureRowIds(draft.variables.map((v) => ({ key: v.key, value: v.value })));
		timeoutSec = draft.timeoutSec;
		sessionName = draft.sessionName;
		sessionNew = draft.sessionNew;
		onenvironmentchange?.(draft.environment);
		assertionRules = draft.assertions.map((a) => ({ ...newAssertionRule({ kind: a.kind, target: a.target, expected: a.expected }), id: newRowId() }));
		assertionError = '';
	}

	// ---- Request execution --------------------------------------------------
	const httpMethods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'];

	function buildHeadersMap(): Record<string, string> {
		const headersMap: Record<string, string> = {};
		for (const h of headers) {
			if (h.key?.trim()) {
				headersMap[h.key.trim()] = h.value ?? '';
			}
		}

		// Auth header (overrides a manually-entered Authorization header)
		if (authType === 'basic' && authUsername.trim()) {
			headersMap['Authorization'] = `Basic ${btoa(unescape(encodeURIComponent(`${authUsername}:${authPassword}`)))}`;
		} else if (authType === 'bearer' && authToken.trim()) {
			headersMap['Authorization'] = `Bearer ${authToken}`;
		}
		return headersMap;
	}

	export async function send() {
		if (isLoading || !url.trim()) return;
		onrequeststate?.(true);
		onerror?.('');

		try {
			const result = await sendRequestToBackend({
				method: selectedMethod,
				url,
				body: requestBody,
				bodyFormat,
				formFields,
				headers: buildHeadersMap(),
				timeoutSec,
				sessionName,
				sessionNew,
				variables: Object.fromEntries(
					variables.filter((v) => v.key?.trim()).map((v) => [v.key.trim(), v.value ?? ''])
				),
				environment: environmentName,
			});

			// Run configured assertions against the response
			assertionError = '';
			let results: AssertionResult[] = [];
			if (assertionRules.length) {
				try {
					results = await evaluateAssertions(result, assertionRules);
				} catch (e) {
					results = [];
					assertionError = extractErrorMessage(e);
				}
			}
			onresponse?.(result, results);
		} catch (err) {
			onerror?.(extractErrorMessage(err));
		} finally {
			onrequeststate?.(false);
		}
	}

	// ---- Header rows --------------------------------------------------------
	function addHeader() {
		headers = [...headers, { id: newRowId(), key: '', value: '' }];
	}

	function removeHeader(id: string) {
		headers = headers.filter((h) => h.id !== id);
	}

	function updateHeader(id: string, field: 'key' | 'value', value: string) {
		const h = headers.find((row) => row.id === id);
		if (!h) return;
		h[field] = value;
		headers = headers;
	}

	// ---- Query parameters ---------------------------------------------------
	function parseQueryParams() {
		queryParams = ensureRowIds(parseQueryString(url));
	}

	function rebuildUrlFromParams() {
		url = buildUrlFromParams(url.split('?')[0], queryParams);
	}

	function addQueryParam() {
		queryParams = [...queryParams, { id: newRowId(), key: '', value: '', enabled: true }];
	}

	function removeQueryParam(id: string) {
		queryParams = queryParams.filter((p) => p.id !== id);
		rebuildUrlFromParams();
	}

	function updateQueryParam(id: string, field: 'key' | 'value' | 'enabled', value: string | boolean) {
		const p = queryParams.find((row) => row.id === id);
		if (!p) return;
		if (field === 'enabled') {
			p.enabled = value as boolean;
		} else {
			p[field] = value as string;
		}
		queryParams = queryParams;
		rebuildUrlFromParams();
	}

	// ---- Variables rows -----------------------------------------------------
	function addVariable() {
		variables = [...variables, { id: newRowId(), key: '', value: '' }];
	}

	function removeVariable(id: string) {
		variables = variables.filter((v) => v.id !== id);
	}

	function updateVariable(id: string, field: 'key' | 'value', value: string) {
		const v = variables.find((row) => row.id === id);
		if (!v) return;
		v[field] = value;
		variables = variables;
	}

	// ---- Auth ---------------------------------------------------------------
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

	// ---- Collection save / imports ------------------------------------------
	let collectionName = $state('');
	let requestName = $state('');
	let openapiFileInput: HTMLInputElement | undefined = $state();

	async function saveCurrentRequest() {
		oncollectionstatus?.('', '');
		if (!collectionName.trim()) {
			oncollectionstatus?.('Enter a collection name to save this request.', '');
			return;
		}
		if (!requestName.trim()) {
			oncollectionstatus?.('Enter a request name to save this request.', '');
			return;
		}

		const saved = toSavedRequest(requestName.trim(), snapshotDraft());

		try {
			await saveRequest(collectionName.trim(), newSavedRequest(saved));
			oncollectionstatus?.('', `Saved "${requestName.trim()}" to "${collectionName.trim()}".`);
			oncollectionchanged?.();
		} catch (e) {
			oncollectionstatus?.(extractErrorMessage(e), '');
		}
	}

	async function importCollectionFromClipboard() {
		oncollectionstatus?.('', '');
		try {
			const json = await navigator.clipboard.readText();
			await importCollection(json);
			oncollectionstatus?.('', 'Collection imported from clipboard.');
			oncollectionchanged?.();
		} catch (e) {
			oncollectionstatus?.(extractErrorMessage(e), '');
		}
	}

	async function importCurlFromClipboard() {
		oncollectionstatus?.('', '');
		try {
			const cmd = await navigator.clipboard.readText();
			const req = await importCurl(cmd);
			applyDraft(fromSavedRequest(req));
			oncollectionstatus?.('', `Imported ${req.method} ${req.url} from cURL.`);
		} catch (e) {
			oncollectionstatus?.(extractErrorMessage(e), '');
		}
	}

	async function importOpenAPIFromFile() {
		oncollectionstatus?.('', '');
		const file = openapiFileInput?.files?.[0];
		if (!file) return;
		try {
			const content = await file.text();
			const target = collectionName.trim();
			const n = await importOpenAPI(content, target);
			oncollectionstatus?.(
				'',
				target ? `Imported ${n} endpoints into "${target}".` : `Imported ${n} endpoints.`
			);
			oncollectionchanged?.();
		} catch (e) {
			oncollectionstatus?.(extractErrorMessage(e), '');
		} finally {
			if (openapiFileInput) openapiFileInput.value = '';
		}
	}

	// ---- cURL ---------------------------------------------------------------
	async function copyAsCurl() {
		onerror?.('');
		try {
			const cmd = await generateCurl(selectedMethod, url, buildHeadersMap(), requestBody);
			await navigator.clipboard.writeText(cmd);
			onerror?.('');
		} catch (e) {
			onerror?.(extractErrorMessage(e));
		}
	}

	// Exported for App.svelte via component exports (bind:this)
	// - snapshotDraft / applyDraft: history restore, collection load, imports
	// - send: keyboard shortcut (Cmd/Ctrl+Enter)
</script>

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
					<span class={methodColorClass(selectedMethod)}>
						{selectedMethod || 'GET'}
					</span>
				</SelectTrigger>
				<SelectContent>
					{#each httpMethods as m}
						<SelectItem value={m} class="font-medium">
							<span class={methodColorClass(m)}>
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
				onclick={send}
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
			<Button
				variant="outline"
				onclick={copyAsCurl}
				disabled={!url}
				class="text-sm"
				title="Copy the current request as a curl command"
			>
				<Copy class="mr-2 h-4 w-4" />
				Copy as cURL
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
				<Select type="single" value={environmentName} onValueChange={(v) => onenvironmentchange?.(typeof v === 'string' ? v : '')}>
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
				onclick={importCurlFromClipboard}
				class="text-xs"
				title="Parse a curl command from the clipboard into the request"
			>
				Import cURL
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
		<TabsList class="grid w-full grid-cols-6">
			<TabsTrigger value="params">Params</TabsTrigger>
			<TabsTrigger value="headers">Headers</TabsTrigger>
			<TabsTrigger value="auth">Auth</TabsTrigger>
			<TabsTrigger value="variables">Variables</TabsTrigger>
			<TabsTrigger value="assertions">Assertions</TabsTrigger>
			<TabsTrigger value="body" disabled={selectedMethod === 'GET' || selectedMethod === 'HEAD'}>Body</TabsTrigger>
		</TabsList>

			<!-- Params Tab -->
			<TabsContent value="params" class="space-y-2">
				<p class="text-xs text-slate-500 dark:text-slate-400">
					Query parameters are merged into the URL automatically.
				</p>
				<div class="space-y-2">
					{#each queryParams as param (param.id)}
						<div class="flex gap-2 items-center">
							<input
								type="checkbox"
								checked={param.enabled}
								onchange={(e) => updateQueryParam(param.id, 'enabled', e.currentTarget.checked)}
								class="h-4 w-4 shrink-0"
								title="Enable / disable this parameter"
							/>
							<Input
								placeholder="Parameter name"
								value={param.key}
								oninput={(e) => updateQueryParam(param.id, 'key', e.currentTarget.value)}
								class="flex-1"
							/>
							<Input
								placeholder="Parameter value"
								value={param.value}
								oninput={(e) => updateQueryParam(param.id, 'value', e.currentTarget.value)}
								class="flex-1"
							/>
							<Button
								variant="outline"
								size="icon"
								onclick={() => removeQueryParam(param.id)}
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
					{#each headers as header (header.id)}
						<div class="flex gap-2">
							<Input
								placeholder="Header name"
								bind:value={header.key}
								onchange={(e) => updateHeader(header.id, 'key', e.currentTarget.value)}
								class="flex-1"
							/>
							<Input
								placeholder="Header value"
								bind:value={header.value}
								onchange={(e) => updateHeader(header.id, 'value', e.currentTarget.value)}
								class="flex-1"
							/>
							<Button
								variant="outline"
								size="icon"
								onclick={() => removeHeader(header.id)}
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
					{#each variables as variable (variable.id)}
						<div class="flex gap-2">
							<Input
								placeholder="Variable name"
								bind:value={variable.key}
								onchange={(e) => updateVariable(variable.id, 'key', e.currentTarget.value)}
								class="flex-1"
							/>
							<Input
								placeholder="Variable value"
								bind:value={variable.value}
								onchange={(e) => updateVariable(variable.id, 'value', e.currentTarget.value)}
								class="flex-1"
							/>
							<Button
								variant="outline"
								size="icon"
								onclick={() => removeVariable(variable.id)}
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

			<!-- Assertions Tab -->
			<TabsContent value="assertions" class="space-y-2">
				<p class="text-xs text-slate-500 dark:text-slate-400">
					Rules run against the response after each send. JSON paths use dot notation, e.g. data.items[0].name.
				</p>
				{#if assertionError}
					<p class="text-xs text-red-600 dark:text-red-400">{assertionError}</p>
				{/if}
				<div class="space-y-2">
					{#each assertionRules as rule (rule.id)}
						<div class="flex flex-wrap gap-2 items-center">
							<Select type="single" bind:value={rule.kind}>
								<SelectTrigger class="w-44 text-sm">
									<span>{assertionKindLabel(rule.kind)}</span>
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="status_equals">Status equals</SelectItem>
									<SelectItem value="status_range">Status in range (2xx)</SelectItem>
									<SelectItem value="body_contains">Body contains</SelectItem>
									<SelectItem value="body_not_contains">Body not contains</SelectItem>
									<SelectItem value="json_path_exists">JSON path exists</SelectItem>
									<SelectItem value="json_path_equals">JSON path equals</SelectItem>
									<SelectItem value="json_path_contains">JSON path contains</SelectItem>
								</SelectContent>
							</Select>
							<Input
								placeholder="Name (optional)"
								value={rule.name}
								oninput={(e) => updateAssertionRule(rule.id, 'name', e.currentTarget.value)}
								class="w-32 text-sm"
							/>
							<Input
								placeholder={assertionTargetPlaceholder(rule.kind)}
								value={rule.target}
								oninput={(e) => updateAssertionRule(rule.id, 'target', e.currentTarget.value)}
								class="flex-1 min-w-40 text-sm font-mono"
							/>
							{#if ruleNeedsExpected(rule.kind)}
								<Input
									placeholder="Expected value"
									value={rule.expected}
									oninput={(e) => updateAssertionRule(rule.id, 'expected', e.currentTarget.value)}
									class="flex-1 min-w-40 text-sm font-mono"
								/>
							{/if}
							<Button
								variant="outline"
								size="icon"
								onclick={() => removeAssertionRule(rule.id)}
							>
								<Trash2 class="h-4 w-4" />
							</Button>
						</div>
					{/each}
				</div>
				<Button
					variant="outline"
					onclick={addAssertionRule}
					class="w-full"
				>
					<Plus class="mr-2 h-4 w-4" />
					Add Assertion
				</Button>
			</TabsContent>

			<!-- Body Tab -->
			<TabsContent value="body">
				<div class="space-y-2">
					<div class="flex flex-wrap gap-2">
						<div class="w-36">
							<Select type="single" bind:value={bodyFormat}>
								<SelectTrigger>
									<span>{bodyFormat === 'raw' ? 'Raw' : bodyFormat === 'form-data' ? 'Form-Data' : 'URL Encoded'}</span>
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="raw">Raw</SelectItem>
									<SelectItem value="form-data">Form-Data</SelectItem>
									<SelectItem value="urlencoded">URL Encoded</SelectItem>
								</SelectContent>
							</Select>
						</div>
					</div>
					{#if bodyFormat === 'raw'}
						<Textarea
							placeholder="Request body (JSON, XML, etc.)"
							bind:value={requestBody}
							class="min-h-[200px] font-mono text-sm"
						/>
					{:else}
						<div class="space-y-2">
							{#each formFields as f (f.id)}
								<div class="flex flex-wrap gap-2 items-center">
									<Input
										placeholder="Field name"
										value={f.key}
										oninput={(e) => updateFormField(f.id, 'key', e.currentTarget.value)}
										class="w-40 font-mono text-sm"
									/>
									{#if bodyFormat === 'form-data'}
										<div class="w-24">
											<Select type="single" bind:value={f.type}>
												<SelectTrigger>
													<span>{f.type === 'file' ? 'File' : 'Text'}</span>
												</SelectTrigger>
												<SelectContent>
													<SelectItem value="text">Text</SelectItem>
													<SelectItem value="file">File</SelectItem>
												</SelectContent>
											</Select>
										</div>
									{/if}
									{#if bodyFormat === 'urlencoded' || f.type === 'text'}
										<Input
											placeholder="Field value"
											value={f.value}
											oninput={(e) => updateFormField(f.id, 'value', e.currentTarget.value)}
											class="flex-1 font-mono text-sm"
										/>
									{:else}
										<div class="flex items-center gap-2 flex-1">
											<input
												type="file"
												id={fileInputId(f.id)}
												class="hidden"
												onchange={(e) => handleFormFileSelect(f.id, e.currentTarget)}
											/>
											<Button
												variant="outline"
												size="sm"
												onclick={() => document.getElementById(fileInputId(f.id))?.click()}
											>
												Choose file
											</Button>
											<span class="text-xs font-mono text-slate-400 truncate">
												{f.fileName || 'No file chosen'}
											</span>
										</div>
									{/if}
									<Button variant="ghost" size="icon" onclick={() => removeFormField(f.id)} aria-label="Remove field">
										<X class="h-4 w-4" />
									</Button>
								</div>
							{/each}
							<Button variant="outline" size="sm" onclick={addFormField}>
								<Plus class="h-4 w-4" />
								Add field
							</Button>
						</div>
					{/if}
				</div>
			</TabsContent>
		</Tabs>
	</CardContent>
</Card>
