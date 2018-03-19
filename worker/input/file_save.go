package input

import (
	"io"
	"os"
)

type fileSaver interface {
	CopyTo(fileName string, dataStream io.ReadCloser, fileMode os.FileMode) error
}

type defaultFileSaver struct{}

// CopyTo copies the contents of dataStream to a file at the given filename, and closes the data stream
func (fs defaultFileSaver) CopyTo(fileName string, dataStream io.ReadCloser, fileMode os.FileMode) error {
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, dataStream)
	defer dataStream.Close()
	if err != nil {
		return err
	}

	return f.Close()
}

var fileSaverInstance fileSaver = defaultFileSaver{}
