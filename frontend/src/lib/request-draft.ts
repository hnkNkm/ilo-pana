// Request draft: a versioned, serializable snapshot of the request editor
// state. History entries, saved collections and imported requests all
// serialize/restore through this module so no editor state is lost.
//
// Secrets policy:
//   - History (automatic, localStorage): sensitive header values
//     (Authorization, Cookie, Set-Cookie, X-API-Key, X-Auth-Token) are
//     masked before serialization. Auth credentials (basic/bearer fields)
//     and multipart file contents are never persisted.
//   - Collections (explicit user action, saved to disk): headers are kept
//     as-is because saving them is intentional; auth and file contents are
//     still excluded.

export const DRAFT_VERSION = 1;

export type DraftBodyFormat = 'raw' | 'form-data' | 'urlencoded';

export interface DraftHeader {
	key: string;
	value: string;
}

export interface DraftVariable {
	key: string;
	value: string;
}

export interface DraftFormField {
	key: string;
	value: string;
	type: 'text' | 'file';
	fileName: string;
	contentType: string;
	/** True when a file was attached. The content itself is never persisted. */
	hasFile: boolean;
}

export interface DraftAssertion {
	name: string;
	kind: string;
	target: string;
	expected: string;
}

export interface RequestDraft {
	version: typeof DRAFT_VERSION;
	method: string;
	url: string;
	headers: DraftHeader[];
	body: string;
	bodyFormat: DraftBodyFormat;
	formFields: DraftFormField[];
	variables: DraftVariable[];
	timeoutSec: number;
	sessionName: string;
	sessionNew: boolean;
	environment: string;
	assertions: DraftAssertion[];
}

const SENSITIVE_HEADERS = new Set([
	'authorization',
	'cookie',
	'set-cookie',
	'x-api-key',
	'x-auth-token',
]);

/** Returns a copy of headers with sensitive values blanked out. */
export function maskSensitiveHeaders(headers: DraftHeader[]): DraftHeader[] {
	return headers.map((h) =>
		SENSITIVE_HEADERS.has(h.key.trim().toLowerCase()) ? { key: h.key, value: '' } : h
	);
}

/** A blank draft with defaults matching a fresh editor state. */
export function emptyDraft(): RequestDraft {
	return {
		version: DRAFT_VERSION,
		method: 'GET',
		url: '',
		headers: [],
		body: '',
		bodyFormat: 'raw',
		formFields: [],
		variables: [],
		timeoutSec: 30,
		sessionName: '',
		sessionNew: false,
		environment: '',
		assertions: [],
	};
}

/** Serializes a draft to JSON for storage. */
export function serializeDraft(draft: RequestDraft): string {
	return JSON.stringify(draft);
}

/**
 * Parses a stored draft. Legacy entries (pre-versioning, e.g. old history
 * items that only carried method/url/headers/body) are filled with defaults
 * so previously persisted data keeps loading. Returns null for invalid input.
 */
export function deserializeDraft(raw: string): RequestDraft | null {
	let data: unknown;
	try {
		data = JSON.parse(raw);
	} catch {
		return null;
	}
	if (typeof data !== 'object' || data === null) return null;
	const src = data as Record<string, unknown>;
	const draft = emptyDraft();
	if (typeof src.method === 'string') draft.method = src.method;
	if (typeof src.url === 'string') draft.url = src.url;
	if (typeof src.body === 'string') draft.body = src.body;
	if (src.bodyFormat === 'raw' || src.bodyFormat === 'form-data' || src.bodyFormat === 'urlencoded') {
		draft.bodyFormat = src.bodyFormat;
	}
	if (typeof src.timeoutSec === 'number') draft.timeoutSec = src.timeoutSec;
	if (typeof src.sessionName === 'string') draft.sessionName = src.sessionName;
	if (typeof src.sessionNew === 'boolean') draft.sessionNew = src.sessionNew;
	if (typeof src.environment === 'string') draft.environment = src.environment;
	if (Array.isArray(src.headers)) {
		draft.headers = (src.headers as unknown[])
			.filter((h) => typeof h === 'object' && h !== null)
			.map((h) => {
				const row = h as Record<string, unknown>;
				return { key: typeof row.key === 'string' ? row.key : '', value: typeof row.value === 'string' ? row.value : '' };
			});
	}
	if (Array.isArray(src.formFields)) {
		draft.formFields = (src.formFields as unknown[])
			.filter((f) => typeof f === 'object' && f !== null)
			.map((f) => {
				const row = f as Record<string, unknown>;
				return {
					key: typeof row.key === 'string' ? row.key : '',
					value: typeof row.value === 'string' ? row.value : '',
					type: row.type === 'file' ? 'file' : 'text',
					fileName: typeof row.fileName === 'string' ? row.fileName : '',
					contentType: typeof row.contentType === 'string' ? row.contentType : '',
					hasFile: row.hasFile === true || row.type === 'file',
				};
			});
	}
	if (Array.isArray(src.variables)) {
		draft.variables = (src.variables as unknown[])
			.filter((v) => typeof v === 'object' && v !== null)
			.map((v) => {
				const row = v as Record<string, unknown>;
				return { key: typeof row.key === 'string' ? row.key : '', value: typeof row.value === 'string' ? row.value : '' };
			});
	}
	if (Array.isArray(src.assertions)) {
		draft.assertions = (src.assertions as unknown[])
			.filter((a) => typeof a === 'object' && a !== null)
			.map((a) => {
				const row = a as Record<string, unknown>;
				return {
					name: typeof row.name === 'string' ? row.name : '',
					kind: typeof row.kind === 'string' ? row.kind : '',
					target: typeof row.target === 'string' ? row.target : '',
					expected: typeof row.expected === 'string' ? row.expected : '',
				};
			});
	}
	return draft;
}

/** Shape of the subset a collection entry can hold (matches Go SavedRequest). */
export interface SavedRequestLike {
	name?: string;
	method: string;
	url: string;
	headers?: Record<string, string>;
	body?: string;
	variables?: Record<string, string>;
}

/** Builds a draft from a collection entry, filling defaults for gaps. */
export function fromSavedRequest(req: SavedRequestLike): RequestDraft {
	const draft = emptyDraft();
	draft.method = req.method;
	draft.url = req.url;
	draft.body = req.body ?? '';
	if (req.headers) {
		draft.headers = Object.entries(req.headers).map(([key, value]) => ({ key, value }));
	}
	if (req.variables) {
		draft.variables = Object.entries(req.variables).map(([key, value]) => ({ key, value }));
	}
	return draft;
}

/** Maps a draft to the subset a collection entry stores. */
export function toSavedRequest(name: string, draft: RequestDraft): SavedRequestLike {
	const headers: Record<string, string> = {};
	for (const h of draft.headers) {
		if (h.key.trim()) headers[h.key.trim()] = h.value;
	}
	const variables: Record<string, string> = {};
	for (const v of draft.variables) {
		if (v.key.trim()) variables[v.key.trim()] = v.value;
	}
	return {
		name,
		method: draft.method,
		url: draft.url,
		headers,
		body: draft.body,
		variables,
	};
}
