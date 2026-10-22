# Yivi Casino

Yivi Casino is a demo of an online casino, showcasing age verification using IRMA.

It showcases the following features:

* Age verification of attributes.
* Granting access to a resource only after successful Yivi (age) attribute verification.

The server is written in Go. It serves the page and has two endpoints:

* `/api/session?type=verification` starts an over-18 disclosure session at the IRMA server.
* `POST /api/verifysession` checks the signed result JWT. It answers `200 {"success":true}` for a valid over-18 disclosure and `403` otherwise. The frontend shows the under-18 message on a 403.

## Configuration

| Environment variable    | Description                                                                 |
|-------------------------|-----------------------------------------------------------------------------|
| `IRMA_SERVER_URL`       | URL of the IRMA server. Defaults to `http://localhost:8088`.                |
| `IRMA_SERVER_API_TOKEN` | Access token of the IRMA server, if it requires one.                        |
| `IRMA_SERVER_PUBLICKEY` | Path to the PEM public key of the IRMA server. Required.                    |

The server listens on port 8080. `/` returns 200 and can be used for health probes.

## Installation

### Without Docker

* Install Go (see `go.mod` for the version) and Node.js.
* `npm ci && npm run vendor` copies the frontend libraries to `www/vendor`. They are embedded in the binary, so run this before `go build`.
* `IRMA_SERVER_PUBLICKEY=data/demo-publickey.pem go run .`
* You will also need to run an [IRMA server](https://github.com/privacybydesign/irmago).
* Note that demo keys provided are only for local testing. Yivi Casino needs [IRMA server's](https://github.com/privacybydesign/irmago) public key and the IRMA server will need it's corresponding private key, which can be provided using `--jwt-privkey-file` flag.

### With Docker

You can use the `docker compose up --build` to build and test the project in one go. 

## Tests

`go vet ./... && go test ./...`
