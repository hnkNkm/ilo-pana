import { describe, expect, it } from 'vitest';
import {
	DRAFT_VERSION,
	deserializeDraft,
	emptyDraft,
	fromSavedRequest,
	maskSensitiveHeaders,
	serializeDraft,
	toSavedRequest,
	type RequestDraft,
} from './request-draft';

function fullDraft(): RequestDraft {
	return {
		version: DRAFT_VERSION,
		method: 'POST',
		url: 'https://api.example.com/v1/pets?verbose=yes',
		headers: [
			{ key: 'Content-Type', value: 'application/json' },
			{ key: 'X-Trace', value: 'abc' },
		],
		body: '{"name":"rex"}',
		bodyFormat: 'form-data',
		formFields: [
			{ key: 'name', value: 'rex', type: 'text', fileName: '', contentType: '', hasFile: false },
			{ key: 'avatar', value: '', type: 'file', fileName: 'avatar.png', contentType: 'image/png', hasFile: true },
		],
		variables: [{ key: 'TOKEN', value: 'secret' }],
		timeoutSec: 15,
		sessionName: 'dev',
		sessionNew: true,
		environment: 'staging',
		assertions: [{ name: '', kind: 'status_equals', target: '200', expected: '' }],
	};
}

describe('request-draft round-trip', () => {
	it('serialize -> deserialize preserves the full editor state', () => {
		const draft = fullDraft();
		const restored = deserializeDraft(serializeDraft(draft));
		expect(restored).toEqual(draft);
	});

	it('deserialize fills defaults for legacy entries', () => {
		// Pre-versioning history entry: only method/url/headers/body.
		const legacy = JSON.stringify({
			method: 'GET',
			url: 'https://api.example.com/x',
			headers: [{ key: 'Accept', value: 'application/json' }],
			body: 'hello',
		});
		const restored = deserializeDraft(legacy);
		expect(restored).not.toBeNull();
		expect(restored!.method).toBe('GET');
		expect(restored!.url).toBe('https://api.example.com/x');
		expect(restored!.headers).toEqual([{ key: 'Accept', value: 'application/json' }]);
		expect(restored!.body).toBe('hello');
		expect(restored!.bodyFormat).toBe('raw');
		expect(restored!.formFields).toEqual([]);
		expect(restored!.timeoutSec).toBe(30);
	});

	it('returns null for invalid input', () => {
		expect(deserializeDraft('not json')).toBeNull();
		expect(deserializeDraft('null')).toBeNull();
		expect(deserializeDraft('42')).toBeNull();
	});

	it('emptyDraft matches editor defaults', () => {
		const draft = emptyDraft();
		expect(draft.method).toBe('GET');
		expect(draft.bodyFormat).toBe('raw');
		expect(draft.timeoutSec).toBe(30);
	});
});

describe('secrets policy', () => {
	it('maskSensitiveHeaders blanks Authorization and friends', () => {
		const masked = maskSensitiveHeaders([
			{ key: 'Authorization', value: 'Bearer abc' },
			{ key: 'X-API-Key', value: 'k' },
			{ key: 'X-Trace', value: 'keep' },
		]);
		expect(masked).toEqual([
			{ key: 'Authorization', value: '' },
			{ key: 'X-API-Key', value: '' },
			{ key: 'X-Trace', value: 'keep' },
		]);
	});

	it('file contents are never part of a draft', () => {
		const draft = fullDraft();
		expect(draft.formFields.every((f) => !('fileContent' in f))).toBe(true);
	});
});

describe('collection conversion', () => {
	it('toSavedRequest keeps only the persistent subset', () => {
		const saved = toSavedRequest('createPet', fullDraft());
		expect(saved.name).toBe('createPet');
		expect(saved.method).toBe('POST');
		expect(saved.url).toBe('https://api.example.com/v1/pets?verbose=yes');
		expect(saved.headers).toEqual({
			'Content-Type': 'application/json',
			'X-Trace': 'abc',
		});
		expect(saved.variables).toEqual({ TOKEN: 'secret' });
		expect(saved.body).toBe('{"name":"rex"}');
	});

	it('fromSavedRequest fills gaps with defaults', () => {
		const draft = fromSavedRequest({
			method: 'DELETE',
			url: 'https://api.example.com/pets/1',
		});
		expect(draft.method).toBe('DELETE');
		expect(draft.body).toBe('');
		expect(draft.headers).toEqual([]);
		expect(draft.bodyFormat).toBe('raw');
	});

	it('collection round-trip preserves what the subset can hold', () => {
		const saved = toSavedRequest('x', fullDraft());
		const restored = fromSavedRequest(saved);
		expect(restored.method).toBe('POST');
		expect(restored.url).toBe('https://api.example.com/v1/pets?verbose=yes');
		expect(restored.headers).toEqual([
			{ key: 'Content-Type', value: 'application/json' },
			{ key: 'X-Trace', value: 'abc' },
		]);
		expect(restored.variables).toEqual([{ key: 'TOKEN', value: 'secret' }]);
		expect(restored.body).toBe('{"name":"rex"}');
	});
});
