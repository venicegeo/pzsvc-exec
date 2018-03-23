package poll

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVCAPApplicationID_BadEnv(t *testing.T) {
	// Setup
	mockVCAP := setMockEnv("VCAP_APPLICATION", "bad json value")
	defer mockVCAP.Restore()

	// Tested code
	id, err := getVCAPApplicationID()

	// Asserts
	assert.Empty(t, id)
	assert.NotNil(t, err)
}

func TestGetVCAPApplicationID_NonString(t *testing.T) {
	// Setup
	mockVCAP := setMockEnv("VCAP_APPLICATION", `{"application_id": {}}`)
	defer mockVCAP.Restore()

	// Tested code
	id, err := getVCAPApplicationID()

	// Asserts
	assert.Empty(t, id)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "string type assertion failed")
}

func TestGetVCAPApplicationID_Success(t *testing.T) {
	// Setup
	mockVCAP := setMockEnv("VCAP_APPLICATION", `{"application_id": "test-id-123"}`)
	defer mockVCAP.Restore()

	// Tested code
	id, err := getVCAPApplicationID()

	// Asserts
	assert.Nil(t, err)
	assert.Equal(t, "test-id-123", id)
}
