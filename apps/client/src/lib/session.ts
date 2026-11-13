import { createSession } from '@facile/client/session';
import { createTheme } from '@facile/client/theme';
import { request, type Me } from './api';

/** The signed-in operator, read from `GET /api/me`, which answers the user bare. */
export const session = createSession<Me>(request, {
	mePath: '/me',
	read: (body) => (body as Me | undefined) ?? null
});

/** The light, dark or system choice, kept under the key `app.html` reads before first paint. */
export const theme = createTheme('gare-theme');
