// The single typed boundary to the Wails bindings. All DTO construction
// (models.ts) and binding calls ($wailsjs) happen here; components and App.svelte
// only deal with plain data. Regenerating the bindings touches this file alone.

import {
	DeleteCollection,
	DeleteEnvironment,
	DeleteRequest,
	EvaluateAssertions,
	ExecuteRequest,
	ExportCollection,
	GenerateCurl,
	GetCollection,
	GetEnvironment,
	ImportCollection,
	ImportCurl,
	ImportOpenAPI,
	ListCollections,
	ListEnvironments,
	SaveEnvironment,
	SaveRequest,
} from '$wailsjs/go/main/App';
import { assertion, collection, config, curl, environment, main, response } from '$wailsjs/go/models';

export type ResponseData = response.ResponseData;
export type Collection = collection.Collection;
export type SavedRequest = collection.SavedRequest;
export type Environment = environment.Environment;
export type AssertionRule = assertion.Rule;
export type AssertionResult = assertion.Result;
export type CurlRequest = curl.Request;

export interface SendRequestInput {
	method: string;
	url: string;
	body: string;
	bodyFormat: 'raw' | 'form-data' | 'urlencoded';
	formFields: Array<{
		key: string;
		value: string;
		type: 'text' | 'file';
		fileName: string;
		fileContent: number[];
		contentType: string;
	}>;
	headers: Record<string, string>;
	timeoutSec: number;
	sessionName: string;
	sessionNew: boolean;
	variables: Record<string, string>;
	environment: string;
}

function toRequestParams(input: SendRequestInput): main.RequestParams {
	const formFields = input.formFields
		.filter((f) => f.key.trim())
		.map(
			(f) =>
				new config.FormField({
					key: f.key.trim(),
					value: f.value,
					isFile: f.type === 'file',
					fileName: f.type === 'file' ? f.fileName : '',
					fileContent: f.type === 'file' ? f.fileContent : [],
					contentType: f.contentType,
				})
		);
	return new main.RequestParams({
		Method: input.method,
		URL: input.url,
		Body: input.body,
		BodyFormat: input.bodyFormat === 'raw' ? '' : input.bodyFormat === 'form-data' ? 'multipart' : input.bodyFormat,
		FormFields: formFields,
		Headers: input.headers,
		TimeoutMs: input.timeoutSec * 1000,
		SessionName: input.sessionName,
		SessionNew: input.sessionNew,
		Variables: input.variables,
		Environment: input.environment,
	});
}

export async function sendRequest(input: SendRequestInput): Promise<ResponseData> {
	return ExecuteRequest(toRequestParams(input));
}

export async function evaluateAssertions(result: ResponseData, rules: AssertionRule[]): Promise<AssertionResult[]> {
	return EvaluateAssertions(result, rules);
}

/** Builds an assertion rule DTO (used by the editor's rule rows). */
export function newAssertionRule(partial: { kind: string; target?: string; expected?: string }): AssertionRule {
	return new assertion.Rule({ kind: partial.kind, target: partial.target, expected: partial.expected });
}

export async function generateCurl(method: string, url: string, headers: Record<string, string>, body: string): Promise<string> {
	return GenerateCurl(new main.CurlParams({ Method: method, URL: url, Headers: headers, Body: body }));
}

export async function importCurl(cmd: string): Promise<CurlRequest> {
	return ImportCurl(cmd);
}

export async function importOpenAPI(content: string, collectionName: string): Promise<number> {
	return ImportOpenAPI(content, collectionName);
}

// Collections

export async function listCollections(): Promise<string[]> {
	return ListCollections();
}

export async function getCollection(name: string): Promise<Collection> {
	return GetCollection(name);
}

export async function saveRequest(collectionName: string, request: SavedRequest): Promise<void> {
	return SaveRequest(collectionName, request);
}

/** Wraps plain collection request data in the SavedRequest DTO. */
export function newSavedRequest(source: unknown): SavedRequest {
	return new collection.SavedRequest(source);
}

export async function deleteRequest(collectionName: string, requestName: string): Promise<void> {
	return DeleteRequest(collectionName, requestName);
}

export async function deleteCollection(name: string): Promise<void> {
	return DeleteCollection(name);
}

export async function exportCollection(name: string): Promise<string> {
	return ExportCollection(name);
}

export async function importCollection(json: string): Promise<void> {
	return ImportCollection(json);
}

// Environments

export async function listEnvironments(): Promise<string[]> {
	return ListEnvironments();
}

export async function getEnvironment(name: string): Promise<Environment> {
	return GetEnvironment(name);
}

export async function saveEnvironment(name: string, variables: Record<string, string>): Promise<void> {
	return SaveEnvironment(name, variables);
}

export async function deleteEnvironment(name: string): Promise<void> {
	return DeleteEnvironment(name);
}
