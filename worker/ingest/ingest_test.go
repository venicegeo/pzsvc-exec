package ingest

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/worker/config"
)

// test setup/teardown

type mockAsyncIngestorCall struct {
	s                                         pzsvc.Session
	filePath, fileType, serviceID, algVersion string
	attMap                                    map[string]string
}

type mockAsyncIngestor struct {
	Calls         []mockAsyncIngestorCall
	ReturnOutputs chan singleIngestOutput
}

func (ingestor *mockAsyncIngestor) ingestFileAsync(s pzsvc.Session, filePath, fileType, serviceID, algVersion string, attMap map[string]string) <-chan singleIngestOutput {
	ingestor.Calls = append(ingestor.Calls, mockAsyncIngestorCall{s, filePath, fileType, serviceID, algVersion, attMap})
	returnChan := make(chan singleIngestOutput)
	go func() {
		returnChan <- <-ingestor.ReturnOutputs
		close(returnChan)
	}()
	return returnChan
}

func (ingestor *mockAsyncIngestor) Reset(returnOutputs []singleIngestOutput) {
	ingestor.Calls = []mockAsyncIngestorCall{}
	ingestor.ReturnOutputs = make(chan singleIngestOutput, len(returnOutputs))
	for _, output := range returnOutputs {
		ingestor.ReturnOutputs <- output
	}
}

var mockAsyncIngestorInstance mockAsyncIngestor

func setUpMockAsyncIngestor() {
	mockAsyncIngestorInstance = mockAsyncIngestor{[]mockAsyncIngestorCall{}, nil}
	asyncIngestorInstance = &mockAsyncIngestorInstance
}

func tearDownMockAsyncIngestor() {
	asyncIngestorInstance = &defaultAsyncIngestor{}
}

var testWorkerConfig = config.WorkerConfig{
	Session:         &pzsvc.Session{},
	PiazzaBaseURL:   "https://piazza.example.localdomain/",
	PiazzaAPIKey:    "test-piazza-key-123",
	PiazzaServiceID: "test-piazza-service-id",
	CLICommandExtra: "--extra",
	UserID:          "test-user-id-123",
	JobID:           "test-job-id-123",
	Inputs: []config.InputSource{
		config.InputSource{FileName: "input1.txt", URL: "http://example1.localdomain/input_url1.txt"},
		config.InputSource{FileName: "input2.tif", URL: "http://example2.localdomain/input_url2.tif"},
	},
	Outputs:    []string{},
	PzSEConfig: pzsvc.Config{},
	MuteLogs:   true,
}

var mockOutput1 *os.File
var mockOutput2 *os.File

func setUpMockOutputs() {
	var err error
	mockOutput1, err = ioutil.TempFile("", "ingest_mock_output_1")
	if err != nil {
		panic(err)
	}
	mockOutput2, err = ioutil.TempFile("", "ingest_mock_output_2")
	if err != nil {
		panic(err)
	}
}

func tearDownMockOutputs() {
	os.Remove(mockOutput1.Name())
	os.Remove(mockOutput2.Name())
}

// actual test functions

func TestAssembleIngestorCalls_Success(t *testing.T) {
	testWorkerConfig.Outputs = []string{mockOutput1.Name(), mockOutput2.Name()}

	ingestorCalls, asmErrors := assembleIngestorCalls(testWorkerConfig, "./run_algo --someArg 123 --anotherAlg value", "1.2.3test")

	assert.Empty(t, asmErrors)
	assert.Len(t, ingestorCalls, 2)

	assert.Equal(t, "1.2.3test", ingestorCalls[0].algVersion)
	assert.Equal(t, mockOutput1.Name(), ingestorCalls[0].filePath)
	assert.Equal(t, testWorkerConfig.PiazzaServiceID, ingestorCalls[0].serviceID)
	assert.Equal(t, "./run_algo --someArg 123 --anotherAlg value", ingestorCalls[0].attMap["algoCmd"])
	assert.Equal(t, "1.2.3test", ingestorCalls[0].attMap["algoVersion"])

	assert.Equal(t, "1.2.3test", ingestorCalls[1].algVersion)
	assert.Equal(t, mockOutput2.Name(), ingestorCalls[1].filePath)
	assert.Equal(t, testWorkerConfig.PiazzaServiceID, ingestorCalls[1].serviceID)
	assert.Equal(t, "./run_algo --someArg 123 --anotherAlg value", ingestorCalls[1].attMap["algoCmd"])
	assert.Equal(t, "1.2.3test", ingestorCalls[1].attMap["algoVersion"])
}

func TestAssembleIngestorCalls_Failure(t *testing.T) {
	testWorkerConfig.Outputs = []string{"does_not_exist_1.txt", "does_not_exist_2.geojson"}

	ingestorCalls, asmErrors := assembleIngestorCalls(testWorkerConfig, "./run_algo --someArg 123 --anotherAlg value", "1.2.3test")

	assert.Empty(t, ingestorCalls)
	assert.Len(t, asmErrors, 2)
}

