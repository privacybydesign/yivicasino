package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const (
	listenAddr        = ":8080"
	defaultIrmaURL    = "http://localhost:8088"
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second
)

type config struct {
	irmaServerURL string
	irmaAPIToken  string
	publicKeyPath string
}

func main() {
	if err := run(); err != nil {
		slog.Error("yivicasino stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	verifier, err := newVerifierFromFile(cfg.publicKeyPath)
	if err != nil {
		return err
	}

	site, err := publicSite()
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           newHandler(newIrmaClient(cfg.irmaServerURL, cfg.irmaAPIToken), verifier, site),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	slog.Info("listening", "addr", listenAddr)

	return srv.ListenAndServe()
}

func loadConfig() (config, error) {
	cfg := config{
		irmaServerURL: os.Getenv("IRMA_SERVER_URL"),
		irmaAPIToken:  os.Getenv("IRMA_SERVER_API_TOKEN"),
		publicKeyPath: os.Getenv("IRMA_SERVER_PUBLICKEY"),
	}

	if cfg.irmaServerURL == "" {
		cfg.irmaServerURL = defaultIrmaURL
	}

	if cfg.publicKeyPath == "" {
		return config{}, errors.New("IRMA_SERVER_PUBLICKEY is not set")
	}

	return cfg, nil
}
