package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	disclosureContext   = "https://irma.app/ld/request/disclosure/v2"
	irmaSessionPath     = "/session"
	headerAuthorization = "Authorization"
	headerContentType   = "Content-Type"
	contentTypeJSON     = "application/json"
	irmaTimeout         = 15 * time.Second
	maxIrmaResponse     = 1 << 20
)

// ageAttributes are the over-18 attributes of the credentials that can prove
// the age. The session request asks for any one of them, and the verification
// accepts the same set.
var ageAttributes = []string{
	"pbdf.pbdf.passport.over18",
	"pbdf.pbdf.idcard.over18",
	"pbdf.pbdf.drivinglicence.over18",
	"pbdf.nijmegen.ageLimits.over18",
	"pbdf.gemeente.personalData.over18",
	"irma-demo.gemeente.personalData.over18",
}

type disclosureRequest struct {
	Context  string       `json:"@context"`
	Disclose [][][]string `json:"disclose"`
}

type irmaClient struct {
	baseURL  string
	apiToken string
	http     *http.Client
}

func newIrmaClient(baseURL, apiToken string) *irmaClient {
	return &irmaClient{
		baseURL:  baseURL,
		apiToken: apiToken,
		http:     &http.Client{Timeout: irmaTimeout},
	}
}

// startAgeSession starts the over-18 disclosure session and returns the IRMA
// server's response, which the frontend needs to continue the session.
func (c *irmaClient) startAgeSession(ctx context.Context) ([]byte, error) {
	body, err := json.Marshal(ageRequest())
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+irmaSessionPath, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set(headerContentType, contentTypeJSON)
	if c.apiToken != "" {
		req.Header.Set(headerAuthorization, c.apiToken)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("irma server returned status %d", resp.StatusCode)
	}

	return io.ReadAll(io.LimitReader(resp.Body, maxIrmaResponse))
}

func ageRequest() disclosureRequest {
	alternatives := make([][]string, 0, len(ageAttributes))
	for _, id := range ageAttributes {
		alternatives = append(alternatives, []string{id})
	}

	return disclosureRequest{
		Context:  disclosureContext,
		Disclose: [][][]string{alternatives},
	}
}
