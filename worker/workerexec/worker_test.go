package workerexec

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/worker/config"
	"github.com/venicegeo/pzsvc-exec/worker/ingest"
)

// mock objects setup, isolate functionality of the worker

type outputFilesToPiazzaCall struct {
	algFullCommand, algVersion string
}

type sendExecResultDataCall struct {
	pzAddr, svcID, jobID string
	status               pzsvc.PiazzaStatus
	resultData           []byte
}

type workerMock struct {
	workerConfig             *config.WorkerConfig
	worker                   *Worker
	fetchInputsCalls         [][]config.InputSource
	outputFilesToPiazzaCalls []outputFilesToPiazzaCall
	sendExecResultDataCalls  []sendExecResultDataCall
	commandRunnerCalls       [][]string
}

func execMockSetup() *workerMock {
	mock := &workerMock{
		workerConfig:             &config.WorkerConfig{MuteLogs: true, Session: &pzsvc.Session{}},
		worker:                   NewWorker(),
		fetchInputsCalls:         [][]config.InputSource{},
		outputFilesToPiazzaCalls: []outputFilesToPiazzaCall{},
		sendExecResultDataCalls:  []sendExecResultDataCall{},
		commandRunnerCalls:       [][]string{},
	}

	mock.worker.fetchInputsFunc = func(cfg config.WorkerConfig, inputs []config.InputSource) error {
		mock.fetchInputsCalls = append(mock.fetchInputsCalls, inputs)
		return nil
	}
	mock.worker.outputFilesToPiazzaFunc = func(cfg config.WorkerConfig, algFullCommand string, algVersion string) ingest.MultiIngestOutput {
		mock.outputFilesToPiazzaCalls = append(mock.outputFilesToPiazzaCalls, outputFilesToPiazzaCall{algFullCommand, algVersion})
		return ingest.MultiIngestOutput{}
	}
	mock.worker.piazzaOutputter = newPiazzaOutputter()
	mock.worker.piazzaOutputter.sendExecResultData = func(s pzsvc.Session, pzAddr, svcID, jobID string, status pzsvc.PiazzaStatus, resultData []byte) *pzsvc.PzCustomError {
		mock.sendExecResultDataCalls = append(mock.sendExecResultDataCalls, sendExecResultDataCall{pzAddr, svcID, jobID, status, resultData})
		return nil
	}
	mock.worker.commandRunner = newCommandRunner()
	mock.worker.commandRunner.exec = func(cmdName string, args ...string) ([]byte, error) {
		mock.commandRunnerCalls = append(mock.commandRunnerCalls, append([]string{cmdName}, args...))
		return nil, nil
	}

	return mock
}

// actual tests

func TestExec_Success(t *testing.T) {
	// Setup
	execMock := execMockSetup()
	execMock.workerConfig.Inputs = []config.InputSource{
		config.InputSource{FileName: "input1.txt", URL: "http://example1.localdomain/input1_source.txt"},
		config.InputSource{FileName: "input2.tif", URL: "http://example2.localdomain/input2_source.tif"},
	}
	execMock.workerConfig.Outputs = []string{"output1.txt", "output2.geojson"}
	execMock.workerConfig.PzSEConfig.CliCmd = "test cli command"
	execMock.workerConfig.CLICommandExtra = "--extra"
	execMock.workerConfig.PzSEConfig.VersionCmd = "version cli command"
	oldExec := execMock.worker.commandRunner.exec
	execMock.worker.commandRunner.exec = func(cmdName string, args ...string) ([]byte, error) {
		oldExec(cmdName, args...)
		return []byte("1.2.3test"), nil
	}

	// Tested code
	err := execMock.worker.Exec(*execMock.workerConfig)

	// Asserts
	assert.Nil(t, err) // check no unrecoverable error

	// check inputs were fetched
	assert.Len(t, execMock.fetchInputsCalls, 1)
	assert.Equal(t, execMock.workerConfig.Inputs, execMock.fetchInputsCalls[0])

	// check output files were sent to piazza
	assert.Len(t, execMock.outputFilesToPiazzaCalls, 1)
	assert.Equal(t, outputFilesToPiazzaCall{"test cli command --extra", "1.2.3test"}, execMock.outputFilesToPiazzaCalls[0])

	// check successful exec result was sent to piazza
	assert.Len(t, execMock.sendExecResultDataCalls, 1)
	assert.Equal(t, pzsvc.PiazzaStatusSuccess, execMock.sendExecResultDataCalls[0].status)
}

