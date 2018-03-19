package workerexec

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/worker/config"
)

func TestDefaultCommandRunner_Success(t *testing.T) {
	// Setup
	execCalls := [][]string{}
	exec := func(cmdName string, args ...string) ([]byte, error) {
		call := append([]string{cmdName}, args...)
		execCalls = append(execCalls, call)
		return []byte("ok"), nil
	}
	workerConfig := config.WorkerConfig{MuteLogs: true, Session: &pzsvc.Session{}}

	// Tested code
	runner := newDefaultCommandRunner()
	runner.exec = exec
	output := runner.Run(workerConfig, "test command")

	// Asserts
	assert.Equal(t, []byte("ok"), output.Stdout)
	assert.Empty(t, output.Stderr)
	assert.Nil(t, output.Error)
	assert.Len(t, execCalls, 1)
	assert.Equal(t, []string{"sh", "-c", "test command"}, execCalls[0])
}

func TestDefaultCommandRunner_ExitError(t *testing.T) {
	// Setup
	execCalls := [][]string{}
	exec := func(cmdName string, args ...string) ([]byte, error) {
		call := append([]string{cmdName}, args...)
		execCalls = append(execCalls, call)
		return []byte("stdout test error"), &exec.ExitError{Stderr: []byte("stderr test error")}
	}
	workerConfig := config.WorkerConfig{MuteLogs: true, Session: &pzsvc.Session{}}

	// Tested code
	runner := newDefaultCommandRunner()
	runner.exec = exec
	output := runner.Run(workerConfig, "test command")

	// Asserts
	assert.Equal(t, []byte("stdout test error"), output.Stdout)
	assert.Equal(t, []byte("stderr test error"), output.Stderr)
	assert.NotNil(t, output.Error)
	assert.Len(t, execCalls, 1)
	assert.Equal(t, []string{"sh", "-c", "test command"}, execCalls[0])
}

func TestDefaultCommandRunner_UnknownError(t *testing.T) {
	// Setup
	execCalls := [][]string{}
	exec := func(cmdName string, args ...string) ([]byte, error) {
		call := append([]string{cmdName}, args...)
		execCalls = append(execCalls, call)
		return []byte("stdout test error"), errors.New("unknown error")
	}
	workerConfig := config.WorkerConfig{MuteLogs: true, Session: &pzsvc.Session{}}

	// Tested code
	runner := newDefaultCommandRunner()
	runner.exec = exec
	output := runner.Run(workerConfig, "test command")

	// Asserts
	assert.Equal(t, []byte("stdout test error"), output.Stdout)
	assert.Empty(t, output.Stderr)
	assert.NotNil(t, output.Error)
	assert.Len(t, execCalls, 1)
	assert.Equal(t, []string{"sh", "-c", "test command"}, execCalls[0])
}

func TestDefaultCommandRunner_NativeCommand(t *testing.T) {
	// Availability probe
	probeOutput, err := exec.Command("sh", "-c", "echo hello").Output()
	if err != nil || string(probeOutput) != "hello\n" {
		t.Skip("`sh -c` not available on this platform")
	}
	workerConfig := config.WorkerConfig{MuteLogs: true, Session: &pzsvc.Session{}}

	// Tested code
	runner := newDefaultCommandRunner()
	output := runner.Run(workerConfig, "echo hello")

	// Asserts
	assert.Equal(t, []byte("hello\n"), output.Stdout)
	assert.Empty(t, output.Stderr)
	assert.Nil(t, output.Error)
}
