import { createApi, redirectToLogin } from '@facile/client';

export { ApiError, isUnauthorized } from '@facile/client';

export type WorkloadType = 'container' | 'static' | 'compose';
export type AppStatus = 'active' | 'activating' | 'deactivating' | 'inactive' | 'failed' | 'unknown';
export type DeployStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'rolled-back';
export type DeployTrigger = 'cli' | 'webhook' | 'api';
export type Lifecycle = 'start' | 'stop' | 'restart';

export type Deploy = {
	id: string;
	app: string;
	commit: string;
	trigger: DeployTrigger;
	status: DeployStatus;
	started_at: string;
	finished_at: string | null;
	error: string | null;
};

export type App = {
	name: string;
	type: WorkloadType;
	status: AppStatus;
	repo_url: string;
	branch: string;
	commit: string;
	port: number;
	domains: string[];
	tags: string[];
	created_at: string;
	last_deploy: Deploy | null;
};

export type EnvEntry = { key: string; masked: string };
export type Me = { email: string; name: string; picture: string | null };

export type NewApp = {
	name?: string;
	repo_url: string;
	branch?: string;
	type?: WorkloadType;
	port?: number;
};

export type EnvPatch = { set?: Record<string, string>; unset?: string[] };

/** The suite fetch kernel: relative `/api`, same-origin cookie, CSRF header on every mutation, 401 to sign-in. */
export const request = createApi({ onUnauthorized: redirectToLogin });

/** Encodes one path segment, so an app name or hostname can never reach another route. */
const seg = encodeURIComponent;

/** The path of one app's routes. */
const app = (name: string) => `/apps/${seg(name)}`;

/** Sends `body` as JSON with `method`. */
function send(method: string, body?: unknown): RequestInit {
	return { method, body: body === undefined ? undefined : JSON.stringify(body) };
}

export const api = {
	me: () => request<Me>('/me', {}, { quiet: true }),
	logout: () => request<void>('/auth/logout', send('POST')),

	apps: () => request<{ apps: App[] }>('/apps').then((answer) => answer.apps),
	app: (name: string) => request<App>(app(name)),
	createApp: (input: NewApp) => request<App>('/apps', send('POST', input)),
	destroyApp: (name: string, volumes: boolean) =>
		request<void>(`${app(name)}${volumes ? '?volumes=true' : ''}`, send('DELETE', { confirm: name })),

	deploy: (name: string) => request<Deploy>(`${app(name)}/deploy`, send('POST')),
	lifecycle: (name: string, action: Lifecycle) => request<App>(`${app(name)}/${action}`, send('POST')),
	rollback: (name: string, deployId?: string) =>
		request<Deploy>(`${app(name)}/rollback`, send('POST', deployId ? { deploy_id: deployId } : {})),
	deploys: (name: string) =>
		request<{ deploys: Deploy[] }>(`${app(name)}/deploys`).then((answer) => answer.deploys),

	env: (name: string) => request<{ entries: EnvEntry[] }>(`${app(name)}/env`).then((answer) => answer.entries),
	reveal: (name: string, key: string) =>
		request<{ key: string; value: string }>(`${app(name)}/env?reveal=${seg(key)}`).then((answer) => answer.value),
	patchEnv: (name: string, patch: EnvPatch) =>
		request<{ entries: EnvEntry[] }>(`${app(name)}/env`, send('PATCH', patch)).then((answer) => answer.entries),

	domains: (name: string) =>
		request<{ domains: string[] }>(`${app(name)}/domains`).then((answer) => answer.domains),
	addDomain: (name: string, hostname: string) =>
		request<{ domains: string[] }>(`${app(name)}/domains`, send('POST', { hostname })).then((a) => a.domains),
	removeDomain: (name: string, hostname: string) =>
		request<{ domains: string[] }>(`${app(name)}/domains/${seg(hostname)}`, send('DELETE')).then((a) => a.domains)
};

/**
 * Reads a plain-text answer, which the JSON kernel refuses: a finished deploy's log. A 401 still
 * goes to sign-in.
 */
export async function fetchText(url: string): Promise<string> {
	const response = await fetch(url, { credentials: 'same-origin', headers: { Accept: 'text/plain' } });
	if (response.status === 401) redirectToLogin();
	if (!response.ok) throw new Error(`Request failed with status ${response.status}`);
	return response.text();
}

/** The URL of a deploy's log: an event stream while it runs, plain text once it finished. */
export const deployLogUrl = (name: string, id: string) => `/api${app(name)}/deploys/${seg(id)}/log`;

/** The URL of an app's journal stream, starting `lines` lines back. */
export const runtimeLogUrl = (name: string, lines = 200) => `/api${app(name)}/logs?lines=${lines}`;
