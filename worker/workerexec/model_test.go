package workerexec

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkerOutputData_AddErrors(t *testing.T) {
	// Tested code
	outputData := workerOutputData{Errors: []string{"error1"}}
	outputData.AddErrors(errors.New("error2"), errors.New("error3"))

	// Asserts
	assert.Equal(t, []string{"error1", "error2", "error3"}, outputData.Errors)
}
