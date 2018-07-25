package input

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/venicegeo/pzsvc-exec/worker/config"
)

type mockDownloadHandler struct {
	ValidPaths  []string
	CalledPaths *[]string
}

func (h mockDownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	*h.CalledPaths = append(*h.CalledPaths, r.URL.EscapedPath())
	for _, validPath := range h.ValidPaths {
		if validPath == r.URL.EscapedPath() {
			w.Write([]byte("test data"))
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	return
}

type mockFileChecker struct {
	tempFile  *os.File
	fileError error
}

func newMockFileChecker(fileError error) *mockFileChecker {
	tempFile, err := ioutil.TempFile("", "test_file")
	if err != nil {
		panic(err)
	}
	return &mockFileChecker{tempFile, fileError}
}

func (mfc *mockFileChecker) CheckAndOpen(fileName string, fileMode os.FileMode) (*os.File, error) {
	if mfc.fileError != nil {
		return nil, mfc.fileError
	}
	return mfc.tempFile, nil
}

func TestDefaultAsyncDownloader_OK(t *testing.T) {
	// Setup
	handler := mockDownloadHandler{[]string{"/ok1.txt"}, &[]string{}}
	server := httptest.NewServer(handler)
	defer server.Close()

	mockFileCheckerInstance := newMockFileChecker(nil)
	defer os.Remove(mockFileCheckerInstance.tempFile.Name())
	oldFileChecker := fileCheckerInstance
	fileCheckerInstance = mockFileCheckerInstance

	inputSource := config.InputSource{FileName: "file1.txt", URL: server.URL + "/ok1.txt"}

	// Tested code
	downloader := defaultAsyncDownloader{}
	errChan := downloader.DownloadInputAsync(inputSource)

	// Asserts
	select {
	case err, ok := <-errChan:
		if ok {
			assert.Fail(t, "unexpected error from async download channel: "+err.Error())
		}
	case <-time.After(1 * time.Second):
		assert.Fail(t, "failed to download from mock server for 1 second")
	}

	writtenFile, _ := os.Open(mockFileCheckerInstance.tempFile.Name())
	writtenData, _ := ioutil.ReadAll(writtenFile)
	assert.Equal(t, "test data", string(writtenData))

	// Teardown
	fileCheckerInstance = oldFileChecker
}

func TestDefaultAsyncDownloader_HTTPError(t *testing.T) {
	// Setup
	handler := mockDownloadHandler{[]string{"/ok1.txt"}, &[]string{}}
	server := httptest.NewServer(handler)
	defer server.Close()

	mockFileCheckerInstance := newMockFileChecker(nil)
	defer os.Remove(mockFileCheckerInstance.tempFile.Name())
	oldFileChecker := fileCheckerInstance
	fileCheckerInstance = mockFileCheckerInstance

	inputSource := config.InputSource{FileName: "file1.txt", URL: server.URL + "/notfound.txt"}

	// Tested code
	downloader := defaultAsyncDownloader{ Retries: 3 }
	errChan := downloader.DownloadInputAsync(inputSource)

	// Asserts
	select {
	case err, ok := <-errChan:
		if !ok {
			assert.Fail(t, "errChan returned no error, expected failure")
		}
		assert.NotNil(t, err)
	case <-time.After(1 * time.Second):
		assert.Fail(t, "failed to download from mock server for 1 second")
	}

	// Teardown
	fileCheckerInstance = oldFileChecker
}

func TestDefaultAsyncDownloader_FileError(t *testing.T) {
	// Setup
	handler := mockDownloadHandler{[]string{"/ok1.txt"}, &[]string{}}
	server := httptest.NewServer(handler)
	defer server.Close()

	mockFileCheckerInstance := newMockFileChecker(errors.New("test file error"))
	defer os.Remove(mockFileCheckerInstance.tempFile.Name())
	oldFileChecker := fileCheckerInstance
	fileCheckerInstance = mockFileCheckerInstance

	inputSource := config.InputSource{FileName: "file1.txt", URL: server.URL + "/ok1.txt"}

	// Tested code
	downloader := defaultAsyncDownloader{}
	errChan := downloader.DownloadInputAsync(inputSource)

	// Asserts
	select {
	case err, ok := <-errChan:
		if !ok {
			assert.Fail(t, "errChan returned no error, expected failure")
		}
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "test file error")
	case <-time.After(1 * time.Second):
		assert.Fail(t, "failed to download from mock server for 1 second")
	}

	// Teardown
	fileCheckerInstance = oldFileChecker
}
