package adminapi

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const (
	version   = "4.9.0"
	userAgent = "Adminapi Go Client " + version
)

type config struct {
	baseURL    string
	apiVersion string
	authToken  []byte
	sshSigner  ssh.Signer
}

// getConfig returns the configuration for the API client. Loading config only once
var getConfig = sync.OnceValues(loadConfig)

// loadConfig returns the configuration for the API client
var loadConfig = func() (config, error) {
	cfg := config{
		apiVersion: version,
	}

	baseURL := os.Getenv("SERVERADMIN_BASE_URL")
	if baseURL == "" {
		return cfg, errors.New("env var SERVERADMIN_BASE_URL not set")
	}
	cfg.baseURL = strings.TrimSuffix(baseURL, "/api")

	if privateKeyPath, ok := os.LookupEnv("SERVERADMIN_KEY_PATH"); ok {
		keyBytes, err := os.ReadFile(privateKeyPath)
		if err != nil {
			return cfg, fmt.Errorf("failed to read private key from %s: %w", privateKeyPath, err)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return cfg, fmt.Errorf("failed to parse private key: %w", err)
		}
		cfg.sshSigner = signer
	} else if authSock, ok := os.LookupEnv("SSH_AUTH_SOCK"); ok && authSock != "" {
		sock, err := net.Dial("unix", authSock)
		if err != nil {
			return cfg, fmt.Errorf("failed to connect to SSH agent: %w", err)
		}
		signers, err := agent.NewClient(sock).Signers()
		if err != nil {
			return cfg, fmt.Errorf("failed to get SSH agent signers: %w", err)
		}
		for _, signer := range signers {
			_, err := signer.Sign(rand.Reader, []byte("test"))
			if err == nil {
				cfg.sshSigner = signer
				break
			}
		}
	}

	if cfg.sshSigner == nil {
		cfg.authToken = []byte(os.Getenv("SERVERADMIN_TOKEN"))
	}

	if len(cfg.authToken) == 0 && cfg.sshSigner == nil {
		return cfg, errors.New("no authentication method found: set SERVERADMIN_TOKEN/SERVERADMIN_KEY_PATH/SSH_AUTH_SOCK")
	}

	return cfg, nil
}
