package poll

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

func TestNewLoop_BadVCAP(t *testing.T) {
	// Setup
	mockVCAP := setMockEnv("VCAP_APPLICATION", `{"application_id": {}}`)
	defer mockVCAP.Restore()

	// Tested code
	loop, err := NewLoop(&pzsvc.Session{}, pzsvc.Config{}, "test-svcid-123", "/path/to/config", nil)

	// Asserts
	assert.Nil(t, loop)
	assert.NotNil(t, err)
}

func TestNewLoop_NoTaskLimit(t *testing.T) {
	// Setup
	mockVCAP := setMockEnv("VCAP_APPLICATION", `{"application_id": "test-app-123"}`)
	defer mockVCAP.Restore()
	mockTaskLimit := setMockEnv("TASK_LIMIT", "")
	defer mockTaskLimit.Restore()

	// Tested code
	loop, err := NewLoop(&pzsvc.Session{}, pzsvc.Config{}, "test-svcid-123", "/path/to/config", nil)

	// Asserts
	assert.Nil(t, err)
	assert.NotNil(t, loop)
	assert.Equal(t, 3, loop.taskLimit)
	assert.Equal(t, "test-app-123", loop.vcapID)
	assert.Equal(t, "test-svcid-123", loop.SvcID)
	assert.Equal(t, "/path/to/config", loop.ConfigPath)
}

func TestNewLoop_SetTaskLimit(t *testing.T) {
	// Setup
	mockVCAP := setMockEnv("VCAP_APPLICATION", `{"application_id": "test-app-123"}`)
	defer mockVCAP.Restore()
	mockTaskLimit := setMockEnv("TASK_LIMIT", "100")
	defer mockTaskLimit.Restore()

	// Tested code
	loop, err := NewLoop(&pzsvc.Session{}, pzsvc.Config{}, "test-svcid-123", "/path/to/config", nil)

	// Asserts
	assert.Nil(t, err)
	assert.NotNil(t, loop)
	assert.Equal(t, 100, loop.taskLimit)
	assert.Equal(t, "test-app-123", loop.vcapID)
	assert.Equal(t, "test-svcid-123", loop.SvcID)
	assert.Equal(t, "/path/to/config", loop.ConfigPath)
}

func TestLoop_StartStop(t *testing.T) {
	// Setup
	iterations := 0
	errorsEmitted := 0
	errorsReceived := 0

	mockVCAP := setMockEnv("VCAP_APPLICATION", `{"application_id": "test-app-123"}`)
	defer mockVCAP.Restore()
	loop, _ := NewLoop(&pzsvc.Session{}, pzsvc.Config{}, "test-svcid-123", "/path/to/config", nil)
	loop.intervalTick = 5 * time.Millisecond
	loop.runIterationFunc = func(l Loop) error {
		iterations++
		if iterations%2 == 0 {
			errorsEmitted++
			return errors.New("Test error")
		}
		return nil
	}

	// Tested code
	errChan := loop.Start()
	go func() {
		<-time.After(50 * time.Millisecond)
		loop.Stop()
	}()
	for range errChan {
		errorsReceived++
	}

	// Asserts
	assert.Condition(t, func() bool {
		return 10-iterations <= 1
	})
	assert.Equal(t, errorsEmitted, errorsReceived)
}

func TestLoop_CalculateDiskAndMemoryLimits_NonS3(t *testing.T) {
	// With at least one non-S3 source, the result should be the default disk/memory sizes

	// Setup
	original := setMockPzsvcGetS3FileSizeInMegabytes(func(string) (int, *pzsvc.PzCustomError) { return 128, nil })
	defer original.Restore()
	jobInput := pzsvc.InpStruct{InExtFiles: []string{"https://s3.amazonaws.localdomain/file1.txt", "https://not-aws.somehost.com"}}
	loop := Loop{PzSession: &pzsvc.Session{}}

	// Tested code
	diskMB, memoryMB := loop.calculateDiskAndMemoryLimits(&jobInput)

	// Asserts
	assert.Equal(t, defaultTaskDiskMB, diskMB)
	assert.Equal(t, defaultTaskMemoryMB, memoryMB)
}

