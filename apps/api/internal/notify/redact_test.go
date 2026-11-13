package notify

import (
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
)

func TestFromEnvNeedsBothURLAndSecret(t *testing.T) {
	t.Setenv("ANTENNE_URL", "http://antenne.test")
	t.Setenv("ANTENNE_INSTANCE", "")
	t.Setenv("ANTENNE_SECRET", "")
	if FromEnv() != nil {
		t.Error("a URL without a secret must leave alerts off")
	}
	var off *Notifier
	off.Notify(finished(deploys.StatusFailed, "boom", nil), noWarn(t))
}

func TestRedactDropsSecretLinesAndCaps(t *testing.T) {
	msg := "build failed\nGITHUB_TOKEN=ghp_abc\n  export DB_PASSWORD = hunter2\napi_key=xyz\nPORT=8000"
	got := Redact(msg)
	if got != "build failed\nPORT=8000" {
		t.Errorf("Redact = %q", got)
	}
	if long := Redact(strings.Repeat("é", 3000)); len([]rune(long)) != maxMessage {
		t.Errorf("Redact kept %d runes, want %d", len([]rune(long)), maxMessage)
	}
}
