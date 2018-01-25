// Copyright 2018, RadiantBlue Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package input

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/venicegeo/pzsvc-exec/worker/config"
	"github.com/venicegeo/pzsvc-exec/worker/log"
)

var httpClient = http.Client{
	Timeout: 30 * time.Second,
}

// FetchInputs recovers and writes input files, using the input source configuration
func FetchInputs(cfg config.WorkerConfig, inputs []config.InputSource) error {
	inputResults := []chan error{}
	for _, source := range inputs {
		errChan := downloadInputAsync(source)
		workerlog.Info(cfg, fmt.Sprintf("async downloading input: %s; from: %s", source.FileName, source.URL))
		inputResults = append(inputResults, errChan)
	}

	errors := []error{}

	for i, errChan := range inputResults {
		err := <-errChan
		if err != nil {
			errors = append(errors, fmt.Errorf("error downloading input: %s; %v", inputs[i].FileName, err))
		} else {
			workerlog.Info(cfg, fmt.Sprintf("downloaded input: %s", inputs[i].FileName))
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
		} else if !os.IsNotExist(fStatErr) {
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
