/** One line of the deploy log the hero demo streams, marked the way gare's terminal sink marks it. */
export type Line = { mark: '▸' | '✓' | '!'; text: string; final?: boolean };

export const OLD = 'a41c9e2';
export const NEW = '4f2c1a9';
export const BAD = '9e3b7d0';

/** A deploy that passes its health probe, as gare prints it. */
export const LOG: Line[] = [
	{ mark: '▸', text: 'Syncing shop with its git remote...' },
	{ mark: '▸', text: `Building container image localhost/shop:${NEW}...` },
	{ mark: '✓', text: 'Synced manifest.yaml from repository' },
	{ mark: '▸', text: 'Verifying health probe port 8003 at http://127.0.0.1:8003/...' },
	{ mark: '✓', text: '1 health probe(s) passed' },
	{ mark: '✓', text: `Successfully deployed container app shop (${NEW}) -> shop.example.com`, final: true }
];

/** The next push fails its health probe and gare puts the previous build back. */
export const FAIL_LOG: Line[] = [
	{ mark: '▸', text: 'Syncing shop with its git remote...' },
	{ mark: '▸', text: `Building container image localhost/shop:${BAD}...` },
	{ mark: '▸', text: 'Verifying health probe port 8003 at http://127.0.0.1:8003/...' },
	{ mark: '!', text: `health probe port 8003 failed; rolling back to ${NEW}` },
	{ mark: '▸', text: 'Verifying health probe port 8003 at http://127.0.0.1:8003/...' },
	{ mark: '✓', text: `Rolled back shop to deploy ${NEW}`, final: true }
];

/** The two other app cards, drawn as placeholders. */
export const OTHERS = [
	{ name: 'blog', type: 'Static', widths: [54, 40] },
	{ name: 'api', type: 'Compose', widths: [46, 58] }
];
