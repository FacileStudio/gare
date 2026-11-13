package deploys

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/FacileStudio/gare/apps/api/internal/apikit"
	"github.com/FacileStudio/gare/apps/api/internal/deploys"
)

// streamLog sends the deploy's log to stream from the line after after, one event per line with
// its number as the event id, then follows it while the deploy runs, and ends with a done event
// carrying the final record. It stops early, without done, when ctx ends, which is the client
// going away, or when the client can no longer be written to: there is nobody left to tell.
func streamLog(ctx context.Context, history deploys.History, id string, after int, stream *apikit.Stream) {
	final, err := history.Follow(ctx, id, after, func(n int, text string) error {
		return stream.Send(apikit.Event{ID: strconv.Itoa(n), Data: text})
	})
	if err != nil {
		return
	}
	data, err := json.Marshal(final)
	if err != nil {
		return
	}
	if err := stream.Send(apikit.Event{Name: "done", Data: string(data)}); err != nil {
		return
	}
}
