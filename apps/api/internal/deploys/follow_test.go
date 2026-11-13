package deploys

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
)

// fakeDeploy writes lines to a running deploy over time, the way a build streams its output, and
// finishes it after the last one.
func fakeDeploy(t *testing.T, r *Recorder, lines []string, gap time.Duration) {
	for _, line := range lines {
		time.Sleep(gap)
		r.Line(line)
	}
	if err := r.Finish(nil); err != nil {
		t.Error(err)
	}
}

func TestFollowReplaysThenStreamsThenEndsWithTheRecord(t *testing.T) {
	h := Open(t.TempDir(), "app")
	r, err := h.Start(Request{Trigger: TriggerAPI})
	if err != nil {
		t.Fatal(err)
	}
	r.Line("already there")
	go fakeDeploy(t, r, []string{"one", "two", "three"}, 100*time.Millisecond)

	var got []string
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	rec, err := h.Follow(ctx, r.Record().ID, 0, func(n int, text string) error {
		got = append(got, fmt.Sprintf("%d:%s", n, text))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1:already there", "2:one", "3:two", "4:three"}
	if !slices.Equal(got, want) {
		t.Errorf("followed lines = %v, want %v", got, want)
	}
	if rec.Status != StatusSucceeded {
		t.Errorf("follow must end with the final record, got %+v", rec)
	}
}

func TestFollowResumesAfterTheLastLineReceived(t *testing.T) {
	h := Open(t.TempDir(), "app")
	r, err := h.Start(Request{Trigger: TriggerAPI})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"a", "b", "c"} {
		r.Line(line)
	}
	if err := r.Finish(nil); err != nil {
		t.Fatal(err)
	}
	var got []int
	if _, err := h.Follow(t.Context(), r.Record().ID, 2, func(n int, _ string) error {
		got = append(got, n)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []int{3}) {
		t.Errorf("a resumed follow delivered lines %v, want [3]", got)
	}
}

func TestFollowStopsWithItsContext(t *testing.T) {
	h := Open(t.TempDir(), "app")
	r, err := h.Start(Request{Trigger: TriggerAPI})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	if _, err := h.Follow(ctx, r.Record().ID, 0, func(int, string) error { return nil }); err == nil {
		t.Error("following a deploy that never finishes must stop with the context")
	}
	if err := r.Finish(nil); err != nil {
		t.Fatal(err)
	}
}
