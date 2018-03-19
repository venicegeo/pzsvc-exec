package workerlog

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/worker/config"
)

var mockStdout = bytes.NewBuffer([]byte{})

func TestMain(m *testing.M) {
	oldLogFunc := pzsvc.LogFunc
	pzsvc.LogFunc = func(msg string) {
		fmt.Fprintln(mockStdout, msg)
	}

	retCode := m.Run()

	pzsvc.LogFunc = oldLogFunc
	os.Exit(retCode)
}

func readAndClearMockStdout() string {
	stdoutData := mockStdout.String()
	mockStdout.Reset()
	return stdoutData
}

var sharedMockConfig = config.WorkerConfig{JobID: "job-id-123", Session: &pzsvc.Session{}}

func TestInfo(t *testing.T) {
	// Tested code
	Info(sharedMockConfig, "test message")

	// Asserts
	stdoutData := readAndClearMockStdout()
	assert.Contains(t, stdoutData, "{Worker, jobID=job-id-123}")
	assert.Contains(t, stdoutData, "test message")
	// TODO: no distinguishing feature of an "info" message?
}

func TestWarn(t *testing.T) {
	// Tested code
	Warn(sharedMockConfig, "test message")

	// Asserts
	stdoutData := readAndClearMockStdout()
	assert.Contains(t, stdoutData, "{Worker, jobID=job-id-123}")
	assert.Contains(t, stdoutData, "test message")
	// TODO: no distinguishing feature of a "warn" message?
}

func TestAlert(t *testing.T) {
	// Tested code
	Alert(sharedMockConfig, "test message")

	// Asserts
	stdoutData := readAndClearMockStdout()
	assert.Contains(t, stdoutData, "{Worker, jobID=job-id-123}")
	assert.Contains(t, stdoutData, "test message")
	assert.Contains(t, stdoutData, "ALERT")
}

func TestSimpleErr(t *testing.T) {
	// Tested code
	SimpleErr(sharedMockConfig, "test message", errors.New("test error"))

	// Asserts
	stdoutData := readAndClearMockStdout()
	assert.Contains(t, stdoutData, "{Worker, jobID=job-id-123}")
	assert.Contains(t, stdoutData, "test message")
	assert.Contains(t, stdoutData, "test error")
}

func TestMuteLogs(t *testing.T) {
	// Setup
	silentConfig := sharedMockConfig
	silentConfig.MuteLogs = true

	// Tested code
	Info(silentConfig, "test message")
	Warn(silentConfig, "test message")
	Alert(silentConfig, "test message")
	SimpleErr(silentConfig, "test message", errors.New("test error"))

	// Asserts
	stdoutData := readAndClearMockStdout()
	assert.Len(t, stdoutData, 0)
}
