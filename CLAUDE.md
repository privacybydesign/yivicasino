# yivicasino — repo notes

Go server with a static frontend (`www/`). It starts an over-18 disclosure
session at an IRMA server and checks the signed result JWT. Port of the former
PHP implementation (CERT-EU advisory 2026-018).

## Build & test

- `go vet ./... && go test ./...`. The Go version is set in `go.mod`.
- `npm ci && npm run vendor` copies the npm frontend assets into `www/vendor`,
  which is git-ignored. The Dockerfile does the same in its Node stage, before
  `go build`. Tests do not need it.
- Run locally: `IRMA_SERVER_PUBLICKEY=data/demo-publickey.pem go run .`, or
  `docker compose up --build` for an IRMA server as well.

## Gotchas

- `POST /api/verifysession` keeps a contract the frontend relies on: a bad or
  invalid JWT gets a **403 with an empty body**, and a failed age check gets
  `403 {"success":false}`. The frontend shows the under-18 message on any 403.
- JWTs are accepted only as RS256 and only with the key in
  `IRMA_SERVER_PUBLICKEY`. Never take the algorithm or key from the token.
- `ageAttributes` in `irma.go` drives both the session request and the age check.
- Only `www/` is embedded (`site.go`), so nothing else in the repository can be
  served. `static_test.go` checks this. Keep secrets and config out of `www/`.
