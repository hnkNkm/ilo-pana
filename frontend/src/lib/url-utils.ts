// Pure URL/query-string helpers for the request editor. No DOM or Wails
// dependencies, so these are unit-testable in a plain Node environment.

export interface QueryParam {
	key: string;
	value: string;
	enabled: boolean;
}

/** Splits the query string of a URL into its parameters (values decoded). */
export function parseQueryParams(url: string): QueryParam[] {
	const q = url.split('?')[1];
	const params: QueryParam[] = [];
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
	return params;
}

/** Merges enabled parameters back into the URL (values encoded). */
export function buildUrlFromParams(base: string, params: QueryParam[]): string {
	const parts = params
		.filter((p) => p.enabled && p.key.trim())
		.map((p) => `${encodeURIComponent(p.key.trim())}=${encodeURIComponent(p.value)}`);
	return parts.length ? `${base}?${parts.join('&')}` : base;
}
