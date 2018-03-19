package input

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultFileChecker_OK(t *testing.T) {
	// Setup
	f, _ := ioutil.TempFile("", "test_filechecker")
	fileName := f.Name()
	f.Close()
	os.Remove(fileName)

	// Tested code
	fileChecker := defaultFileChecker{}
	openedFile, err := fileChecker.CheckAndOpen(fileName, 0777)

	// Asserts
	assert.Nil(t, err)
	if openedFile == nil {
		return
	}
	assert.Equal(t, fileName, openedFile.Name())

	// Teardown
	openedFile.Close()
	os.Remove(openedFile.Name())
}

func TestDefaultFileChecker_FileAlreadyExists(t *testing.T) {
	// Setup
	f, _ := ioutil.TempFile("", "test_filechecker")
	fileName := f.Name()
	f.Close()

	// Tested code
	fileChecker := defaultFileChecker{}
	_, err := fileChecker.CheckAndOpen(fileName, 0777)

	// Asserts
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Teardown
	os.Remove(fileName)
}
