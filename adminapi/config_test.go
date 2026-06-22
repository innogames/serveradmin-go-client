package adminapi

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigFromEnv(t *testing.T) {
	// without SERVERADMIN_BASE_URL set
	t.Setenv("SERVERADMIN_BASE_URL", "")
	_, err := configFromEnv()
	require.Error(t, err, "env var SERVERADMIN_BASE_URL not set")

	// spawn mocked serveradmin server
	server := httptest.NewServer(nil)
	defer server.Close()
	t.Setenv("SERVERADMIN_BASE_URL", server.URL)

	t.Run("load static token", func(t *testing.T) {
		// Unset SSH-related env vars to prevent SSH agent from taking precedence
		t.Setenv("SSH_AUTH_SOCK", "")
		t.Setenv("SERVERADMIN_KEY_PATH", "")
		t.Setenv("SERVERADMIN_TOKEN", "jolo")

		cfg, err := configFromEnv()
		require.NoError(t, err)
		assert.Nil(t, cfg.SSHSigner)
		assert.Empty(t, cfg.KeyPath)
		assert.Equal(t, "jolo", cfg.Token)

		client, err := NewClient(cfg)
		require.NoError(t, err)
		assert.Nil(t, client.sshSigner)
		assert.Equal(t, "jolo", string(client.authToken))
	})

	t.Run("load valid private key", func(t *testing.T) {
		t.Setenv("SSH_AUTH_SOCK", "")
		t.Setenv("SERVERADMIN_KEY_PATH", "testdata/test.key")

		cfg, err := configFromEnv()
		require.NoError(t, err)
		assert.Equal(t, "testdata/test.key", cfg.KeyPath)
		assert.Empty(t, cfg.Token)

		client, err := NewClient(cfg)
		require.NoError(t, err)
		assert.NotNil(t, client.sshSigner)
		assert.Empty(t, client.authToken)
	})

	t.Run("load invalid private key", func(t *testing.T) {
		t.Setenv("SSH_AUTH_SOCK", "")
		t.Setenv("SERVERADMIN_KEY_PATH", "testdata/nope.key")

		cfg, err := configFromEnv()
		require.NoError(t, err)

		// The file is read and parsed by NewClient, so the error surfaces there.
		_, err = NewClient(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read private key from testdata/nope.key")
	})
}
