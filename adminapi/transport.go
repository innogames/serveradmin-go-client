package adminapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // SHA1 is required by the protocol
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	apiEndpointQuery     = "/api/dataset/query"
	apiEndpointNewObject = "/api/dataset/new_object"
	apiEndpointCommit    = "/api/dataset/commit"
)

func sendRequest(endpoint string, postData any) (*http.Response, error) {
	config, err := getConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	postStr, err := json.Marshal(postData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, config.baseURL+endpoint, bytes.NewBuffer(postStr))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	now := time.Now().Unix()
	req.Header.Set("Content-Type", "application/x-json")
	req.Header.Set("X-Timestamp", strconv.FormatInt(now, 10))
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Encoding", "gzip")

	if config.sshSigner != nil {
		// sign with private key or SSH agent
		messageToSign := calcMessage(now, postStr)
		signature, sigErr := config.sshSigner.Sign(rand.Reader, messageToSign)
		if sigErr != nil {
			return nil, fmt.Errorf("failed to sign request: %w", sigErr)
		}
		publicKey := base64.StdEncoding.EncodeToString(config.sshSigner.PublicKey().Marshal())
		sshSignature := base64.StdEncoding.EncodeToString(ssh.Marshal(signature))

		req.Header.Set("X-PublicKeys", publicKey)
		req.Header.Set("X-Signatures", sshSignature)
	} else if len(config.authToken) > 0 {
		req.Header.Set("X-SecurityToken", calcSecurityToken(config.authToken, now, postStr))
		req.Header.Set("X-Application", calcAppID(config.authToken))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()

		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("HTTP error %d %s (failed to read error details: %w)",
				resp.StatusCode, http.StatusText(resp.StatusCode), readErr)
		}

		var nestedErrorResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if jsonErr := json.Unmarshal(bodyBytes, &nestedErrorResp); jsonErr == nil && nestedErrorResp.Error.Message != "" {
			return nil, fmt.Errorf("HTTP error %d %s: %s",
				resp.StatusCode, http.StatusText(resp.StatusCode), nestedErrorResp.Error.Message)
		}

		// If body is empty, just return the status code
		return nil, fmt.Errorf("HTTP error %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	// If the server responded with gzip encoding, wrap the response body accordingly.
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}

		// Replace the resp.Body with our gzip-aware ReadCloser
		resp.Body = &gzipReadCloser{
			Reader: gz,
			body:   resp.Body,
			gz:     gz,
		}
	}

	return resp, nil
}

// gzipReadCloser wraps a gzip.Reader so that
// closing it also closes the underlying body.
type gzipReadCloser struct {
	io.Reader
	body io.Closer
	gz   *gzip.Reader
}

// Close closes the gzip.Reader and the underlying body.
func (grc *gzipReadCloser) Close() error {
	return errors.Join(grc.gz.Close(), grc.body.Close())
}

// calcSecurityToken calculates HMAC-SHA1 of timestamp:data
func calcSecurityToken(authToken []byte, timestamp int64, data []byte) string {
	mac := hmac.New(sha1.New, authToken)
	mac.Write(calcMessage(timestamp, data))

	return hex.EncodeToString(mac.Sum(nil))
}

// calcMessage efficiently concatenates timestamp:data without redundant allocations
func calcMessage(timestamp int64, data []byte) []byte {
	return append(append(strconv.AppendInt(nil, timestamp, 10), ':'), data...)
}

// calcAppID computes SHA-1 hash of the auth token
func calcAppID(authToken []byte) string {
	hash := sha1.Sum(authToken) //nolint:gosec // SHA1 is required by the protocol

	return hex.EncodeToString(hash[:])
}
