// Package notify tells Antenne, the Facile suite's alert bus, when a deploy fails or rolls back.
// It posts to Antenne's generic webhook rather than joining the antenne-client pool: the pool is a
// WebSocket bus between apps whose events never reach Antenne's delivery targets, while a webhook
// event is routed to Matrix, Discord or ntfy like every other alert.
package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
)

// timeout bounds one alert, so an Antenne that does not answer costs a deploy at most this long.
const timeout = 5 * time.Second

// webhookPath is where Antenne's generic provider listens by default.
const webhookPath = "/webhook/generic"

// signatureHeader is the header Antenne's generic provider reads the body signature from by default.
const signatureHeader = "x-generic-signature-256"

// Notifier posts deploy alerts to one Antenne instance. A nil Notifier is valid and sends nothing,
// which is what an unconfigured host gets.
type Notifier struct {
	url    string
	secret string
	host   string
	client *http.Client
}

// FromEnv returns the Notifier the environment configures, using the names antenne-client reads:
// ANTENNE_URL, or its older synonym ANTENNE_INSTANCE, and ANTENNE_SECRET. It returns nil unless
// both a URL and a secret are set, because an unsigned alert is one Antenne should refuse.
func FromEnv() *Notifier {
	base := firstEnv("ANTENNE_URL", "ANTENNE_INSTANCE")
	secret := strings.TrimSpace(os.Getenv("ANTENNE_SECRET"))
	if base == "" || secret == "" {
		return nil
	}
	host, _ := os.Hostname()
	return &Notifier{
		url:    strings.TrimRight(base, "/") + webhookPath,
		secret: secret,
		host:   host,
		client: &http.Client{Timeout: timeout},
	}
}

// Notify posts the alert a finished deploy calls for and reports a failure to send through warn.
// A succeeded deploy, and any deploy on a nil Notifier, sends nothing. It returns once the post is
// done; a caller that must not wait runs it in a goroutine.
func (n *Notifier) Notify(d deploys.Deploy, warn func(string)) {
	if n == nil {
		return
	}
	body, ok := alertBody(d, n.host)
	if !ok {
		return
	}
	if err := n.post(body); err != nil {
		warn(fmt.Sprintf("Could not alert Antenne about deploy %s (%v)", d.ID, err))
	}
}

// post sends one signed alert and treats any answer outside 2xx as a failure.
func (n *Notifier) post(body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, []byte(n.secret))
	mac.Write(body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(signatureHeader, "sha256="+hex.EncodeToString(mac.Sum(nil)))
	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("antenne answered %s", resp.Status)
	}
	return nil
}

// alertBody renders a finished deploy as the payload Antenne's generic provider reads. Antenne has
// no severity field, so severity travels in data next to the deploy's facts. It reports false for a
// deploy that calls for no alert.
func alertBody(d deploys.Deploy, host string) ([]byte, bool) {
	severity, verb, ok := classify(d.Status)
	if !ok {
		return nil, false
	}
	data := map[string]any{
		"severity": severity, "app": d.App, "deploy_id": d.ID,
		"commit": d.Commit, "trigger": string(d.Trigger), "host": host,
	}
	if d.RolledBackTo != nil {
		data["rolled_back_to"] = *d.RolledBackTo
	}
	message := ""
	if d.Error != nil {
		message = Redact(*d.Error)
	}
	body, err := json.Marshal(map[string]any{
		"title": fmt.Sprintf("Deploy of %s %s on %s", d.App, verb, host), "message": message,
		"event": "gare.deploy." + strings.ReplaceAll(string(d.Status), "-", "_"), "source": "gare", "data": data,
	})
	return body, err == nil
}

// classify maps a deploy's final status to the alert's severity and the verb its title uses.
func classify(status deploys.Status) (string, string, bool) {
	switch status {
	case deploys.StatusFailed:
		return "error", "failed", true
	case deploys.StatusRolledBack:
		return "warning", "rolled back", true
	}
	return "", "", false
}

// firstEnv returns the first of names set to a non-blank value.
func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}
