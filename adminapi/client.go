package adminapi

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Config holds the explicit, per-instance configuration for a Client.
//
// Authentication is selected explicitly from the fields below, in this order:
// SSHSigner, then KeyPath, then Token. No environment variables are consulted,
// so an ambient SSH_AUTH_SOCK can never override an explicitly configured token.
type Config struct {
	// BaseURL is the Serveradmin base URL (required). A trailing "/api" is trimmed.
	BaseURL string

	// Token enables security-token authentication (HMAC-SHA1).
	Token string

	// SSHSigner enables SSH-signature authentication using a pre-built signer.
	// This takes precedence over KeyPath and Token.
	SSHSigner ssh.Signer

	// KeyPath is the path to a private key file used for SSH-signature
	// authentication. Used only when SSHSigner is nil.
	KeyPath string

	// HTTPClient is the HTTP client used for all requests. If nil, a dedicated
	// client is created using Timeout.
	HTTPClient *http.Client

	// Timeout is applied to the generated HTTP client. Ignored when HTTPClient
	// is provided. A zero value means no timeout.
	Timeout time.Duration
}

// Client is a per-instance Serveradmin API client. It carries its own
// configuration and *http.Client and is safe for concurrent use: all fields are
// set once at construction and never mutated afterwards.
type Client struct {
	baseURL    string
	authToken  []byte
	sshSigner  ssh.Signer
	httpClient *http.Client
}

// NewClient builds a Client from an explicit Config. It performs no environment
// reads and keeps no global state, so multiple clients with different base URLs
// and credentials can coexist and be used concurrently in the same process.
func NewClient(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("config: BaseURL is required")
	}

	c := &Client{
		baseURL: strings.TrimSuffix(cfg.BaseURL, "/api"),
	}

	switch {
	case cfg.SSHSigner != nil:
		c.sshSigner = cfg.SSHSigner
	case cfg.KeyPath != "":
		keyBytes, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key from %s: %w", cfg.KeyPath, err)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		c.sshSigner = signer
	case cfg.Token != "":
		c.authToken = []byte(cfg.Token)
	default:
		return nil, errors.New("config: no authentication method configured: set Token, SSHSigner or KeyPath")
	}

	if cfg.HTTPClient != nil {
		c.httpClient = cfg.HTTPClient
	} else {
		c.httpClient = &http.Client{Timeout: cfg.Timeout}
	}

	return c, nil
}
