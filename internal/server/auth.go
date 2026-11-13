package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
)

// verifyAuth accepts a request only when it proves knowledge of the server's secret. A server with
// no secret accepts nothing, so a missing secret can never leave the deploy endpoint open.
func (s *Server) verifyAuth(r *http.Request, body []byte) bool {
	if s.secret == "" {
		return false
	}
	if gitlabToken := r.Header.Get("X-Gitlab-Token"); gitlabToken != "" {
		return subtle.ConstantTimeCompare([]byte(gitlabToken), []byte(s.secret)) == 1
	}

	hubSig := r.Header.Get("X-Hub-Signature-256")
	if strings.HasPrefix(hubSig, "sha256=") {
		return verifyHubSignature(hubSig[len("sha256="):], s.secret, body)
	}

	return false
}

func verifyHubSignature(actualHex, secret string, body []byte) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedHex := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(actualHex)), []byte(expectedHex))
}
