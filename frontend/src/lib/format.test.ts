import { describe, expect, it } from 'vitest';
import { btoaUtf8, escapeHtml, extractErrorMessage, formatBytes, formatJson, getStatusColor, methodColorClass } from './format';

describe('escapeHtml', () => {
	it('escapes &, < and >', () => {
		expect(escapeHtml('<script>&"quote"</script>')).toBe('&lt;script&gt;&amp;"quote"&lt;/script&gt;');
	});
});

describe('formatJson', () => {
	it('returns empty string for empty input', () => {
		expect(formatJson('')).toBe('');
	});

	it('escapes non-JSON input', () => {
		expect(formatJson('<html>')).toBe('&lt;html&gt;');
	});

	it('highlights JSON keys and values', () => {
		const out = formatJson('{"a": 1, "b": "x"}');
		expect(out).toContain('text-sky-400');
		expect(out).toContain('text-green-400');
		expect(out).toContain('text-yellow-400');
		expect(out).not.toContain('<script>');
	});

	it('escapes untrusted content inside JSON strings', () => {
		const out = formatJson('{"k": "<img onerror=alert(1)>"}');
		expect(out).not.toContain('<img');
	});
});

describe('formatBytes', () => {
	it('formats sizes', () => {
		expect(formatBytes(0)).toBe('0 B');
		expect(formatBytes(1024)).toBe('1 KB');
		expect(formatBytes(1536)).toBe('1.5 KB');
		expect(formatBytes(1048576)).toBe('1 MB');
	});
});

describe('getStatusColor', () => {
	it('maps status classes', () => {
		expect(getStatusColor(200)).toContain('green');
		expect(getStatusColor(301)).toContain('yellow');
		expect(getStatusColor(404)).toContain('orange');
		expect(getStatusColor(500)).toContain('red');
		expect(getStatusColor(100)).toContain('gray');
	});
});

describe('extractErrorMessage', () => {
	it('handles Error, string, message-object and unknown', () => {
		expect(extractErrorMessage(new Error('boom'))).toBe('boom');
		expect(extractErrorMessage('raw')).toBe('raw');
		expect(extractErrorMessage({ message: 'obj' })).toBe('obj');
		expect(extractErrorMessage(42)).toBe('An unknown error occurred');
	});
});

describe('btoaUtf8', () => {
	it('encodes UTF-8 strings', () => {
		expect(btoaUtf8('user:päss')).toBe('dXNlcjpww6Rzcw==');
	});
});

describe('methodColorClass', () => {
	it('maps methods to colors', () => {
		expect(methodColorClass('GET')).toBe('text-green-600');
		expect(methodColorClass('POST')).toBe('text-blue-600');
		expect(methodColorClass('PUT')).toBe('text-yellow-600');
		expect(methodColorClass('DELETE')).toBe('text-red-600');
		expect(methodColorClass('PATCH')).toBe('text-purple-600');
		expect(methodColorClass('OPTIONS')).toBe('text-purple-600');
	});
});
