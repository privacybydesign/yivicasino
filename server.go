package main

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
)

const (
	sessionTypeParam        = "type"
	sessionTypeVerification = "verification"
	maxVerifyBody           = 64 << 10
)

type verdict struct {
	Success bool `json:"success"`
}

var (
	verdictAllowed = verdict{Success: true}
	verdictDenied  = verdict{Success: false}
)

type server struct {
	irma     *irmaClient
	verifier *tokenVerifier
}

func newHandler(irma *irmaClient, verifier *tokenVerifier, site fs.FS) http.Handler {
	s := &server{irma: irma, verifier: verifier}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/session", s.handleSession)
	mux.HandleFunc("POST /api/session", s.handleSession)
	mux.HandleFunc("POST /api/verifysession", s.handleVerify)
	mux.Handle("/", http.FileServerFS(site))

	return mux
}

// handleSession starts the over-18 disclosure session at the IRMA server and
// passes the response through.
func (s *server) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get(sessionTypeParam) != sessionTypeVerification {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	response, err := s.irma.startAgeSession(r.Context())
	if err != nil {
		slog.Error("failed to start session", "error", err)
		http.Error(w, "Failed to start session", http.StatusBadGateway)
		return
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	_, _ = w.Write(response)
}

// handleVerify lets the user in only when the result JWT is signed by our IRMA
// server, the disclosure was valid and the age attribute says over 18. An
// invalid JWT gets a 403 with an empty body, which the frontend takes as the
// under-18 case.
func (s *server) handleVerify(w http.ResponseWriter, r *http.Request) {
	token := readToken(w, r)
	if token == "" {
		http.Error(w, "No token provided", http.StatusBadRequest)
		return
	}

	result, err := s.verifier.verify(token)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if !isProofValid(result) || !isAgeAllowed(result.Disclosed) {
		writeVerdict(w, http.StatusForbidden, verdictDenied)
		return
	}

	writeVerdict(w, http.StatusOK, verdictAllowed)
}

func readToken(w http.ResponseWriter, r *http.Request) string {
	var body struct {
		Token string `json:"token"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxVerifyBody)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return ""
	}

	return body.Token
}

func writeVerdict(w http.ResponseWriter, status int, v verdict) {
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