func TestLoop_CalculateDiskAndMemoryLimits_S3Error(t *testing.T) {
	// With at least one S3 source returning an error, the result should be the default disk/memory sizes

	// Setup
	original := setMockPzsvcGetS3FileSizeInMegabytes(func(url string) (int, *pzsvc.PzCustomError) {
		if strings.Contains(url, "file2.tif") {
			return 0, &pzsvc.PzCustomError{}
		}
		return 128, nil
	})
	defer original.Restore()
	jobInput := pzsvc.InpStruct{InExtFiles: []string{"https://s3.amazonaws.localdomain/file1.txt", "https://s3.amazonaws.localdomain/file2.tif", "https://s3.amazonaws.localdomain/file3.jp2"}}
	loop := Loop{PzSession: &pzsvc.Session{}}

	// Tested code
	diskMB, memoryMB := loop.calculateDiskAndMemoryLimits(&jobInput)

	// Asserts
	assert.Equal(t, defaultTaskDiskMB, diskMB)
	assert.Equal(t, defaultTaskMemoryMB, memoryMB)
}

func TestLoop_CalculateDiskAndMemoryLimits_Success(t *testing.T) {
	// Setup
	original := setMockPzsvcGetS3FileSizeInMegabytes(func(url string) (int, *pzsvc.PzCustomError) { return 128, nil })
	defer original.Restore()
	jobInput := pzsvc.InpStruct{InExtFiles: []string{"https://s3.amazonaws.localdomain/file1.txt", "https://s3.amazonaws.localdomain/file2.tif"}}
	loop := Loop{PzSession: &pzsvc.Session{}}

	// Tested code
	diskMB, memoryMB := loop.calculateDiskAndMemoryLimits(&jobInput)

	// Asserts
	assert.Equal(t, 2048+(128+128)*2, diskMB)
	assert.Equal(t, defaultTaskMemoryMB+(128+128)*5, memoryMB)
}

func TestLoop_BuildWorkerCommand_BadInput(t *testing.T) {
	// Setup
	loop := Loop{PzSession: &pzsvc.Session{}, ConfigPath: "/path/to/config", SvcID: "test-svcid-123"}
	jobInput := pzsvc.InpStruct{
		Command:    "test-command-extra",
		UserID:     "test-user-123",
		InExtNames: []string{"inputFile1.txt", "inputFile2.tif", "inputfile3.jp2"},
		InExtFiles: []string{"https://s3.amazonaws.localdomain/file1.txt", "https://s3.amazonaws.localdomain/file2.tif"},
		OutGeoJs:   []string{"output1.geojson", "output2.geojson"},
	}

	// Tested code
	command, err := loop.buildWorkerCommand(&jobInput, "job-id-123")

	// Asserts
	assert.Empty(t, command)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "did not match")
}

func TestLoop_BuildWorkerCommand_Success(t *testing.T) {
	// Setup
	loop := Loop{PzSession: &pzsvc.Session{}, ConfigPath: "/path/to/config", SvcID: "test-svcid-123"}
	jobInput := pzsvc.InpStruct{
		Command:    "test-command-extra",
		UserID:     "test-user-123",
		InExtNames: []string{"inputFile1.txt", "inputFile2.tif"},
		InExtFiles: []string{"https://s3.amazonaws.localdomain/file1.txt", "https://s3.amazonaws.localdomain/file2.tif"},
		OutGeoJs:   []string{"output1.geojson", "output2.geojson"},
	}

	// Tested code
	command, err := loop.buildWorkerCommand(&jobInput, "job-id-123")

	// Asserts
	assert.Nil(t, err)
	assert.Equal(t, `worker --cliExtra 'test-command-extra' --userID 'test-user-123'`+
		` --config '/path/to/config' --serviceID 'test-svcid-123' --jobID 'job-id-123'`+
		` -i inputFile1.txt:https://s3.amazonaws.localdomain/file1.txt -i inputFile2.tif:https://s3.amazonaws.localdomain/file2.tif`+
		` -o output1.geojson -o output2.geojson`, command)
}

