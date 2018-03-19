package input

import (
	"fmt"
	"os"
)

type fileChecker interface {
	CheckAndOpen(fileName string, fileMode os.FileMode) (*os.File, error)
}

type defaultFileChecker struct{}

// CheckAndOpen checks that a file of the given name does not already exist, then opens it for writing
func (dfc defaultFileChecker) CheckAndOpen(fileName string, fileMode os.FileMode) (*os.File, error) {
	_, fStatErr := os.Stat(fileName)
	if fStatErr == nil {
		return nil, fmt.Errorf("File already exists: %v", fileName)
	} else if !os.IsNotExist(fStatErr) {
		return nil, fmt.Errorf("Error statting file: %v; %v", fileName, fStatErr)
	}

	return os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
}

var fileCheckerInstance fileChecker = defaultFileChecker{}
