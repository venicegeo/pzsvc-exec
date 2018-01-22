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

const ingestTimeout = 1 * time.Minute

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
		resultChan := ingestFileAsync(*cfg.Session, filePath, fileType, cfg.PiazzaServiceID, algVersion, attMap)
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

func ingestFileAsync(s pzsvc.Session, filePath string, fileType string,
	serviceID string, algVersion string, attMap map[string]string) <-chan singleIngestOutput {
	outChan := make(chan singleIngestOutput)

	// Nested goroutines to allow for 1 minute for ingestion to succeed
	go func() {
		resultChan := make(chan singleIngestOutput)
		go func() {
			dataID, err := pzsvc.IngestFile(s, filePath, fileType, serviceID, algVersion, attMap)

			resultChan <- singleIngestOutput{
				FilePath: filePath,
				DataID:   dataID,
				Error:    err,
			}
			close(resultChan)
		}()
		select {
		case result := <-resultChan:
			outChan <- result
		case <-time.After(ingestTimeout):
			outChan <- singleIngestOutput{
				FilePath: filePath,
				Error:    errors.New("File ingest timed out"),
			}
		}
		close(outChan)
	}()

	return outChan
}

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
