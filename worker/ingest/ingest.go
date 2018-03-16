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

package ingest

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/worker/config"
	"github.com/venicegeo/pzsvc-exec/worker/log"
)

// MultiIngestOutput holds response data for batch-ingesting several files
type MultiIngestOutput struct {
	DataIDs       map[string]string
	Errors        []error
	CombinedError error
}

type singleIngestOutput struct {
	FilePath string
	DataID   string
	Error    error
}

type asyncIngestorCall struct {
	s                                         pzsvc.Session
	filePath, fileType, serviceID, algVersion string
	attMap                                    map[string]string
}

// OutputFilesToPiazza ingests the given files into the Piazza system
func OutputFilesToPiazza(cfg config.WorkerConfig, algFullCommand string, algVersion string) MultiIngestOutput {
	ingestorCalls, asmErrors := assembleIngestorCalls(cfg, algFullCommand, algVersion)
	ingestorResultChans := callAsyncIngestor(ingestorCalls)
	return handleIngestResults(cfg, ingestorResultChans, asmErrors)
}

func assembleIngestorCalls(cfg config.WorkerConfig, algFullCommand string, algVersion string) ([]asyncIngestorCall, []error) {
	ingestorCalls := []asyncIngestorCall{}
	outputErrors := []error{}

	for _, filePath := range cfg.Outputs {
		workerlog.Info(cfg, "preparing ingest call: "+filePath)
		if _, fStatErr := os.Stat(filePath); fStatErr != nil {
			errMsg := fmt.Sprintf("error statting file `%s`: %v", filePath, fStatErr)
			workerlog.SimpleErr(cfg, errMsg, fStatErr)
			outputErrors = append(outputErrors, errors.New(errMsg))
			continue
		}

		fileType := detectPiazzaFileType(filePath)

		attMap := map[string]string{
			"algoName":     cfg.PiazzaServiceID,
			"algoVersion":  algVersion,
			"algoCmd":      algFullCommand,
			"algoProcTime": time.Now().UTC().Format("20060102.150405.99999"),
		}

		workerlog.Info(cfg, fmt.Sprintf("async ingest call: path=%s type=%s serviceID=%s, version=%s, attMap=%v",
			filePath, fileType, cfg.PiazzaServiceID, algVersion, attMap))

		ingestorCalls = append(ingestorCalls, asyncIngestorCall{*cfg.Session, filePath, fileType, cfg.PiazzaServiceID, algVersion, attMap})
	}
	return ingestorCalls, outputErrors
}

func callAsyncIngestor(ingestorCalls []asyncIngestorCall) (outputChans []<-chan singleIngestOutput) {
	ingestResultChans := []<-chan singleIngestOutput{}
	for _, call := range ingestorCalls {
		resultChan := asyncIngestorInstance.ingestFileAsync(call.s, call.filePath, call.fileType, call.serviceID, call.algVersion, call.attMap)
		ingestResultChans = append(ingestResultChans, resultChan)
	}
	return ingestResultChans
}

func handleIngestResults(cfg config.WorkerConfig, resultChans []<-chan singleIngestOutput, prependErrors []error) (multiOutput MultiIngestOutput) {
	multiOutput.DataIDs = map[string]string{}
	multiOutput.Errors = append([]error{}, prependErrors...)

	for _, resultChan := range resultChans {
		for result := range resultChan {
			if result.Error != nil {
				workerlog.SimpleErr(cfg, "received async ingest error", result.Error)
				multiOutput.Errors = append(multiOutput.Errors, result.Error)
			} else {
				workerlog.Info(cfg, fmt.Sprintf("ingested file `%s` as ID: %s", result.FilePath, result.DataID))
				multiOutput.DataIDs[result.FilePath] = result.DataID
			}
		}
	}

	if len(multiOutput.Errors) > 0 {
		errorTexts := []string{}
		for _, err := range multiOutput.Errors {
			errorTexts = append(errorTexts, err.Error())
		}
		fullErrorText := strings.Join(errorTexts, "; ")
		multiOutput.CombinedError = errors.New("ingest errors: " + fullErrorText)
		workerlog.SimpleErr(cfg, "concatenated ingest errors", multiOutput.CombinedError)
	}

	return
}
