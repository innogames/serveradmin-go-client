package adminapi

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// make a test without SERVERADMIN_BASE_URL set
	t.Setenv("SERVERADMIN_BASE_URL", "")
	_, err := loadConfig()
	require.Error(t, err, "env var SERVERADMIN_BASE_URL not set")

	// spawn mocked serveradmin server
	server := httptest.NewServer(nil)
	defer server.Close()
	t.Setenv("SERVERADMIN_BASE_URL", server.URL)

	t.Run("load static token", func(t *testing.T) {
		t.Setenv("SERVERADMIN_TOKEN", "jolo")
		cfg, err := loadConfig()

		require.NoError(t, err)
		assert.Nil(t, cfg.sshSigner)
		assert.Equal(t, "jolo", string(cfg.authToken))
	})

	t.Run("load valid private key", func(t *testing.T) {
		t.Setenv("SERVERADMIN_KEY_PATH", "testdata/test.key")
		cfg, err := loadConfig()

		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Empty(t, cfg.authToken)
	})

	t.Run("load invalid private Key", func(t *testing.T) {
		t.Setenv("SERVERADMIN_KEY_PATH", "testdata/nope.key")
		_, err := loadConfig()

		assert.Error(t, err, "failed to read private key from testdata/nope.key: open testdata/nope.key: no such file or directory")
	})
}
