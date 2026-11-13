/** What a pasted `.env` block holds: the pairs it sets, and the lines that could not be read. */
export type ParsedEnv = {
	entries: Record<string, string>;
	invalid: number[];
};

const KEY = /^[A-Za-z_][A-Za-z0-9_]*$/;

const ESCAPES: Record<string, string> = { n: '\n', r: '\r', t: '\t', '"': '"', '\\': '\\' };

/** Whether `key` is a name a process environment accepts, so the editor refuses it before the API does. */
export function isEnvKey(key: string): boolean {
	return KEY.test(key);
}

/** Returns the index of the quote closing a double-quoted value that opens at 0, or -1 when it stays open. */
function closingQuote(text: string): number {
	for (let i = 1; i < text.length; i++) {
		if (text[i] === '\\') i++;
		else if (text[i] === '"') return i;
	}
	return -1;
}

/** Decodes the escapes a double-quoted value may carry, the ones Go's `strconv.Quote` writes. */
function unescape(body: string): string {
	return body.replace(/\\(.)/g, (whole, char: string) => ESCAPES[char] ?? whole);
}

/** Reads an unquoted value: everything up to an inline ` #` comment, trimmed. */
function bareValue(raw: string): string {
	const comment = raw.search(/\s#/);
	return (comment === -1 ? raw : raw.slice(0, comment)).trim();
}

/**
 * Parses a pasted `.env` block. Blank lines, comments and `export ` prefixes are skipped; a
 * double-quoted value may span lines and decodes its escapes; a single-quoted value is taken
 * literally. A line without `=` or with an invalid key is reported by its 1-based number rather
 * than dropped silently, so the person can fix the paste before anything is saved.
 */
export function parseDotenv(text: string): ParsedEnv {
	const lines = text.replace(/\r\n?/g, '\n').split('\n');
	const entries: Record<string, string> = {};
	const invalid: number[] = [];
	for (let index = 0; index < lines.length; index++) {
		const line = lines[index].trim().replace(/^export\s+/, '');
		if (line === '' || line.startsWith('#')) continue;
		const eq = line.indexOf('=');
		const key = eq === -1 ? '' : line.slice(0, eq).trim();
		if (!isEnvKey(key)) {
			invalid.push(index + 1);
			continue;
		}
		let raw = line.slice(eq + 1).trimStart();
		if (raw.startsWith('"')) {
			while (closingQuote(raw) === -1 && index + 1 < lines.length) raw += '\n' + lines[++index];
			const end = closingQuote(raw);
			entries[key] = unescape(raw.slice(1, end === -1 ? raw.length : end));
		} else if (raw.startsWith("'") && raw.indexOf("'", 1) !== -1) {
			entries[key] = raw.slice(1, raw.indexOf("'", 1));
		} else {
			entries[key] = bareValue(raw);
		}
	}
	return { entries, invalid };
}
