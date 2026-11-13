import type { Tone } from '@facile/muse';
import type { AppStatus, Deploy, DeployStatus, WorkloadType } from './api';

export const appTone: Record<AppStatus, Tone> = {
	active: 'success',
	activating: 'info',
	deactivating: 'info',
	inactive: 'neutral',
	failed: 'danger',
	unknown: 'neutral'
};

export const appLabel: Record<AppStatus, string> = {
	active: 'Running',
	activating: 'Starting',
	deactivating: 'Stopping',
	inactive: 'Stopped',
	failed: 'Failed',
	unknown: 'Unknown'
};

export const deployTone: Record<DeployStatus, Tone> = {
	queued: 'neutral',
	running: 'info',
	succeeded: 'success',
	failed: 'danger',
	'rolled-back': 'warning'
};

export const deployLabel: Record<DeployStatus, string> = {
	queued: 'Queued',
	running: 'Running',
	succeeded: 'Succeeded',
	failed: 'Failed',
	'rolled-back': 'Rolled back'
};

export const workloadLabel: Record<WorkloadType, string> = {
	container: 'Container',
	static: 'Static',
	compose: 'Compose'
};

/** Whether a status is a transition, which the UI shows with a pulsing dot. */
export function isTransient(status: AppStatus): boolean {
	return status === 'activating' || status === 'deactivating';
}

/** Whether a deploy has not finished yet, so its log is still a stream. */
export function isLive(deploy: Deploy): boolean {
	return deploy.status === 'queued' || deploy.status === 'running';
}

/** The seven-character commit people recognise, or an empty string before the first deploy. */
export function shortCommit(commit: string): string {
	return commit ? commit.slice(0, 7) : '';
}

const relative = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });

const STEPS: [Intl.RelativeTimeFormatUnit, number][] = [
	['year', 31_536_000],
	['month', 2_592_000],
	['day', 86_400],
	['hour', 3_600],
	['minute', 60]
];

/** Says how long ago `iso` was, in the coarsest unit that is at least one. */
export function ago(iso: string, now = Date.now()): string {
	const seconds = Math.round((Date.parse(iso) - now) / 1000);
	for (const [unit, size] of STEPS) {
		if (Math.abs(seconds) >= size) return relative.format(Math.round(seconds / size), unit);
	}
	return relative.format(seconds, 'second');
}

/** How long a finished deploy took, as `1m 12s`, or an empty string while it runs. */
export function took(deploy: Deploy): string {
	if (!deploy.finished_at) return '';
	const total = Math.max(0, Math.round((Date.parse(deploy.finished_at) - Date.parse(deploy.started_at)) / 1000));
	return total >= 60 ? `${Math.floor(total / 60)}m ${total % 60}s` : `${total}s`;
}

/** The message to show for a failed call: the API's own words when it sent some. */
export function errorText(error: unknown): string {
	if (error instanceof Error) return error.message;
	return 'Something went wrong.';
}
