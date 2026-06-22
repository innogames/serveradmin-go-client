package adminapi

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const (
	version   = "4.9.0"
	userAgent = "Adminapi Go Client " + version
)

// NewClientFromEnv builds a Client from the SERVERADMIN_* environment variables,
// applying the legacy auth precedence SERVERADMIN_KEY_PATH > SSH_AUTH_SOCK >
// SERVERADMIN_TOKEN. It is a convenience for env-configured deployments (such as
// the CLI); prefer NewClient with an explicit Config when you control the
// configuration, especially in multi-tenant processes.
func NewClientFromEnv() (*Client, error) {
	cfg, err := configFromEnv()
	if err != nil {
		return nil, err
	}
	return NewClient(cfg)
}

// configFromEnv builds a Config from the SERVERADMIN_* environment variables.
//
// This is the only place that applies the legacy ambient auth precedence:
// SERVERADMIN_KEY_PATH > SSH_AUTH_SOCK > SERVERADMIN_TOKEN. The SSH agent
// (SSH_AUTH_SOCK) is resolved here into a concrete ssh.Signer, as NewClient
// itself does not consult the agent.
func configFromEnv() (Config, error) {
	cfg := Config{}

	baseURL := os.Getenv("SERVERADMIN_BASE_URL")
	if baseURL == "" {
		return cfg, errors.New("env var SERVERADMIN_BASE_URL not set")
	}
	cfg.BaseURL = baseURL

	if privateKeyPath, ok := os.LookupEnv("SERVERADMIN_KEY_PATH"); ok && privateKeyPath != "" {
		cfg.KeyPath = privateKeyPath
	} else if authSock, ok := os.LookupEnv("SSH_AUTH_SOCK"); ok && authSock != "" {
		signer, err := agentSigner(authSock)
		if err != nil {
			return cfg, err
		}
		cfg.SSHSigner = signer
	}

	if cfg.KeyPath == "" && cfg.SSHSigner == nil {
		cfg.Token = os.Getenv("SERVERADMIN_TOKEN")
	}

	if cfg.Token == "" && cfg.KeyPath == "" && cfg.SSHSigner == nil {
		return cfg, errors.New("no authentication method found: set SERVERADMIN_TOKEN/SERVERADMIN_KEY_PATH/SSH_AUTH_SOCK")
	}

	return cfg, nil
}

// agentSigner connects to the SSH agent at authSock and returns the first signer
// that can produce a signature.
func agentSigner(authSock string) (ssh.Signer, error) {
	var dialer net.Dialer
	sock, err := dialer.DialContext(context.Background(), "unix", authSock)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}
	signers, err := agent.NewClient(sock).Signers()
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH agent signers: %w", err)
	}
	for _, signer := range signers {
		if _, err := signer.Sign(rand.Reader, []byte("test")); err == nil {
			return signer, nil
		}
	}
	return nil, errors.New("no usable signer found in SSH agent")
}
