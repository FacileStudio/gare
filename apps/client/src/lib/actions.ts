import { goto } from '$app/navigation';
import { toast } from '@facile/muse';
import { api, type Deploy } from './api';
import { errorText } from './status';

/** The page that follows one deploy's log. */
export function deployHref(deploy: Deploy): string {
	return `/apps/${encodeURIComponent(deploy.app)}/deploys/${encodeURIComponent(deploy.id)}`;
}

/** Queues a deploy, or a rollback when `rollbackTo` is set, then opens its live log. */
export async function startDeploy(name: string, rollbackTo?: string): Promise<void> {
	try {
		const deploy = rollbackTo === undefined ? await api.deploy(name) : await api.rollback(name, rollbackTo);
		await goto(deployHref(deploy));
	} catch (error) {
		toast.danger(errorText(error), { title: rollbackTo === undefined ? 'Deploy refused' : 'Rollback refused' });
	}
}

/**
 * Calls `tick` every `ms` while the tab is visible and once more when it comes back, so a status
 * list stays current without polling a tab nobody looks at. Returns the cleanup for `$effect`.
 */
export function poll(ms: number, tick: () => void): () => void {
	const timer = setInterval(() => {
		if (!document.hidden) tick();
	}, ms);
	const onVisible = () => {
		if (!document.hidden) tick();
	};
	document.addEventListener('visibilitychange', onVisible);
	return () => {
		clearInterval(timer);
		document.removeEventListener('visibilitychange', onVisible);
	};
}
