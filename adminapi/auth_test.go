package adminapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// just some simple example tests, e2e tests might make much more sense here for full coverage
func TestAppId(t *testing.T) {
	testCases := []struct {
		input    []byte
		expected string
	}{
		{
			input:    []byte("1234567898"),
			expected: "d396f232a5ca1f7a0ad8f1b59975515123780553",
		},
	}

	for _, testCase := range testCases {
		actual := calcAppID(testCase.input)
		assert.Equal(t, testCase.expected, actual)
	}
}

func TestSecurityToken(t *testing.T) {
	testCases := []struct {
		apiKey   []byte
		message  string
		expected string
	}{
		{
			apiKey:   []byte("1234567898"),
			message:  "",
			expected: "4199b91c6f92f3e1d29f88a5f67973ad8aaec5b5",
		},
		{
			apiKey:   []byte("1234567898"),
			message:  "foobar",
			expected: "e17ba31a1a664617653869db8289f92a49213e7b",
		},
	}

	now := int64(123456789)
	for _, testCase := range testCases {
		actual := calcSecurityToken(testCase.apiKey, now, []byte(testCase.message))
		assert.Equal(t, testCase.expected, actual)
	}
}

func BenchmarkCalcSecurityToken(b *testing.B) {
	now := int64(123456789)
	message := []byte("foobar")
	authToken := []byte("1234567898")
	for b.Loop() {
		calcSecurityToken(authToken, now, message)
	}
}
