package server

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func newBranchServer(calls *atomic.Int32) *httptest.Server {
	s := New(Config{
		Secret: testSecret,
		DeployHandler: func(ctx context.Context, appName string) error {
			calls.Add(1)
			return nil
		},
		BranchFor: func(app string) (string, error) {
			switch app {
			case "app":
				return "main", nil
			case "anybranch":
				return "", nil
			}
			return "", ErrUnknownApp
		},
	})
	return httptest.NewServer(s.Handler())
}

func postEvent(t *testing.T, targetURL, header, event, body string) (int, string) {
	t.Helper()
	payload := []byte(body)
	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Hub-Signature-256", computeSignature(testSecret, payload))
	req.Header.Set(header, event)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	answer, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(answer)
}

func TestWebhookEventFilter(t *testing.T) {
	var calls atomic.Int32
	ts := newBranchServer(&calls)
	defer ts.Close()

	cases := []struct {
		name, app, header, event, body string
		status                         int
		answer                         string
	}{
		{"github ping", "app", "X-GitHub-Event", "ping", `{"zen":"x"}`, http.StatusOK, "ignored: ping"},
		{"github issue", "app", "X-GitHub-Event", "issues", `{}`, http.StatusOK, "ignored: issues"},
		{"push to other branch", "app", "X-GitHub-Event", "push", `{"ref":"refs/heads/feature"}`, http.StatusOK, "ignored: refs/heads/feature"},
		{"gitlab tag push", "app", "X-Gitlab-Event", "Tag Push Hook", `{"ref":"refs/tags/v1"}`, http.StatusOK, "ignored: Tag Push Hook"},
		{"push without ref", "app", "X-GitHub-Event", "push", `{}`, http.StatusBadRequest, "Bad Request\n"},
		{"unknown app", "ghost", "X-GitHub-Event", "push", `{"ref":"refs/heads/main"}`, http.StatusNotFound, "unknown app"},
	}
	for _, tc := range cases {
		status, answer := postEvent(t, ts.URL+"/webhook/"+tc.app, tc.header, tc.event, tc.body)
		if status != tc.status || answer != tc.answer {
			t.Errorf("%s: got %d %q, want %d %q", tc.name, status, answer, tc.status, tc.answer)
		}
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("got %d deploys for ignored deliveries, want 0", got)
	}
}

func TestWebhookPushToDeployedBranch(t *testing.T) {
	var calls atomic.Int32
	ts := newBranchServer(&calls)
	defer ts.Close()

	cases := []struct{ app, header, event, ref string }{
		{"app", "X-GitHub-Event", "push", "refs/heads/main"},
		{"app", "X-Gitlab-Event", "Push Hook", "refs/heads/main"},
		{"anybranch", "X-GitHub-Event", "push", "refs/heads/feature"},
	}
	for _, tc := range cases {
		status, answer := postEvent(t, ts.URL+"/webhook/"+tc.app, tc.header, tc.event, `{"ref":"`+tc.ref+`"}`)
		if status != http.StatusAccepted {
			t.Errorf("%s %s %s: got %d %q, want 202", tc.app, tc.event, tc.ref, status, answer)
		}
	}
}
