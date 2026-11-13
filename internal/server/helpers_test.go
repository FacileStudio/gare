package server

import (
	"bytes"
	"context"
	"net/http"
)

const testSecret = "test-secret-value"

func newSignedServer(handler func(ctx context.Context, appName string) error) *Server {
	return New(Config{Secret: testSecret, DeployHandler: handler})
}

func signedPost(targetURL string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature-256", computeSignature(testSecret, body))
	return http.DefaultClient.Do(req)
}
