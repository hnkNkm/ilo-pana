<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Label } from '$lib/components/ui/label';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Trash2 } from '@lucide/svelte/icons';
	import { extractErrorMessage } from '$lib/format';
	import { deleteEnvironment, getEnvironment, saveEnvironment } from '$lib/app-client';

	let {
		environments,
		environmentName,
		onuse,
		onchanged,
	} = $props<{
		environments: string[];
		environmentName: string;
		onuse?: (name: string) => void;
		onchanged?: () => void;
	}>();

	let envName = $state('');
	let envVars = $state<Array<{ id: string; key: string; value: string }>>([]);
	let envError = $state('');
	let envMessage = $state('');

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

	async function loadEnvironmentForEdit(name: string) {
		envError = '';
		envMessage = '';
		try {
			const env = await getEnvironment(name);
			envName = env.name;
			envVars = env.variables
				? ensureRowIds(Object.entries(env.variables).map(([key, value]) => ({ key, value })))
				: [];
		} catch (e) {
			envError = extractErrorMessage(e);
		}
	}

	async function saveEnv() {
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
			await saveEnvironment(name, varsMap);
			envMessage = `Saved environment "${name}".`;
			onchanged?.();
		} catch (e) {
			envError = extractErrorMessage(e);
		}
	}

	async function removeEnv(name: string) {
		envError = '';
		envMessage = '';
		try {
			await deleteEnvironment(name);
			if (envName === name) {
				envName = '';
				envVars = [];
			}
			onchanged?.();
		} catch (e) {
			envError = extractErrorMessage(e);
		}
	}

	function addEnvVar() {
		envVars = [...envVars, { id: newRowId(), key: '', value: '' }];
	}

	function removeEnvVar(id: string) {
		envVars = envVars.filter((v) => v.id !== id);
	}

	function updateEnvVar(id: string, field: 'key' | 'value', value: string) {
		const v = envVars.find((row) => row.id === id);
		if (!v) return;
		v[field] = value;
		envVars = envVars;
	}
</script>

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
				<Button variant="outline" size="sm" class="text-xs" onclick={saveEnv} title="Save or update this environment">
					Save Environment
				</Button>
				{#if envName}
					<Button variant="outline" size="sm" class="text-xs text-red-600 hover:text-red-700" onclick={() => removeEnv(envName.trim())} title="Delete this environment">
						Delete
					</Button>
				{/if}
			</div>
			<div class="space-y-2">
				{#each envVars as v (v.id)}
					<div class="flex gap-2 items-center">
						<Input
							placeholder="Variable name"
							value={v.key}
							oninput={(e) => updateEnvVar(v.id, 'key', e.currentTarget.value)}
							class="flex-1 font-mono"
						/>
						<Input
							placeholder={'Value ({{NAME}} is replaced with this)'}
							value={v.value}
							oninput={(e) => updateEnvVar(v.id, 'value', e.currentTarget.value)}
							class="flex-1 font-mono"
						/>
						<Button variant="outline" size="icon" onclick={() => removeEnvVar(v.id)} title="Remove variable">
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
						<Button variant="ghost" size="sm" class="text-xs text-emerald-600 hover:text-emerald-700" onclick={() => onuse?.(name)} title="Use this environment for requests">
							Use
						</Button>
					</li>
				{/each}
			</ul>
		{/if}
	</CardContent>
</Card>
