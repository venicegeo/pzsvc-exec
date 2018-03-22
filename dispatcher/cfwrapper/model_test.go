package cfwrapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFactoryConfig_CFClientConfig(t *testing.T) {
	// Tested code
	config := FactoryConfig{
		APIAddress:  "https://test.address.localdomain",
		Username:    "testuser",
		Password:    "testpass",
		HTTPClient:  nil,
		Token:       "test-token-abc",
		TokenSource: nil,
	}.CFClientConfig()

	// Asserts
	assert.Equal(t, "https://test.address.localdomain", config.ApiAddress)
	assert.Equal(t, "testuser", config.Username)
	assert.Equal(t, "testpass", config.Password)
	assert.Nil(t, config.HttpClient)
	assert.Equal(t, "test-token-abc", config.Token)
	assert.Nil(t, config.TokenSource)
	assert.Equal(t, "venicegeo/pzsvc-exec/dispatcher/cfwrapper 2.0", config.UserAgent)

}
