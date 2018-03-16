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
	"path/filepath"
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

// OutputFilesToPiazza ingests the given files into the Piazza system
func OutputFilesToPiazza(cfg config.WorkerConfig, algFullCommand string, algVersion string) (output MultiIngestOutput) {
	output.DataIDs = map[string]string{}
	ingestResultChans := []<-chan singleIngestOutput{}

	for _, filePath := range cfg.Outputs {
		workerlog.Info(cfg, "ingesting file to Piazza: "+filePath)
		if _, fStatErr := os.Stat(filePath); fStatErr != nil {
			errMsg := fmt.Sprintf("error statting file `%s`: %v", filePath, fStatErr)
			workerlog.SimpleErr(cfg, errMsg, fStatErr)
			output.Errors = append(output.Errors, errors.New(errMsg))
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
		resultChan := asyncIngestorInstance.ingestFileAsync(*cfg.Session, filePath, fileType, cfg.PiazzaServiceID, algVersion, attMap)
		ingestResultChans = append(ingestResultChans, resultChan)
	}

	for _, resultChan := range ingestResultChans {
		for result := range resultChan {
			if result.Error != nil {
				workerlog.SimpleErr(cfg, "received async ingest error", result.Error)
				output.Errors = append(output.Errors, result.Error)
			} else {
				workerlog.Info(cfg, fmt.Sprintf("ingested file `%s` as ID: %s", result.FilePath, result.DataID))
				output.DataIDs[result.FilePath] = result.DataID
			}
		}
	}

	if len(output.Errors) > 0 {
		errorTexts := []string{}
		for _, err := range output.Errors {
			errorTexts = append(errorTexts, err.Error())
		}
		fullErrorText := strings.Join(errorTexts, "; ")
		output.CombinedError = errors.New("ingest errors: " + fullErrorText)
		workerlog.SimpleErr(cfg, "concatenated ingest errors", output.CombinedError)
	}

	return
}

// pzSvcIngestor is an interface providing mock-able pzsvc.IngestFile functionality, for modularity/testing purposes
type pzSvcIngestor interface {
	IngestFile(s pzsvc.Session, fName, fType, sourceName, version string, props map[string]string) (string, pzsvc.LoggedError)
	Timeout() <-chan time.Time
}

type defaultPzSvcIngestor struct{}

func (ingestor defaultPzSvcIngestor) IngestFile(s pzsvc.Session, fName, fType, sourceName, version string, props map[string]string) (string, pzsvc.LoggedError) {
	return pzsvc.IngestFile(s, fName, fType, sourceName, version, props)
}

func (ingestor defaultPzSvcIngestor) Timeout() <-chan time.Time {
	return time.After(1 * time.Minute)
}

var pzSvcIngestorInstance pzSvcIngestor = &defaultPzSvcIngestor{}

func detectPiazzaFileType(fileName string) string {
	ext := filepath.Ext(strings.ToLower(fileName))

	switch ext {
	case ".geojson":
		return "geojson"
	case ".tiff", ".geotiff":
		return "raster"
	default:
		return "text"
	}
}
