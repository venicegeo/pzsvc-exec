package cfwrapper

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	cfclient "github.com/venicegeo/go-cfclient"
)

func TestIsMemoryLimitError(t *testing.T) {
	// Tested code
	var (
		any                                          = IsMemoryLimitError(errors.New("test error"))
		cfUnauthenticatedError                       = IsMemoryLimitError(cfclient.CloudFoundryError{Code: 10002})
		custom                                       = IsMemoryLimitError(&CustomMemoryLimitError{"stuff"})
		cfAppMemoryQuotaExceededError                = IsMemoryLimitError(cfclient.CloudFoundryError{Code: 100005})
		cfQuotaInstanceLimitExceededError            = IsMemoryLimitError(cfclient.CloudFoundryError{Code: 100008})
		cfQuotaInstanceMemoryLimitExceededError      = IsMemoryLimitError(cfclient.CloudFoundryError{Code: 100007})
		cfSpaceQuotaInstanceMemoryLimitExceededError = IsMemoryLimitError(cfclient.CloudFoundryError{Code: 310004})
	)

	// Asserts
	assert.False(t, any)
	assert.False(t, cfUnauthenticatedError)
	assert.True(t, custom)
	assert.True(t, cfAppMemoryQuotaExceededError)
	assert.True(t, cfQuotaInstanceLimitExceededError)
	assert.True(t, cfQuotaInstanceMemoryLimitExceededError)
	assert.True(t, cfSpaceQuotaInstanceMemoryLimitExceededError)
}

func TestCustomMemoryLimitError_Error(t *testing.T) {
	// Tested code
	err := &CustomMemoryLimitError{"test message"}
	assert.Equal(t, "test message", err.Error())
}
