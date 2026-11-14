import type { Catalogs, Message } from '@facile/muse';
import { demo } from './demo';
import { landing } from './landing';
import { sections } from './sections';
import type { Row } from '../row';

const rows = { ...landing, ...sections, ...demo };

export type AppKey = keyof typeof rows;
type AppCatalog = Record<AppKey, Message>;

function column(index: number): AppCatalog {
	const entries = Object.entries(rows as Record<string, Row>).map(([key, row]) => [key, row[index]]);
	return Object.fromEntries(entries) as AppCatalog;
}

/** The public pages' catalogs: English is the typed base, and every row carries all four locales. */
export const catalogs: Catalogs<AppCatalog> = {
	en: column(0),
	fr: column(1),
	es: column(2),
	de: column(3)
};
