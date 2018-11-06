package input

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/venicegeo/pzsvc-exec/worker/config"
)

type mockAsyncDownloader struct {
	ReturnErrors chan error
	Calls        []config.InputSource
}

func newMockAsyncDownloader(errors []error) *mockAsyncDownloader {
	dl := &mockAsyncDownloader{make(chan error), []config.InputSource{}}
	go func() {
		for _, err := range errors {
			dl.ReturnErrors <- err
		}
		close(dl.ReturnErrors)
	}()
	return dl
}

func (dl *mockAsyncDownloader) DownloadInputAsync(source config.InputSource) chan error {
	dl.Calls = append(dl.Calls, source)
	returnErrChan := make(chan error)
	go func() {
		defer close(returnErrChan)
		if err, ok := <-dl.ReturnErrors; ok {
			returnErrChan <- err
		} else {
			returnErrChan <- nil
		}
	}()
	return returnErrChan
}

func TestFetchInputs_NoErrors(t *testing.T) {
	// Setup
	mockAsyncDownloader := newMockAsyncDownloader([]error{})
	oldAsyncDownloaderInstance := asyncDownloaderInstance
	asyncDownloaderInstance = mockAsyncDownloader
	workerConfig := config.WorkerConfig{MuteLogs: true}
	inputs := []config.InputSource{
		config.InputSource{FileName: "text.txt", URL: "http://example.localdomain/foobar.txt"},
		config.InputSource{FileName: "image.tif", URL: "https://example2.localdomain/foobar.tif"},
	}

	// Tested code
	err := FetchInputs(workerConfig, inputs)

	// Asserts
	assert.Nil(t, err)
	assert.Len(t, mockAsyncDownloader.Calls, len(inputs))
	for _, input := range inputs {
		foundInputInCalls := false
		for _, call := range mockAsyncDownloader.Calls {
			foundInputInCalls = foundInputInCalls || (call == input)
		}
		assert.True(t, foundInputInCalls)
	}

	// Teardown
	asyncDownloaderInstance = oldAsyncDownloaderInstance
}

func TestFetchInputs_Errors(t *testing.T) {
	// Setup
	mockAsyncDownloader := newMockAsyncDownloader([]error{nil, errors.New("test error text"), nil})
	oldAsyncDownloaderInstance := asyncDownloaderInstance
	asyncDownloaderInstance = mockAsyncDownloader
	workerConfig := config.WorkerConfig{MuteLogs: true}
	inputs := []config.InputSource{
		config.InputSource{FileName: "text.txt", URL: "http://example.localdomain/foobar.txt"},
		config.InputSource{FileName: "image.tif", URL: "https://example2.localdomain/foobar.tif"},
		config.InputSource{FileName: "anotherimage.jp2", URL: "https://example3.localdomain/foobar.jp2"},
	}

	// Tested code
	err := FetchInputs(workerConfig, inputs)

	// Asserts
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "error downloading source imagery")
	assert.Len(t, mockAsyncDownloader.Calls, len(inputs))

	for _, input := range inputs {
		foundInputInCalls := false
		for _, call := range mockAsyncDownloader.Calls {
			foundInputInCalls = foundInputInCalls || (call == input)
		}
		assert.True(t, foundInputInCalls)
	}

	// Teardown
	asyncDownloaderInstance = oldAsyncDownloaderInstance
}
