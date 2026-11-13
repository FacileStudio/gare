import { getContext, setContext } from 'svelte';
import { api, type App } from './api';
import { errorText } from './status';

/** The app the detail pages share, so each tab reads one record instead of fetching its own. */
export class CurrentApp {
	name = $state('');
	app = $state<App | null>(null);
	error = $state<string | null>(null);

	/** Points the record at `name` and loads it, dropping what an earlier app left behind. */
	open(name: string): Promise<void> {
		if (name !== this.name) {
			this.name = name;
			this.app = null;
			this.error = null;
		}
		return this.refresh();
	}

	/** Reloads the record, keeping the last good copy on screen when the reload fails. */
	async refresh(): Promise<void> {
		const name = this.name;
		try {
			const app = await api.app(name);
			if (name === this.name) {
				this.app = app;
				this.error = null;
			}
		} catch (error) {
			if (name === this.name) this.error = errorText(error);
		}
	}
}

const KEY = Symbol('gare.current-app');

/** Shares a new `CurrentApp` with every page under the app layout. */
export function provideCurrentApp(): CurrentApp {
	return setContext(KEY, new CurrentApp());
}

/** Reads the `CurrentApp` the app layout shares. */
export function currentApp(): CurrentApp {
	return getContext<CurrentApp>(KEY);
}
