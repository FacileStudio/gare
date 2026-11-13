package daemon

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

// fakeIdP is an in-process OIDC provider shaped like porte's own test
// provider: real RS256 tokens behind a real JWKS, so porte's verification is
// the code under test. email is what it asserts about whoever signs in.
type fakeIdP struct {
	server *httptest.Server
	key    *rsa.PrivateKey
	mu     sync.Mutex
	codes  map[string]url.Values
	email  string
}

// newFakeIdP starts the provider and stops it with the test.
func newFakeIdP(t *testing.T) *fakeIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	idp := &fakeIdP{key: key, codes: map[string]url.Values{}, email: "ops@example.test"}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", idp.discovery)
	mux.HandleFunc("/keys", idp.jwks)
	mux.HandleFunc("/authorize", idp.authorize)
	mux.HandleFunc("/token", idp.token)
	idp.server = httptest.NewServer(mux)
	t.Cleanup(idp.server.Close)
	return idp
}

// discovery serves the document porte's oidc.New reads at boot.
func (p *fakeIdP) discovery(w http.ResponseWriter, _ *http.Request) {
	issuer := p.server.URL
	writeJSON(w, map[string]any{
		"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token",
		"jwks_uri": issuer + "/keys", "response_types_supported": []string{"code"},
		"subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"},
	})
}

// jwks publishes the signing key.
func (p *fakeIdP) jwks(w http.ResponseWriter, _ *http.Request) {
	pub := p.key.Public().(*rsa.PublicKey)
	writeJSON(w, map[string]any{"keys": []map[string]any{{
		"kty": "RSA", "alg": "RS256", "use": "sig", "kid": "k1",
		"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}}})
}

// authorize plays the person approving the sign-in: it sends the browser
// straight back to the app's callback with a code and the app's state.
func (p *fakeIdP) authorize(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	code := rand.Text()
	p.mu.Lock()
	p.codes[code] = query
	p.mu.Unlock()
	target, _ := url.Parse(query.Get("redirect_uri"))
	target.RawQuery = url.Values{"code": {code}, "state": {query.Get("state")}}.Encode()
	http.Redirect(w, r, target.String(), http.StatusFound)
}

// token trades a code for an ID token carrying the nonce porte sent, and
// enforces PKCE so a flow that dropped it fails here.
func (p *fakeIdP) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	p.mu.Lock()
	request, ok := p.codes[r.PostForm.Get("code")]
	delete(p.codes, r.PostForm.Get("code"))
	email := p.email
	p.mu.Unlock()
	sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
	if !ok || base64.RawURLEncoding.EncodeToString(sum[:]) != request.Get("code_challenge") {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
		return
	}
	claims := map[string]any{
		"iss": p.server.URL, "sub": "sub-" + email, "aud": request.Get("client_id"), "nonce": request.Get("nonce"),
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
		"email": email, "email_verified": true, "name": "Ops Person",
	}
	writeJSON(w, map[string]any{
		"access_token": "access", "token_type": "Bearer", "expires_in": 3600, "id_token": p.sign(claims),
	})
}

// sign is a compact RS256 JWS over claims.
func (p *fakeIdP) sign(claims map[string]any) string {
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "k1", "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, _ := rsa.SignPKCS1v15(rand.Reader, p.key, crypto.SHA256, digest[:])
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

// writeJSON answers value as JSON, or a 500 a test will notice.
func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// setEmail changes who signs in next.
func (p *fakeIdP) setEmail(email string) {
	p.mu.Lock()
	p.email = email
	p.mu.Unlock()
}
