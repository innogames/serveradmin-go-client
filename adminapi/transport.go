package adminapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // SHA1 is required by the protocol
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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
		return nil, fmt.Errorf("sending request to %s: %w", endpoint, err)
	}

	// special error handling
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()

		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Status:     http.StatusText(resp.StatusCode),
		}

		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, apiErr
		}

		var nestedErrorResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if jsonErr := json.Unmarshal(bodyBytes, &nestedErrorResp); jsonErr == nil && nestedErrorResp.Error.Message != "" {
			apiErr.Message = nestedErrorResp.Error.Message
		}

		return nil, apiErr
	}

	return resp, nil
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
