package cfwrapper

import cfclient "github.com/venicegeo/go-cfclient"

// IsMemoryLimitError wraps CF detection of multiple memory error codes
func IsMemoryLimitError(err error) bool {
	if _, ok := err.(*CustomMemoryLimitError); ok {
		return true
	}
	return cfclient.IsAppMemoryQuotaExceededError(err) ||
		cfclient.IsQuotaInstanceLimitExceededError(err) ||
		cfclient.IsQuotaInstanceMemoryLimitExceededError(err) ||
		cfclient.IsSpaceQuotaInstanceMemoryLimitExceededError(err)
}

// CustomMemoryLimitError is a struct containing a memory limit error that is not a standard cfclient error
// This is mostly intended for testing purposes
type CustomMemoryLimitError struct {
	Message string
}

func (err *CustomMemoryLimitError) Error() string {
	return err.Message
}