func TestExec_ErrorInputs(t *testing.T) {
	// Setup
	execMock := execMockSetup()
	execMock.workerConfig.Inputs = []config.InputSource{
		config.InputSource{FileName: "input1.txt", URL: "http://example1.localdomain/input1_source.txt"},
		config.InputSource{FileName: "input2.tif", URL: "http://example2.localdomain/input2_source.tif"},
	}
	execMock.worker.fetchInputsFunc = func(cfg config.WorkerConfig, inputs []config.InputSource) error {
		return errors.New("test input error")
	}

	// Tested code
	err := execMock.worker.Exec(*execMock.workerConfig)

	// Asserts
	assert.Nil(t, err) // check no unrecoverable error

	// check error exec result was sent to piazza, containing our error message
	assert.Len(t, execMock.sendExecResultDataCalls, 1)
	assert.Equal(t, pzsvc.PiazzaStatusError, execMock.sendExecResultDataCalls[0].status)
	assert.Contains(t, string(execMock.sendExecResultDataCalls[0].resultData), "test input error")
}

func TestExec_ErrorVersionCmd(t *testing.T) {
	// Setup
	execMock := execMockSetup()
	execMock.workerConfig.PzSEConfig.VersionCmd = "version-cmd"
	execMock.worker.commandRunner.exec = func(cmdName string, args ...string) ([]byte, error) {
		for _, arg := range args {
			if strings.Contains(arg, "version-cmd") {
				return []byte{}, errors.New("test version cmd error")
			}
		}
		return []byte("ok"), nil
	}

	// Tested code
	err := execMock.worker.Exec(*execMock.workerConfig)

	// Asserts
	assert.Nil(t, err) // check no unrecoverable error

	// check error exec result was sent to piazza, containing our error message
	assert.Len(t, execMock.sendExecResultDataCalls, 1)
	assert.Equal(t, pzsvc.PiazzaStatusError, execMock.sendExecResultDataCalls[0].status)
	assert.Contains(t, string(execMock.sendExecResultDataCalls[0].resultData), "test version cmd error")
}

func TestExec_ErrorAlgoCmd(t *testing.T) {
	// Setup
	execMock := execMockSetup()
	execMock.workerConfig.PzSEConfig.CliCmd = "algo-cmd"
	execMock.worker.commandRunner.exec = func(cmdName string, args ...string) ([]byte, error) {
		for _, arg := range args {
			if strings.Contains(arg, "algo-cmd") {
				return []byte{}, errors.New("test algo cmd error")
			}
		}
		return []byte("ok"), nil
	}

	// Tested code
	err := execMock.worker.Exec(*execMock.workerConfig)

	// Asserts
	assert.Nil(t, err) // check no unrecoverable error

	// check error exec result was sent to piazza, containing our error message
	assert.Len(t, execMock.sendExecResultDataCalls, 1)
	assert.Equal(t, pzsvc.PiazzaStatusError, execMock.sendExecResultDataCalls[0].status)
	assert.Contains(t, string(execMock.sendExecResultDataCalls[0].resultData), "test algo cmd error")
}

func TestExec_ErrorIngest(t *testing.T) {
	// Setup
	execMock := execMockSetup()
	execMock.worker.outputFilesToPiazzaFunc = func(cfg config.WorkerConfig, algFullCommand string, algVersion string) ingest.MultiIngestOutput {
		return ingest.MultiIngestOutput{
			CombinedError: errors.New("test combined error"),
			Errors:        []error{errors.New("test error 1"), errors.New("test error 2")},
		}
	}

	// Tested code
	err := execMock.worker.Exec(*execMock.workerConfig)

	// Asserts
	assert.Nil(t, err) // check no unrecoverable error

	// check error exec result was sent to piazza, containing our error message
	assert.Len(t, execMock.sendExecResultDataCalls, 1)
	assert.Equal(t, pzsvc.PiazzaStatusError, execMock.sendExecResultDataCalls[0].status)
	assert.Contains(t, string(execMock.sendExecResultDataCalls[0].resultData), "test error 1")
	assert.Contains(t, string(execMock.sendExecResultDataCalls[0].resultData), "test error 2")
}

func TestExec_ErrorSendExecResult(t *testing.T) {
	// Setup
	execMock := execMockSetup()
	mockError := &pzsvc.PzCustomError{}
	execMock.worker.piazzaOutputter.sendExecResultData = func(s pzsvc.Session, pzAddr, svcID, jobID string, status pzsvc.PiazzaStatus, resultData []byte) *pzsvc.PzCustomError {
		return mockError
	}

	// Tested code
	err := execMock.worker.Exec(*execMock.workerConfig)

	// Asserts
	assert.NotNil(t, err) // check there should be an unrecoverable error
	assert.Contains(t, err.Error(), "failed to send result data")
}
