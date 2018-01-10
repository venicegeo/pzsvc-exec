package input

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/pzsvc-worker/config"
)

var httpClient = http.Client{
	Timeout: 30 * time.Second,
}

// FetchInputs recovers and writes input files, using the input source configuration
func FetchInputs(session pzsvc.Session, inputs []config.InputSource) error {
	inputResults := []chan error{}
	for _, source := range inputs {
		errChan := downloadInputAsync(source)
		pzsvc.LogInfo(session, fmt.Sprintf("async downloading input: %s; from: %s", source.FileName, source.URL))
		inputResults = append(inputResults, errChan)
	}

	errors := []error{}

	for i, errChan := range inputResults {
		err := <-errChan
		if err != nil {
			errors = append(errors, fmt.Errorf("error downloading input: %s; %v", inputs[i].FileName, err))
		} else {
			pzsvc.LogInfo(session, fmt.Sprintf("downloaded input: %s", inputs[i].FileName))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

func downloadInputAsync(source config.InputSource) chan error {
	errChan := make(chan error)

	go func() {
		var err error
		defer close(errChan)

		_, fStatErr := os.Stat(source.FileName)
		if fStatErr == nil {
			err = fmt.Errorf("File already exists: %v", source.FileName)
		} else if fStatErr != os.ErrNotExist {
			err = fmt.Errorf("Error statting file: %v; %v", source.FileName, fStatErr)
		}
		if err != nil {
			errChan <- err
			return
		}

		resp, err := httpClient.Get(source.URL)
		if err == nil && resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("Unexpected HTTP status: %v", resp.StatusCode)
		}
		if err != nil {
			errChan <- err
			return
		}

		f, err := os.OpenFile(source.FileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
		if err != nil {
			errChan <- err
			return
		}

		_, err = io.Copy(f, resp.Body)
		if err != nil {
			errChan <- err
			return
		}

		err = f.Close()
		if err != nil {
			errChan <- err
			return
		}
	}()

	return errChan
}
