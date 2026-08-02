<script lang="ts">
	import './app.css';
	import { onMount } from 'svelte';
	import { Card, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import RequestEditor, { type RequestEditorHandle } from '$lib/components/RequestEditor.svelte';
	import HistoryPanel from '$lib/components/HistoryPanel.svelte';
	import CollectionsPanel from '$lib/components/CollectionsPanel.svelte';
	import EnvironmentPanel from '$lib/components/EnvironmentPanel.svelte';
	import ResponseViewer from '$lib/components/ResponseViewer.svelte';
	import { addHistoryEntry, clearHistory, loadHistory, saveHistory, type HistoryEntry } from '$lib/history';
	import { fromSavedRequest } from '$lib/request-draft';
	import { extractErrorMessage } from '$lib/format';
	import {
		deleteCollection,
		deleteRequest,
		exportCollection as exportCollectionToClipboard,
		getCollection,
		listCollections,
		listEnvironments,
		type AssertionResult,
		type Collection,
		type ResponseData,
		type SavedRequest,
	} from '$lib/app-client';

	// ---- Request editor handle (snapshot/apply/send) ------------------------
	let requestEditor = $state<RequestEditorHandle>();

	// ---- Response state -----------------------------------------------------
	let response = $state('');
	let responseStatus = $state(0);
	let responseHeaders = $state('');
	let responseTime = $state(0);
	let isLoading = $state(false);
	let error = $state('');
	let assertionResults = $state<AssertionResult[]>([]);

	// ---- Shared state -------------------------------------------------------
	let environmentName = $state(''); // selected environment ('' = none)
	let environments = $state<Array<string>>([]);
	let savedCollections = $state<Collection[]>([]);
	let collectionError = $state('');
	let collectionMessage = $state('');
	let activeMethod = $state('GET');

	// ---- History (persisted in localStorage) --------------------------------
	let history = $state<HistoryEntry[]>(loadHistory(localStorage));
	const historyUrls = $derived([...new Set(history.map((h) => h.url))]);

	function saveToHistory() {
		const editor = requestEditor;
		if (!editor) return;
		history = addHistoryEntry(history, editor.snapshotDraft());
		saveHistory(history, localStorage);
	}

	function restoreRequest(entry: HistoryEntry) {
		requestEditor?.applyDraft(entry);
		response = '';
		responseStatus = 0;
		responseHeaders = '';
		error = '';
	}

	function onClearHistory() {
		history = [];
		clearHistory(localStorage);
	}

	// ---- Response handling (from RequestEditor) -----------------------------
	function onResponse(result: ResponseData, results: AssertionResult[]) {
		responseStatus = result.statusCode;
		responseTime = result.elapsedMs;
		response = result.body || '(empty response body)';
		responseHeaders = Object.entries(result.headers)
			.map(([k, v]) => `${k}: ${v}`)
			.join('\n');
		assertionResults = results;
		saveToHistory();
	}

	// ---- Keyboard shortcuts -------------------------------------------------
	function handleKeydown(e: KeyboardEvent) {
		if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
			e.preventDefault();
			if (!isLoading) requestEditor?.send();
		}
	}

	onMount(() => {
		window.addEventListener('keydown', handleKeydown);
		refreshCollections();
		refreshEnvironments();
		return () => window.removeEventListener('keydown', handleKeydown);
	});

	// ---- Environments -------------------------------------------------------
	async function refreshEnvironments() {
		try {
			const names = await listEnvironments();
			environments = names;
			if (environmentName && !names.includes(environmentName)) {
				environmentName = '';
			}
		} catch {
			// environments just won't load
		}
	}

	// ---- Collections --------------------------------------------------------
	async function refreshCollections() {
		try {
			const names = await listCollections();
			const loaded = [];
			for (const name of names) {
				try {
					loaded.push(await getCollection(name));
				} catch {
					// skip unreadable collection
				}
			}
			savedCollections = loaded;
		} catch (e) {
			collectionError = extractErrorMessage(e);
		}
	}

	function loadSavedRequest(c: Collection, req: SavedRequest) {
		requestEditor?.applyDraft(fromSavedRequest(req));
		response = '';
		responseStatus = 0;
		responseHeaders = '';
		error = '';
		collectionError = '';
		collectionMessage = '';
	}

	async function removeSavedRequest(c: Collection, req: SavedRequest) {
		collectionError = '';
		collectionMessage = '';
		try {
			await deleteRequest(c.name, req.name);
			if (c.requests.length === 1) {
				await refreshCollections();
			} else {
				const fresh = await getCollection(c.name);
				const idx = savedCollections.findIndex((s) => s.name === c.name);
				if (idx >= 0) savedCollections[idx] = fresh;
				savedCollections = savedCollections;
			}
		} catch (e) {
			collectionError = extractErrorMessage(e);
		}
	}

	async function removeCollection(c: Collection) {
		collectionError = '';
		collectionMessage = '';
		try {
			await deleteCollection(c.name);
			await refreshCollections();
		} catch (e) {
			collectionError = extractErrorMessage(e);
		}
	}

	async function exportCollection(c: Collection) {
		collectionError = '';
		collectionMessage = '';
		try {
			const json = await exportCollectionToClipboard(c.name);
			await navigator.clipboard.writeText(json);
			collectionMessage = `Exported "${c.name}" to clipboard.`;
		} catch (e) {
			collectionError = extractErrorMessage(e);
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
		<RequestEditor
			bind:this={requestEditor}
			{historyUrls}
			{environments}
			{environmentName}
			{isLoading}
			{collectionError}
			{collectionMessage}
			onresponse={onResponse}
			onerror={(m) => (error = m)}
			onrequeststate={(l) => (isLoading = l)}
			oncollectionchanged={refreshCollections}
			oncollectionstatus={(e, m) => {
				collectionError = e;
				collectionMessage = m;
			}}
			onenvironmentchange={(n) => (environmentName = n)}
			onmethodchange={(m) => (activeMethod = m)}
		/>

		<!-- History Section -->
		{#if history.length > 0}
			<HistoryPanel {history} {activeMethod} onrestore={restoreRequest} onclear={onClearHistory} />
		{/if}

		<!-- Collections Section -->
		{#if savedCollections.length > 0}
			<CollectionsPanel
				collections={savedCollections}
				onload={loadSavedRequest}
				ondeleterequest={removeSavedRequest}
				ondeletecollection={removeCollection}
				onexport={exportCollection}
			/>
		{/if}

		<!-- Environments Section -->
		<EnvironmentPanel
			{environments}
			{environmentName}
			onuse={(n) => (environmentName = n)}
			onchanged={refreshEnvironments}
		/>

		<!-- Response Section -->
		<ResponseViewer {response} {responseStatus} {responseTime} {responseHeaders} {error} {assertionResults} />
	</div>
</main>

<style>
	:global(body) {
		margin: 0;
		padding: 0;
	}
</style>
