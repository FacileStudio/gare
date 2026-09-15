package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
)

func (s *Server) verifyAuth(r *http.Request, body []byte) bool {
	if gitlabToken := r.Header.Get("X-Gitlab-Token"); gitlabToken != "" {
		return subtle.ConstantTimeCompare([]byte(gitlabToken), []byte(s.Secret)) == 1
	}

	hubSig := r.Header.Get("X-Hub-Signature-256")
	if strings.HasPrefix(hubSig, "sha256=") {
		return verifyHubSignature(hubSig[len("sha256="):], s.Secret, body)
	}

	return false
}

func verifyHubSignature(actualHex, secret string, body []byte) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedHex := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(actualHex)), []byte(expectedHex))
}
