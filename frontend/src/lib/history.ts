// Request history persistence. Storage is injected so the logic is testable
// without a browser; pass `localStorage` from the GUI.

import { deserializeDraft, maskSensitiveHeaders, type RequestDraft } from './request-draft';

export interface HistoryEntry extends RequestDraft {
	timestamp: number;
}

export const HISTORY_KEY = 'ilo-pana-request-history';
export const HISTORY_MAX = 10;

/** Reads and normalizes persisted history entries. */
export function loadHistory(storage: Pick<Storage, 'getItem'> | null | undefined): HistoryEntry[] {
	try {
		const raw = storage?.getItem(HISTORY_KEY);
		if (!raw) return [];
		const parsed = JSON.parse(raw);
		if (!Array.isArray(parsed)) return [];
		return parsed
			.map((entry) => deserializeDraft(JSON.stringify(entry)))
			.filter((d): d is RequestDraft => d !== null)
			.map((draft, i) => {
				const src = parsed[i] as Record<string, unknown>;
				return { ...draft, timestamp: typeof src.timestamp === 'number' ? src.timestamp : 0 };
			});
	} catch {
		return [];
	}
}

/**
 * Prepends a draft as the newest history entry. History is automatic, so
 * secrets are masked, entries are deduplicated by method + URL (latest wins),
 * and the list is capped at HISTORY_MAX.
 */
export function addHistoryEntry(entries: HistoryEntry[], draft: RequestDraft, now: number = Date.now()): HistoryEntry[] {
	const entry: HistoryEntry = { ...draft, headers: maskSensitiveHeaders(draft.headers), timestamp: now };
	const filtered = entries.filter((h) => !(h.method === entry.method && h.url === entry.url));
	return [entry, ...filtered].slice(0, HISTORY_MAX);
}

/** Persists the history list (silently ignores storage failures). */
export function saveHistory(entries: HistoryEntry[], storage: Pick<Storage, 'setItem'> | null | undefined): void {
	try {
		storage?.setItem(HISTORY_KEY, JSON.stringify(entries));
	} catch {
		// localStorage unavailable (private mode etc.) - history just won't persist
	}
}

/** Removes all persisted history. */
export function clearHistory(storage: Pick<Storage, 'removeItem'> | null | undefined): void {
	try {
		storage?.removeItem(HISTORY_KEY);
	} catch {
		// ignore
	}
}
