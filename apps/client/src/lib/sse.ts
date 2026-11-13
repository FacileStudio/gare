import type { Deploy } from './api';

export type StreamHandlers = {
	onLine: (line: string) => void;
	onDone?: (deploy: Deploy) => void;
	onBroken: () => void;
};

/**
 * Follows one of the API's event streams: each unnamed `data:` is a line, and a final
 * `event: done` carries the finished `Deploy`. The stream is closed on the first error instead
 * of letting `EventSource` reconnect, because a reconnect replays the stream from its start and
 * would print every line twice; `onBroken` lets the caller decide how to resume. Returns the
 * function that closes the stream.
 */
export function follow(url: string, handlers: StreamHandlers): () => void {
	const source = new EventSource(url);
	source.onmessage = (event) => handlers.onLine(event.data);
	source.addEventListener('done', (event) => {
		source.close();
		handlers.onDone?.(JSON.parse((event as MessageEvent<string>).data) as Deploy);
	});
	source.onerror = () => {
		source.close();
		handlers.onBroken();
	};
	return () => source.close();
}
