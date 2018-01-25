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

package workerexec

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/worker/config"
	"github.com/venicegeo/pzsvc-exec/worker/ingest"
	"github.com/venicegeo/pzsvc-exec/worker/input"
	"github.com/venicegeo/pzsvc-exec/worker/log"
)

// WorkerExec runs the main worker exec subprocess
func WorkerExec(cfg config.WorkerConfig) (err error) {
	outData := workerOutputData{
		InFiles:    map[string]string{},
		OutFiles:   map[string]string{},
		HTTPStatus: http.StatusOK,
	}

	workerlog.Info(cfg, "Fetching inputs")
	err = input.FetchInputs(cfg, cfg.Inputs)
	if err != nil {
		workerlog.SimpleErr(cfg, "Failed to fetch inputs", err)
		outData.AddErrors(err)
		outData.HTTPStatus = http.StatusInternalServerError
		return sendPiazzaJobOutput(cfg, outData)
	}
	outData.InFiles = cfg.InputsAsMap()
	workerlog.Info(cfg, "Inputs fetched")

	workerlog.Info(cfg, "Running version command")
	versionCmdOutput := runCommand(cfg, cfg.PzSEConfig.VersionCmd)
	if versionCmdOutput.Error != nil {
		workerlog.SimpleErr(cfg, "Failed to get algorithm version", versionCmdOutput.Error)
		outData.AddErrors(versionCmdOutput.Error)
		outData.HTTPStatus = http.StatusInternalServerError
		outData.ProgStdErr = string(versionCmdOutput.Stderr)
		return sendPiazzaJobOutput(cfg, outData)
	}
	version := strings.TrimSpace(string(versionCmdOutput.Stdout))
	workerlog.Info(cfg, "Retrieved algorithm version: "+version)

	fullCommand := strings.Join([]string{cfg.PzSEConfig.CliCmd, cfg.CLICommandExtra}, " ")
	workerlog.Info(cfg, "Running algorithm command: "+fullCommand)
	algCmdOutput := runCommand(cfg, fullCommand)
	outData.ProgStdOut = string(algCmdOutput.Stdout)
	outData.ProgStdErr = string(algCmdOutput.Stderr)
	if algCmdOutput.Error != nil {
		workerlog.SimpleErr(cfg, "Failed running algorithm command", algCmdOutput.Error)
		outData.AddErrors(algCmdOutput.Error)
		outData.HTTPStatus = http.StatusInternalServerError
		return sendPiazzaJobOutput(cfg, outData)
	}
	workerlog.Info(cfg, "Algorithm command successful")

	workerlog.Info(cfg, "Ingesting output files to Piazza")
	ingestOutput := ingest.OutputFilesToPiazza(cfg, fullCommand, version)
	if ingestOutput.CombinedError != nil {
		workerlog.SimpleErr(cfg, "Received combined error from ingestion", ingestOutput.CombinedError)
		outData.AddErrors(ingestOutput.Errors...)
		outData.HTTPStatus = http.StatusInternalServerError
		return sendPiazzaJobOutput(cfg, outData)
	}
	outData.OutFiles = ingestOutput.DataIDs
	workerlog.Info(cfg, "Ingest successful")

	workerlog.Info(cfg, "Setting successful Piazza job")
	err = sendPiazzaJobOutput(cfg, outData)
	workerlog.Info(cfg, "Piazza job status updated, worker execution finished")

	return
}

func sendPiazzaJobOutput(cfg config.WorkerConfig, outData workerOutputData) error {
	serializedOutData, _ := json.Marshal(outData)
	workerlog.Info(cfg, "sending serialized output: "+string(serializedOutData))
	var jobStatus pzsvc.PiazzaStatus
	if len(outData.Errors) == 0 {
		jobStatus = pzsvc.PiazzaStatusSuccess
	} else {
		jobStatus = pzsvc.PiazzaStatusError
	}
	pzsvcErr := pzsvc.SendExecResultData(*cfg.Session, cfg.PiazzaBaseURL, cfg.PiazzaServiceID, cfg.JobID, jobStatus, serializedOutData)
	if pzsvcErr != nil {
		return pzsvcErr.Log(*cfg.Session, "failed to send result data")
	}
	return nil
}