func TestLoop_ParseJobInput_BadInput(t *testing.T) {
	// Setup
	loop := Loop{PzSession: &pzsvc.Session{}}
	jobInputStr := "this is bad json"

	// Tested code
	jobInput, err := loop.parseJobInput(jobInputStr)

	// Asserts
	assert.Nil(t, jobInput)
	assert.NotNil(t, err)
}

func TestLoop_ParseJobInput_Success(t *testing.T) {
	// Setup
	loop := Loop{PzSession: &pzsvc.Session{}}
	jobInputStr := `{
		"cmd": "test-command",
		"userID": "test-user-id",
		"inPzFiles": [],
		"inExtFiles": ["https://s3.amazonaws.localdomain/file1.txt", "https://s3.amazonaws.localdomain/file2.tif"],
		"inPzNames": [],
		"inExtNames": ["inputFile1.txt", "inputFile2.tif"],
		"OutTiffs": [],
		"OutTxts": [],
		"OutGeoJson": ["output1.geojson", "output2.geojson"],
		"inExtAuthKey": "test-ext-auth-key",
		"pzAuthKey": "test-pz-auth-key",
		"pzAddr": "test-pz-addr"
	}`

	// Tested code
	jobInput, err := loop.parseJobInput(jobInputStr)

	// Asserts
	assert.Nil(t, err)
	assert.Equal(t, pzsvc.InpStruct{
		Command:    "test-command",
		UserID:     "test-user-id",
		InPzFiles:  []string{},
		InExtFiles: []string{"https://s3.amazonaws.localdomain/file1.txt", "https://s3.amazonaws.localdomain/file2.tif"},
		InPzNames:  []string{},
		InExtNames: []string{"inputFile1.txt", "inputFile2.tif"},
		OutTiffs:   []string{},
		OutTxts:    []string{},
		OutGeoJs:   []string{"output1.geojson", "output2.geojson"},
		ExtAuth:    "*****",
		PzAuth:     "*****",
		PzAddr:     "test-pz-addr",
	}, *jobInput)
}

func TestLoop_GetPzTaskItem_Failure(t *testing.T) {
	// Setup
	setMockPzsvcRequestKnownJSON(func(method, bodyStr, address, authKey string, outObj interface{}) ([]byte, *pzsvc.PzCustomError) {
		return nil, &pzsvc.PzCustomError{}
	})
	loop := Loop{
		PzSession: &pzsvc.Session{
			PzAddr: "https://piazza.localdomain",
		},
		SvcID: "test-svc-id",
	}

	// Test code
	taskItem, err := loop.getPzTaskItem()

	// Asserts
	assert.NotNil(t, err)
	assert.Nil(t, taskItem)
}

func TestLoop_GetPzTaskItem_Success(t *testing.T) {
	// Setup
	requestedAddress := ""
	setMockPzsvcRequestKnownJSON(func(method, bodyStr, address, authKey string, outObj interface{}) ([]byte, *pzsvc.PzCustomError) {
		requestedAddress = address
		body := []byte(`{"data": {"serviceData": {"jobID": "test-job-id", "data": {"dataInputs": {"body": {"content": "test-job-content"}}}}}}`)
		json.Unmarshal(body, outObj)
		return body, nil
	})
	loop := Loop{
		PzSession: &pzsvc.Session{
			PzAddr: "https://piazza.localdomain",
		},
		SvcID: "test-svc-id",
	}

	// Test code
	taskItem, err := loop.getPzTaskItem()

	// Asserts
	assert.Equal(t, "https://piazza.localdomain/service/test-svc-id/task", requestedAddress)
	assert.Nil(t, err)
	assert.Equal(t, "test-job-id", taskItem.Data.SvcData.JobID)
	assert.Equal(t, "test-job-content", taskItem.Data.SvcData.Data.DataInputs.Body.Content)
}
