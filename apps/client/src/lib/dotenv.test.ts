import { expect, test } from 'bun:test';
import { parseDotenv } from './dotenv';

test('parseDotenv reads the forms a pasted .env block uses', () => {
	const block = [
		'# database',
		'export DATABASE_URL=postgres://u:p@db/app',
		'PLAIN = value # trailing comment',
		'HASH=ab#cd',
		'QUOTED="ab #cd \\"x\\""',
		"SINGLE='$NOT_EXPANDED \\n'",
		'MULTI="line1',
		'line2"',
		'ESCAPED="a\\nb"',
		'EMPTY=',
		'not a pair',
		'1BAD=x'
	].join('\r\n');

	expect(parseDotenv(block)).toEqual({
		entries: {
			DATABASE_URL: 'postgres://u:p@db/app',
			PLAIN: 'value',
			HASH: 'ab#cd',
			QUOTED: 'ab #cd "x"',
			SINGLE: '$NOT_EXPANDED \\n',
			MULTI: 'line1\nline2',
			ESCAPED: 'a\nb',
			EMPTY: ''
		},
		invalid: [11, 12]
	});
});