func TestAssembleIngestorCalls_Mixture(t *testing.T) {
	testWorkerConfig.Outputs = []string{"does_not_exist_1.txt", mockOutput1.Name(), "does_not_exist_2.geojson", mockOutput2.Name()}

	ingestorCalls, asmErrors := assembleIngestorCalls(testWorkerConfig, "./run_algo --someArg 123 --anotherAlg value", "1.2.3test")

	assert.Len(t, ingestorCalls, 2)
	assert.Len(t, asmErrors, 2)

	assert.Equal(t, "1.2.3test", ingestorCalls[0].algVersion)
	assert.Equal(t, mockOutput1.Name(), ingestorCalls[0].filePath)
	assert.Equal(t, testWorkerConfig.PiazzaServiceID, ingestorCalls[0].serviceID)
	assert.Equal(t, "./run_algo --someArg 123 --anotherAlg value", ingestorCalls[0].attMap["algoCmd"])
	assert.Equal(t, "1.2.3test", ingestorCalls[0].attMap["algoVersion"])

	assert.Equal(t, "1.2.3test", ingestorCalls[1].algVersion)
	assert.Equal(t, mockOutput2.Name(), ingestorCalls[1].filePath)
	assert.Equal(t, testWorkerConfig.PiazzaServiceID, ingestorCalls[1].serviceID)
	assert.Equal(t, "./run_algo --someArg 123 --anotherAlg value", ingestorCalls[1].attMap["algoCmd"])
}

func TestCallAsyncIngestor(t *testing.T) {
	mockOutputs := []singleIngestOutput{
		singleIngestOutput{"good-output-1.txt", "output-data-id-1", nil},
		singleIngestOutput{"bad-output-1.tif", "", errors.New("test error")},
		singleIngestOutput{"good-output-2.geojson", "output-data-id-2", nil},
	}
	mockAsyncIngestorInstance.Reset(mockOutputs)
	ingestorCalls := []asyncIngestorCall{
		asyncIngestorCall{pzsvc.Session{}, "good-output-1.txt", "text", testWorkerConfig.PiazzaServiceID, "1.2.3test", map[string]string{}},
		asyncIngestorCall{pzsvc.Session{}, "bad-output-1.tif", "raster", testWorkerConfig.PiazzaServiceID, "1.2.3test", map[string]string{}},
		asyncIngestorCall{pzsvc.Session{}, "good-output-2.geojson", "geojson", testWorkerConfig.PiazzaServiceID, "1.2.3test", map[string]string{}},
	}

	outputChans := callAsyncIngestor(ingestorCalls)

	assert.Len(t, outputChans, len(mockOutputs))
	for i, outputChan := range outputChans {
		output := <-outputChan
		foundOutput := false
		for _, mockOutput := range mockOutputs {
			if mockOutput.FilePath == output.FilePath {
				assert.Equal(t, mockOutput, output)
				foundOutput = true
				break
			}
		}

		if !foundOutput {
			assert.Fail(t, fmt.Sprintf("got an output with unexpected filename `%s`", output.FilePath))
		}

		if extra, ok := <-outputChans[i]; ok {
			assert.Fail(t, fmt.Sprintf("outputChans[%d] had more results in it; expected it to close", i), fmt.Sprint(extra))
		}
	}
}

func singleIngestOutputChanWithOneValue(output singleIngestOutput) <-chan singleIngestOutput {
	outputChan := make(chan singleIngestOutput)
	go func() {
		outputChan <- output
		close(outputChan)
	}()
	return outputChan
}

func TestHandleIngestResults(t *testing.T) {
	mockOutputChans := []<-chan singleIngestOutput{
		singleIngestOutputChanWithOneValue(singleIngestOutput{"good-output-1.txt", "good-data-id-1", nil}),
		singleIngestOutputChanWithOneValue(singleIngestOutput{"bad-output-1.tif", "", errors.New("test error")}),
		singleIngestOutputChanWithOneValue(singleIngestOutput{"good-output-2.geojson", "good-data-id-2", nil}),
	}
	mockPrependErrors := []error{errors.New("prepend error")}

	multiOutput := handleIngestResults(testWorkerConfig, mockOutputChans, mockPrependErrors)

	assert.Equal(t, "good-data-id-1", multiOutput.DataIDs["good-output-1.txt"])
	assert.Equal(t, "good-data-id-2", multiOutput.DataIDs["good-output-2.geojson"])
	assert.Len(t, multiOutput.Errors, 2)
	assert.Equal(t, "prepend error", multiOutput.Errors[0].Error())
	assert.Equal(t, "test error", multiOutput.Errors[1].Error())
	assert.Equal(t, "ingest errors: prepend error; test error", multiOutput.CombinedError.Error())
}

func TestOutputFilesToPiazza_Full(t *testing.T) {
	testWorkerConfig.Outputs = []string{mockOutput1.Name(), mockOutput2.Name(), "does_not_exist.txt"}
	mockOutputs := []singleIngestOutput{
		singleIngestOutput{mockOutput1.Name(), "output-data-id-1", nil},
		singleIngestOutput{mockOutput2.Name(), "output-data-id-2", nil},
	}
	mockAsyncIngestorInstance.Reset(mockOutputs)

	multiOutput := OutputFilesToPiazza(testWorkerConfig, "./run_algo --someArg 123 --anotherAlg value", "1.2.3test")

	assert.Len(t, multiOutput.Errors, 1)
}

func TestDetectPiazzaFileType(t *testing.T) {
	assert.Equal(t, "geojson", detectPiazzaFileType("something.geojson"))
	assert.Equal(t, "geojson", detectPiazzaFileType("SOMETHING_ELSE.GEOJSON"))
	assert.Equal(t, "raster", detectPiazzaFileType("image.tiff"))
	assert.Equal(t, "raster", detectPiazzaFileType("Image.GeoTiff"))
	assert.Equal(t, "text", detectPiazzaFileType("stuff.txt"))
	assert.Equal(t, "text", detectPiazzaFileType("abc123.unknownformat"))
	assert.Equal(t, "text", detectPiazzaFileType("no_extension"))
}
