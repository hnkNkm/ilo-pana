import { describe, expect, it } from 'vitest';
import { buildUrlFromParams, parseQueryParams } from './url-utils';

describe('parseQueryParams', () => {
	it('returns [] for a URL without query string', () => {
		expect(parseQueryParams('https://example.com/path')).toEqual([]);
	});

	it('parses a single param', () => {
		expect(parseQueryParams('https://example.com/path?a=1')).toEqual([
			{ key: 'a', value: '1', enabled: true },
		]);
	});

	it('parses multiple params and decodes values', () => {
		expect(parseQueryParams('https://example.com/path?name=hello%20world&n=2')).toEqual([
			{ key: 'name', value: 'hello world', enabled: true },
			{ key: 'n', value: '2', enabled: true },
		]);
	});

	it('handles params without a value', () => {
		expect(parseQueryParams('https://example.com/path?flag&x=1')).toEqual([
			{ key: 'flag', value: '', enabled: true },
			{ key: 'x', value: '1', enabled: true },
		]);
	});

	it('keeps raw text when decoding fails', () => {
		expect(parseQueryParams('https://example.com/path?bad=%ZZ')).toEqual([
			{ key: 'bad', value: '%ZZ', enabled: true },
		]);
	});

	it('skips empty segments', () => {
		expect(parseQueryParams('https://example.com/path?a=1&&b=2')).toEqual([
			{ key: 'a', value: '1', enabled: true },
			{ key: 'b', value: '2', enabled: true },
		]);
	});
});

describe('buildUrlFromParams', () => {
	it('returns the base URL when there are no enabled params', () => {
		expect(buildUrlFromParams('https://example.com/path', [])).toBe('https://example.com/path');
		expect(buildUrlFromParams('https://example.com/path', [{ key: 'old', value: '1', enabled: false }])).toBe('https://example.com/path');
	});

	it('skips disabled and empty-key params', () => {
		const params = [
			{ key: 'a', value: '1', enabled: false },
			{ key: '', value: 'x', enabled: true },
			{ key: 'b', value: '2', enabled: true },
		];
		expect(buildUrlFromParams('https://example.com/path', params)).toBe('https://example.com/path?b=2');
	});

	it('encodes keys and values', () => {
		expect(buildUrlFromParams('https://example.com/path', [{ key: 'q', value: 'a b&c', enabled: true }])).toBe(
			'https://example.com/path?q=a%20b%26c'
		);
	});
});
