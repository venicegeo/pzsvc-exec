package workerexec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/worker/config"
)

type mockExecResult struct {
	s                    pzsvc.Session
	pzAddr, svcID, jobID string
	status               pzsvc.PiazzaStatus
	resultData           []byte
}

func TestDefaultPiazzaOutputter_Success(t *testing.T) {
	// Setup
	mockExecResults := []mockExecResult{}
	mockSendExecResultData := func(s pzsvc.Session, pzAddr, svcID, jobID string, status pzsvc.PiazzaStatus, resultData []byte) *pzsvc.PzCustomError {
		mockExecResults = append(mockExecResults, mockExecResult{s, pzAddr, svcID, jobID, status, resultData})
		return nil
	}
	workerConfig := config.WorkerConfig{MuteLogs: true, Session: &pzsvc.Session{}}
	outData := workerOutputData{Errors: []string{}}

	// Tested code
	outputter := newDefaultPiazzaOutputter()
	outputter.sendExecResultData = mockSendExecResultData
	outputter.OutputToPiazza(workerConfig, outData)

	// Asserts
	assert.Len(t, mockExecResults, 1)
	assert.Equal(t, pzsvc.PiazzaStatusSuccess, mockExecResults[0].status)
}

func TestDefaultPiazzaOutputter_JobError(t *testing.T) {
	// Setup
	mockExecResults := []mockExecResult{}
	mockSendExecResultData := func(s pzsvc.Session, pzAddr, svcID, jobID string, status pzsvc.PiazzaStatus, resultData []byte) *pzsvc.PzCustomError {
		mockExecResults = append(mockExecResults, mockExecResult{s, pzAddr, svcID, jobID, status, resultData})
		return nil
	}
	workerConfig := config.WorkerConfig{MuteLogs: true, Session: &pzsvc.Session{}}
	outData := workerOutputData{Errors: []string{"test error"}}

	// Tested code
	outputter := newDefaultPiazzaOutputter()
	outputter.sendExecResultData = mockSendExecResultData
	outputter.OutputToPiazza(workerConfig, outData)

	// Asserts
	assert.Len(t, mockExecResults, 1)
	assert.Equal(t, pzsvc.PiazzaStatusError, mockExecResults[0].status)
}
