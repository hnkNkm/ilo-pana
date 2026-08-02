import { describe, expect, it } from 'vitest';
import { addHistoryEntry, clearHistory, HISTORY_KEY, loadHistory, saveHistory, type HistoryEntry } from './history';
import { emptyDraft, type RequestDraft } from './request-draft';

function createStorage(initial: Record<string, string> = {}): Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> {
	const map = new Map(Object.entries(initial));
	return {
		getItem: (k) => map.get(k) ?? null,
		setItem: (k, v) => void map.set(k, String(v)),
		removeItem: (k) => void map.delete(k),
	};
}

function draft(overrides: Partial<RequestDraft> = {}): RequestDraft {
	return { ...emptyDraft(), url: 'https://example.com/api', ...overrides };
}

describe('loadHistory', () => {
	it('returns [] when storage is empty', () => {
		expect(loadHistory(createStorage())).toEqual([]);
	});

	it('returns [] when storage is undefined', () => {
		expect(loadHistory(undefined)).toEqual([]);
	});

	it('returns [] on corrupt JSON', () => {
		expect(loadHistory(createStorage({ [HISTORY_KEY]: '{not json' }))).toEqual([]);
	});

	it('returns [] when the stored value is not an array', () => {
		expect(loadHistory(createStorage({ [HISTORY_KEY]: '{"a":1}' }))).toEqual([]);
	});

	it('loads v1 entries with their timestamps', () => {
		const stored = [{ ...draft({ method: 'POST' }), timestamp: 1234 }];
		const loaded = loadHistory(createStorage({ [HISTORY_KEY]: JSON.stringify(stored) }));
		expect(loaded).toEqual([{ ...stored[0], version: 1 }]);
		expect(loaded[0].timestamp).toBe(1234);
	});

	it('fills defaults for legacy entries (no version field)', () => {
		const legacy = { method: 'PATCH', url: 'https://legacy.example.com/x', headers: [{ key: 'Accept', value: 'text/plain' }] };
		const loaded = loadHistory(createStorage({ [HISTORY_KEY]: JSON.stringify([legacy]) }));
		expect(loaded).toHaveLength(1);
		expect(loaded[0].method).toBe('PATCH');
		expect(loaded[0].url).toBe('https://legacy.example.com/x');
		expect(loaded[0].headers).toEqual([{ key: 'Accept', value: 'text/plain' }]);
		expect(loaded[0].bodyFormat).toBe('raw');
		expect(loaded[0].assertions).toEqual([]);
	});

	it('drops entries that are not objects', () => {
		const stored = [{ method: 'GET', url: 'https://a.example', timestamp: 1 }, null, 'junk', 42];
		const loaded = loadHistory(createStorage({ [HISTORY_KEY]: JSON.stringify(stored) }));
		expect(loaded).toHaveLength(1);
	});
});

describe('addHistoryEntry', () => {
	it('prepends the newest entry', () => {
		const old = { ...draft({ url: 'https://a.example' }), timestamp: 1 };
		const next = addHistoryEntry([old], draft({ url: 'https://b.example' }), 2);
		expect(next).toHaveLength(2);
		expect(next[0].url).toBe('https://b.example');
		expect(next[0].timestamp).toBe(2);
	});

	it('masks sensitive headers', () => {
		const d = draft({
			headers: [
				{ key: 'Authorization', value: 'Bearer secret' },
				{ key: 'Accept', value: 'application/json' },
			],
		});
		const next = addHistoryEntry([], d, 1);
		expect(next[0].headers).toEqual([
			{ key: 'Authorization', value: '' },
			{ key: 'Accept', value: 'application/json' },
		]);
	});

	it('does not mutate the input draft or entries', () => {
		const d = draft({ headers: [{ key: 'Authorization', value: 'secret' }] });
		const entries: HistoryEntry[] = [];
		addHistoryEntry(entries, d, 1);
		expect(d.headers[0].value).toBe('secret');
		expect(entries).toHaveLength(0);
	});

	it('deduplicates by method + URL (latest wins)', () => {
		const first = { ...draft({ method: 'GET', url: 'https://a.example' }), timestamp: 1 };
		const second = addHistoryEntry([first], draft({ method: 'GET', url: 'https://a.example' }), 2);
		expect(second).toHaveLength(1);
		expect(second[0].timestamp).toBe(2);
	});

	it('caps the list at 10 entries', () => {
		let entries: HistoryEntry[] = [];
		for (let i = 0; i < 12; i++) {
			entries = addHistoryEntry(entries, draft({ url: `https://a.example/${i}` }), i);
		}
		expect(entries).toHaveLength(10);
		expect(entries[0].url).toBe('https://a.example/11');
	});
});

describe('saveHistory / clearHistory', () => {
	it('round-trips through storage', () => {
		const storage = createStorage();
		const entries: HistoryEntry[] = [{ ...draft(), timestamp: 5 }];
		saveHistory(entries, storage);
		expect(loadHistory(storage)).toEqual(entries);
	});

	it('clearHistory removes the stored key', () => {
		const storage = createStorage({ [HISTORY_KEY]: '[]' });
		clearHistory(storage);
		expect(loadHistory(storage)).toEqual([]);
	});

	it('survives storage failures silently', () => {
		const failing = {
			getItem: () => {
				throw new Error('denied');
			},
			setItem: () => {
				throw new Error('denied');
			},
			removeItem: () => {
				throw new Error('denied');
			},
		};
		expect(loadHistory(failing)).toEqual([]);
		expect(() => saveHistory([], failing)).not.toThrow();
		expect(() => clearHistory(failing)).not.toThrow();
	});
});
