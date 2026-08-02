// Pure formatting and helper functions for the GUI. No DOM, localStorage or
// Wails dependencies, so these are unit-testable in a plain Node environment.

/** Escapes &, <, > for safe interpolation into HTML text. */
export function escapeHtml(s: string): string {
	return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

/**
 * Pretty-prints a JSON string with simple syntax highlighting.
 * Output is HTML-escaped first so untrusted response content cannot inject
 * elements/scripts; the {@html} sink receives escaped text only.
 */
export function formatJson(json: string): string {
	if (!json) return '';
	try {
		const parsed = JSON.parse(json);
		const formatted = JSON.stringify(parsed, null, 2);
		const escaped = escapeHtml(formatted);
		return escaped
			.replace(/"([^"]+)":/g, '<span class="text-sky-400">"$1"</span>:')
			.replace(/:\s*"([^"]*)"/g, ': <span class="text-green-400">"$1"</span>')
			.replace(/:\s*(-?\d+\.?\d*)/g, ': <span class="text-yellow-400">$1</span>')
			.replace(/:\s*(true|false)/g, ': <span class="text-purple-400">$1</span>')
			.replace(/:\s*(null)/g, ': <span class="text-red-400">$1</span>');
	} catch {
		return escapeHtml(json);
	}
}

/** Human-readable byte size, e.g. "457.57 KB". */
export function formatBytes(bytes: number): string {
	if (bytes === 0) return '0 B';
	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB'];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

/** Tailwind badge classes for an HTTP status code. */
export function getStatusColor(status: number): string {
	if (status >= 200 && status < 300) return 'bg-gradient-to-r from-green-500 to-green-600 text-white font-bold';
	if (status >= 300 && status < 400) return 'bg-gradient-to-r from-yellow-500 to-yellow-600 text-white font-bold';
	if (status >= 400 && status < 500) return 'bg-gradient-to-r from-orange-500 to-orange-600 text-white font-bold';
	if (status >= 500) return 'bg-gradient-to-r from-red-500 to-red-600 text-white font-bold';
	return 'bg-gradient-to-r from-gray-500 to-gray-600 text-white font-bold';
}

/** Normalizes unknown error shapes (Wails rejection, Error, string) to text. */
export function extractErrorMessage(err: unknown): string {
	if (err instanceof Error) return err.message;
	if (typeof err === 'string') return err;
	if (err && typeof err === 'object' && 'message' in err && typeof (err as { message: unknown }).message === 'string') {
		return (err as { message: string }).message;
	}
	return 'An unknown error occurred';
}

/** Base64-encodes a UTF-8 string (for Basic auth). */
export function btoaUtf8(s: string): string {
	return btoa(unescape(encodeURIComponent(s)));
}

/** Tailwind text color class for an HTTP method (used across panels). */
export function methodColorClass(method: string): string {
	switch (method) {
		case 'GET':
			return 'text-green-600';
		case 'POST':
			return 'text-blue-600';
		case 'PUT':
			return 'text-yellow-600';
		case 'DELETE':
			return 'text-red-600';
		default:
			return 'text-purple-600';
	}
}
