package adminapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// mustClient builds a token-authenticated Client pointing at baseURL, failing
// the test if construction fails.
func mustClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	c, err := NewClient(Config{BaseURL: baseURL, Token: "test-token"})
	require.NoError(t, err)
	return c
}

func TestNewClientValidation(t *testing.T) {
	t.Run("missing BaseURL", func(t *testing.T) {
		_, err := NewClient(Config{Token: "tok"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "BaseURL is required")
	})

	t.Run("no auth method", func(t *testing.T) {
		_, err := NewClient(Config{BaseURL: "https://example.com"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no authentication method configured")
	})

	t.Run("token auth", func(t *testing.T) {
		c, err := NewClient(Config{BaseURL: "https://example.com", Token: "tok"})
		require.NoError(t, err)
		assert.Equal(t, "tok", string(c.authToken))
		assert.Nil(t, c.sshSigner)
	})

	t.Run("key path auth", func(t *testing.T) {
		c, err := NewClient(Config{BaseURL: "https://example.com", KeyPath: "testdata/test.key"})
		require.NoError(t, err)
		assert.NotNil(t, c.sshSigner)
		assert.Empty(t, c.authToken)
	})

	t.Run("explicit signer auth", func(t *testing.T) {
		keyBytes, err := os.ReadFile("testdata/test.key")
		require.NoError(t, err)
		signer, err := ssh.ParsePrivateKey(keyBytes)
		require.NoError(t, err)

		c, err := NewClient(Config{BaseURL: "https://example.com", SSHSigner: signer})
		require.NoError(t, err)
		assert.Equal(t, signer, c.sshSigner)
	})

	t.Run("signer takes precedence over token", func(t *testing.T) {
		keyBytes, err := os.ReadFile("testdata/test.key")
		require.NoError(t, err)
		signer, err := ssh.ParsePrivateKey(keyBytes)
		require.NoError(t, err)

		c, err := NewClient(Config{BaseURL: "https://example.com", SSHSigner: signer, Token: "tok"})
		require.NoError(t, err)
		assert.NotNil(t, c.sshSigner)
		assert.Empty(t, c.authToken, "token must be ignored when a signer is set")
	})

	t.Run("trims /api suffix", func(t *testing.T) {
		c, err := NewClient(Config{BaseURL: "https://example.com/api", Token: "tok"})
		require.NoError(t, err)
		assert.Equal(t, "https://example.com", c.baseURL)
	})

	t.Run("custom http client honored", func(t *testing.T) {
		custom := &http.Client{Timeout: 7 * time.Second}
		c, err := NewClient(Config{BaseURL: "https://example.com", Token: "tok", HTTPClient: custom})
		require.NoError(t, err)
		assert.Same(t, custom, c.httpClient)
	})

	t.Run("timeout applied to generated client", func(t *testing.T) {
		c, err := NewClient(Config{BaseURL: "https://example.com", Token: "tok", Timeout: 3 * time.Second})
		require.NoError(t, err)
		assert.Equal(t, 3*time.Second, c.httpClient.Timeout)
	})
}

// TestClientSendsOwnAuthHeaders verifies a token client signs requests with its
// own token and never consults global/env configuration.
func TestClientSendsOwnAuthHeaders(t *testing.T) {
	var gotAppID, gotToken, gotUserAgent, gotTimestamp string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAppID = r.Header.Get("X-Application")
		gotToken = r.Header.Get("X-SecurityToken")
		gotUserAgent = r.Header.Get("User-Agent")
		gotTimestamp = r.Header.Get("X-Timestamp")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","result":[{"object_id":1,"hostname":"a.local"}]}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, Token: "secret-token"})
	require.NoError(t, err)

	q := client.NewQuery(Filters{"hostname": "a.local"})
	servers, err := q.All(context.Background())
	require.NoError(t, err)
	require.Len(t, servers, 1)

	assert.Equal(t, calcAppID([]byte("secret-token")), gotAppID)
	assert.NotEmpty(t, gotToken)
	assert.Equal(t, userAgent, gotUserAgent)
	assert.NotEmpty(t, gotTimestamp)
}

// TestTwoClientsParallel is the acceptance test: a single process holds two
// clients with different BaseURL/Token and queries both concurrently. Each
// server must only ever see its own token's application id and return its own
// data. Run with -race to confirm there is no shared mutable state.
func TestTwoClientsParallel(t *testing.T) {
	newTarget := func(hostname, token string) (*httptest.Server, *Client) {
		wantAppID := calcAppID([]byte(token))
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Every request to this server must carry this server's token.
			assert.Equal(t, wantAppID, r.Header.Get("X-Application"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"success","result":[{"object_id":1,"hostname":"` + hostname + `"}]}`))
		}))
		client, err := NewClient(Config{BaseURL: srv.URL, Token: token})
		require.NoError(t, err)
		return srv, client
	}

	srvA, clientA := newTarget("a.example.com", "token-a")
	defer srvA.Close()
	srvB, clientB := newTarget("b.example.com", "token-b")
	defer srvB.Close()

	const iterations = 25
	var wg sync.WaitGroup
	run := func(client *Client, wantHostname string) {
		defer wg.Done()
		for range iterations {
			q := client.NewQuery(Filters{"hostname": wantHostname})
			servers, err := q.All(context.Background())
			if assert.NoError(t, err) && assert.Len(t, servers, 1) {
				assert.Equal(t, wantHostname, servers[0].GetString("hostname"))
			}
		}
	}

	wg.Add(2)
	go run(clientA, "a.example.com")
	go run(clientB, "b.example.com")
	wg.Wait()
}

func TestClientContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","result":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, Token: "tok"})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before issuing the request

	q := client.NewQuery(Filters{"hostname": "a.local"})
	_, err = q.All(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestTypedGetters(t *testing.T) {
	obj := &ServerObject{
		attributes: Attributes{
			"num_cpu":     float64(4),   // integers arrive as float64 from JSON
			"load_avg":    float64(1.5), // genuine float
			"int_field":   7,            // already an int
			"enabled":     true,
			"disabled":    false,
			"hostname":    "web01",
			"missing_int": nil,
		},
		oldValues: Attributes{},
	}

	// GetInt truncates floats and handles native ints.
	assert.Equal(t, 4, obj.GetInt("num_cpu"))
	assert.Equal(t, 1, obj.GetInt("load_avg"))
	assert.Equal(t, 7, obj.GetInt("int_field"))
	assert.Equal(t, 0, obj.GetInt("hostname"))
	assert.Equal(t, 0, obj.GetInt("absent"))

	// GetFloat preserves the fractional part that Get/GetInt would discard.
	assert.InEpsilon(t, 1.5, obj.GetFloat("load_avg"), 1e-9)
	assert.InEpsilon(t, 4.0, obj.GetFloat("num_cpu"), 1e-9)
	assert.InEpsilon(t, 7.0, obj.GetFloat("int_field"), 1e-9)
	assert.InDelta(t, 0.0, obj.GetFloat("hostname"), 1e-9)

	// GetBool type-asserts.
	assert.True(t, obj.GetBool("enabled"))
	assert.False(t, obj.GetBool("disabled"))
	assert.False(t, obj.GetBool("missing_int"))

	// Get still performs the legacy lossy float64->int conversion.
	assert.Equal(t, 1, obj.Get("load_avg"))
}
