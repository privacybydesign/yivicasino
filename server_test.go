package main

import (
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testAPIToken     = "secret-token"
	irmaSessionReply = `{"token":"abc","sessionPtr":{"u":"https://irma.example/irma/session/xyz","irmaqr":"disclosing"}}`

	startSessionTarget = "/api/session?type=verification"
	verifyPath         = "/api/verifysession"
	bodyAllowed        = `{"success":true}`
	bodyDenied         = `{"success":false}`

	// Spelled out instead of reusing the production constants, so a typo in
	// those fails a test.
	wireAuthorization     = "Authorization"
	wireContentType       = "Content-Type"
	wireDisclosureContext = "https://irma.app/ld/request/disclosure/v2"
)

type recordedRequest struct {
	method        string
	path          string
	authorization string
	contentType   string
	body          string
}

// fakeIrma stands in for the IRMA server and records what it receives.
type fakeIrma struct {
	*httptest.Server
	mu       sync.Mutex
	requests []recordedRequest
	status   int
}

func newFakeIrma(t *testing.T) *fakeIrma {
	t.Helper()

	f := &fakeIrma{status: http.StatusOK}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		f.mu.Lock()
		defer f.mu.Unlock()

		f.requests = append(f.requests, recordedRequest{
			method:        r.Method,
			path:          r.URL.Path,
			authorization: r.Header.Get(wireAuthorization),
			contentType:   r.Header.Get(wireContentType),
			body:          string(body),
		})
		w.WriteHeader(f.status)
		_, _ = io.WriteString(w, irmaSessionReply)
	}))
	t.Cleanup(f.Close)

	return f
}

type testApp struct {
	handler http.Handler
	irma    *fakeIrma
	key     *rsa.PrivateKey
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()

	key := newTestKey(t)
	verifier, err := newTokenVerifier(publicKeyPEM(t, key))
	if err != nil {
		t.Fatal(err)
	}

	irma := newFakeIrma(t)
	site := fstest.MapFS{"index.html": {Data: []byte("<h1>Yivi Casino</h1>")}}

	return &testApp{
		handler: newHandler(newIrmaClient(irma.URL, testAPIToken), verifier, site),
		irma:    irma,
		key:     key,
	}
}

func (a *testApp) do(method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)

	return rec
}

func (a *testApp) verify(token string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"token": token})

	return a.do(http.MethodPost, verifyPath, string(body))
}

func TestSessionStartsAgeCheck(t *testing.T) {
	app := newTestApp(t)

	rec := app.do(http.MethodGet, "/api/session?type=verification&0.5", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	if rec.Body.String() != irmaSessionReply {
		t.Errorf("body = %q, want the IRMA server response passed through", rec.Body.String())
	}

	if len(app.irma.requests) != 1 {
		t.Fatalf("IRMA server got %d requests, want 1", len(app.irma.requests))
	}

	got := app.irma.requests[0]
	if got.method != http.MethodPost || got.path != "/session" {
		t.Errorf("IRMA request = %s %s, want POST /session", got.method, got.path)
	}

	if got.authorization != testAPIToken {
		t.Errorf("Authorization = %q, want the API token", got.authorization)
	}

	if got.contentType != contentTypeJSON {
		t.Errorf("Content-Type = %q, want application/json", got.contentType)
	}

	var request disclosureRequest
	if err := json.Unmarshal([]byte(got.body), &request); err != nil {
		t.Fatal(err)
	}

	if request.Context != wireDisclosureContext {
		t.Errorf("@context = %q", request.Context)
	}

	if len(request.Disclose) != 1 || len(request.Disclose[0]) != len(ageAttributes) {
		t.Fatalf("disclose = %v, want one disjunction with %d alternatives", request.Disclose, len(ageAttributes))
	}

	for i, id := range ageAttributes {
		if len(request.Disclose[0][i]) != 1 || request.Disclose[0][i][0] != id {
			t.Errorf("alternative %d = %v, want [%s]", i, request.Disclose[0][i], id)
		}
	}
}

func TestSessionWithoutAPIToken(t *testing.T) {
	irma := newFakeIrma(t)
	client := newIrmaClient(irma.URL, "")

	if _, err := client.startAgeSession(t.Context()); err != nil {
		t.Fatal(err)
	}

	if got := irma.requests[0].authorization; got != "" {
		t.Errorf("Authorization = %q, want none", got)
	}
}

func TestSessionRejectsOtherTypes(t *testing.T) {
	for _, target := range []string{"/api/session", "/api/session?type=", "/api/session?type=issuance"} {
		t.Run(target, func(t *testing.T) {
			app := newTestApp(t)

			if rec := app.do(http.MethodGet, target, ""); rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", rec.Code)
			}

			if len(app.irma.requests) != 0 {
				t.Error("IRMA server was called, want no session started")
			}
		})
	}
}

