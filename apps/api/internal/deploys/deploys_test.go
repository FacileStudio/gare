package deploys

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStartRecordsARunningDeployAndFinishSettlesIt(t *testing.T) {
	h := Open(t.TempDir(), "app")
	r, err := h.Start(Request{Trigger: TriggerCLI})
	if err != nil {
		t.Fatal(err)
	}
	id := r.Record().ID
	rec, err := h.Get(id)
	if err != nil || rec.Status != StatusRunning || rec.Trigger != TriggerCLI || rec.FinishedAt != nil {
		t.Fatalf("a started deploy must be recorded running at once, got %+v, %v", rec, err)
	}
	if err := r.SetCommit("0123456789abcdef"); err != nil {
		t.Fatal(err)
	}
	if err := r.Finish(errors.New("boom")); err != nil {
		t.Fatal(err)
	}
	rec, _ = h.Get(id)
	if rec.Status != StatusFailed || rec.Error == nil || *rec.Error != "boom" || rec.FinishedAt == nil {
		t.Errorf("a failed deploy must be recorded failed with its error, got %+v", rec)
	}
	if rec.Commit != "0123456789abcdef" {
		t.Errorf("commit = %q, want the one set while running", rec.Commit)
	}
	info, err := os.Stat(filepath.Join(h.dir, id, recordFile))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Errorf("record.json must be 0600, got %v, %v", info, err)
	}
}

func TestLogKeepsSubprocessOutputAndLinesApart(t *testing.T) {
	h := Open(t.TempDir(), "app")
	r, err := h.Start(Request{Trigger: TriggerWebhook})
	if err != nil {
		t.Fatal(err)
	}
	r.Line("Building")
	if _, err := r.Write([]byte("step 1\nstep 2 no newline")); err != nil {
		t.Fatal(err)
	}
	r.Line("Done")
	if err := r.Finish(nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(h.dir, r.Record().ID, logFile))
	if err != nil {
		t.Fatal(err)
	}
	want := "Building\nstep 1\nstep 2 no newline\nDone\n"
	if string(data) != want {
		t.Errorf("log = %q, want %q", data, want)
	}
}

func TestHistoryListsNewestFirstAndKeepsFifty(t *testing.T) {
	h := Open(t.TempDir(), "app")
	var first, last string
	for i := range Keep + 2 {
		rec, err := h.Queue(Request{Trigger: TriggerAPI})
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = rec.ID
		}
		last = rec.ID
	}
	list, err := h.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != Keep {
		t.Fatalf("history kept %d records, want %d", len(list), Keep)
	}
	if list[0].ID != last {
		t.Errorf("newest record must come first, got %s want %s", list[0].ID, last)
	}
	if _, err := h.Get(first); !errors.Is(err, ErrNotFound) {
		t.Errorf("the oldest record must be pruned, got %v", err)
	}
}

func TestResumeTurnsAQueuedDeployRunning(t *testing.T) {
	h := Open(t.TempDir(), "app")
	queued, err := h.Queue(Request{Trigger: TriggerWebhook})
	if err != nil || queued.Status != StatusQueued {
		t.Fatalf("queue = %+v, %v", queued, err)
	}
	r, err := h.Resume(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rec, _ := h.Get(queued.ID); rec.Status != StatusRunning {
		t.Errorf("a resumed deploy must be running, got %s", rec.Status)
	}
	if err := r.Finish(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Resume(queued.ID); err == nil {
		t.Error("a finished deploy must not be resumed")
	}
}

func TestValidateIDRefusesPaths(t *testing.T) {
	for _, id := range []string{"", "..", "../x", "20261113T120000.000Z-abc/../../x", "x"} {
		if ValidateID(id) == nil {
			t.Errorf("ValidateID(%q) accepted a non-id", id)
		}
	}
	if err := ValidateID(newID(time.Now(), "")); err != nil {
		t.Errorf("a generated id must validate: %v", err)
	}
}
