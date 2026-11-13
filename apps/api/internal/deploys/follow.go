package deploys

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"time"
)

// followPoll is how often Follow looks for new lines of a running deploy. Polling the file is what
// lets a reader follow a deploy another process writes, the CLI's seen from the daemon's API.
const followPoll = 200 * time.Millisecond

// LineFunc receives one log line and its number, counted from 1. An error it returns stops Follow.
type LineFunc func(n int, text string) error

// Follow delivers the lines of a deploy's log numbered after after, then keeps delivering new ones
// while the deploy runs, and returns the final record once it finishes. A finished deploy gets its
// remaining lines and its record at once. It stops early with ctx's error, so a client that goes
// away ends it. A reconnecting client passes the number of the last line it received as after.
func (h History) Follow(ctx context.Context, id string, after int, deliver LineFunc) (Deploy, error) {
	file, err := h.OpenLog(id)
	if err != nil {
		return Deploy{}, err
	}
	defer file.Close()
	lines := &lineReader{r: bufio.NewReader(file), after: after, deliver: deliver}
	for {
		rec, err := h.Get(id)
		if err != nil {
			return Deploy{}, err
		}
		if err := lines.drain(rec.Finished()); err != nil {
			return Deploy{}, err
		}
		if rec.Finished() {
			return rec, nil
		}
		select {
		case <-ctx.Done():
			return Deploy{}, ctx.Err()
		case <-time.After(followPoll):
		}
	}
}

// lineReader splits a growing log into numbered lines, holding back a line the writer has not
// finished yet until it ends or the deploy does.
type lineReader struct {
	r       *bufio.Reader
	partial string
	n       int
	after   int
	deliver LineFunc
}

// drain delivers every complete line written so far, and the unterminated tail too when final says
// nothing more will be written. The record is read before the log, so a final status seen there
// means the log already holds everything.
func (l *lineReader) drain(final bool) error {
	for {
		chunk, err := l.r.ReadString('\n')
		l.partial += chunk
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if err := l.emit(); err != nil {
			return err
		}
	}
	if final && l.partial != "" {
		return l.emit()
	}
	return nil
}

// emit numbers the pending line and delivers it unless the caller already has it.
func (l *lineReader) emit() error {
	text := strings.TrimSuffix(strings.TrimSuffix(l.partial, "\n"), "\r")
	l.partial = ""
	l.n++
	if l.n <= l.after {
		return nil
	}
	return l.deliver(l.n, text)
}