func TestSessionUpstreamFailure(t *testing.T) {
	app := newTestApp(t)
	app.irma.status = http.StatusInternalServerError

	if rec := app.do(http.MethodGet, startSessionTarget, ""); rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502 when the IRMA server fails", rec.Code)
	}

	app.irma.Close()

	if rec := app.do(http.MethodGet, startSessionTarget, ""); rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502 when the IRMA server is unreachable", rec.Code)
	}
}

func TestVerifyAllowsOver18(t *testing.T) {
	for _, value := range []string{"yes", "ja", "Yes"} {
		t.Run(value, func(t *testing.T) {
			app := newTestApp(t)
			claims := resultClaims("VALID", [][]disclosedAttribute{{attr("pbdf.pbdf.passport.over18", value)}})

			rec := app.verify(signToken(t, jwt.SigningMethodRS256, app.key, claims))

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}

			if got := strings.TrimSpace(rec.Body.String()); got != bodyAllowed {
				t.Errorf("body = %q", got)
			}
		})
	}
}

func TestVerifyDeniesUnder18(t *testing.T) {
	app := newTestApp(t)
	claims := resultClaims("VALID", [][]disclosedAttribute{{attr("pbdf.pbdf.passport.over18", "no")}})

	rec := app.verify(signToken(t, jwt.SigningMethodRS256, app.key, claims))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}

	if got := strings.TrimSpace(rec.Body.String()); got != bodyDenied {
		t.Errorf("body = %q", got)
	}
}

func TestVerifyDeniesBadProof(t *testing.T) {
	disclosed := [][]disclosedAttribute{{attr("pbdf.pbdf.passport.over18", "yes")}}

	for _, status := range []string{"INVALID", "EXPIRED", "MISSING_ATTRIBUTES", "valid", ""} {
		t.Run(status, func(t *testing.T) {
			app := newTestApp(t)

			rec := app.verify(signToken(t, jwt.SigningMethodRS256, app.key, resultClaims(status, disclosed)))

			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403", rec.Code)
			}

			if got := strings.TrimSpace(rec.Body.String()); got != bodyDenied {
				t.Errorf("body = %q", got)
			}
		})
	}
}

func TestVerifyDeniesNoDisclosure(t *testing.T) {
	app := newTestApp(t)
	claims := jwt.MapClaims{"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(), "proofStatus": "VALID"}

	rec := app.verify(signToken(t, jwt.SigningMethodRS256, app.key, claims))

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

// The frontend shows the under-18 message on a 403, so a bad JWT must give a
// 403 with an empty body.
func TestVerifyBadJWTGivesEmpty403(t *testing.T) {
	app := newTestApp(t)
	otherKey := newTestKey(t)

	expired := validClaims()
	expired["exp"] = time.Now().Add(-time.Hour).Unix()

	tests := []struct {
		name  string
		token string
	}{
		{"wrong key", signToken(t, jwt.SigningMethodRS256, otherKey, validClaims())},
		{"wrong algorithm", signToken(t, jwt.SigningMethodHS256, publicKeyPEM(t, app.key), validClaims())},
		{"alg none", signToken(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims())},
		{"expired", signToken(t, jwt.SigningMethodRS256, app.key, expired)},
		{"malformed", "not-a-jwt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := app.verify(tt.token)

			if rec.Code != http.StatusForbidden {
				t.Errorf("status = %d, want 403", rec.Code)
			}

			if rec.Body.Len() != 0 {
				t.Errorf("body = %q, want empty", rec.Body.String())
			}
		})
	}
}

func TestVerifyWithoutToken(t *testing.T) {
	tests := map[string]string{
		"no body":       "",
		"not JSON":      "token",
		"no token":      `{}`,
		"empty token":   `{"token":""}`,
		"numeric token": `{"token":42}`,
		"null token":    `{"token":null}`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			rec := newTestApp(t).do(http.MethodPost, verifyPath, body)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", rec.Code)
			}
		})
	}
}

func TestVerifyRefusesGet(t *testing.T) {
	rec := newTestApp(t).do(http.MethodGet, verifyPath, "")

	if rec.Code == http.StatusOK {
		t.Errorf("status = %d, want GET to be refused", rec.Code)
	}
}
