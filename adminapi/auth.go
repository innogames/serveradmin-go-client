package adminapi

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // SHA1 is required by the protocol
	"encoding/hex"
	"strconv"
)

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
