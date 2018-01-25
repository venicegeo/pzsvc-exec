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

package pzsvc

import (
	"encoding/json"
	"fmt"
)

type statusUpdateJSON struct {
	Status PiazzaStatus            `json:"status"`
	Result *statusUpdateResultJSON `json:"result,omitempty"`
}

type statusUpdateResultJSON struct {
	Type   string `json:"type"`
	DataID string `json:"dataId"`
}

// SendExecResultNoData sends the result of a job execution to Piazza
func SendExecResultNoData(s Session, pzAddr, svcID, jobID string, status PiazzaStatus) *Error {
	outAddr := fmt.Sprintf("%s/service/%s/task/%s", pzAddr, svcID, jobID)

	LogInfo(s, fmt.Sprintf("Sending exec results, no body data. URL=%s Status=%s ", outAddr, status))
	outData := statusUpdateJSON{Status: status}
	outJSON, _ := json.Marshal(outData)

	_, err := SubmitSinglePart("POST", string(outJSON), outAddr, s.PzAuth)
	return err
}

// SendExecResultData sends the result of a job execution to Piazza, including extra text data
func SendExecResultData(s Session, pzAddr, svcID, jobID string, status PiazzaStatus, resultData []byte) *Error {
	outAddr := pzAddr + `/service/` + svcID + `/task/` + jobID
	LogInfo(s, fmt.Sprintf("Sending exec results, with body data. URL=%s Status=%s ", outAddr, status))
	outData := statusUpdateJSON{Status: status}

	LogInfo(s, "Sending exec result: Ingesting body data...")
	dataID, err := Ingest(s, "Output", "text", "pzsvc-taskworker", "", resultData, nil)

	if err != nil {
		LogInfo(s, "Sending exec result: Ingestion failed.")
		outData.Status = PiazzaStatusFail
	} else {
		LogInfo(s, "Sending exec result: Ingestion succeeded.")
		outData.Result = &statusUpdateResultJSON{Type: "data", DataID: dataID}
	}

	outJSON, _ := json.Marshal(outData)
	_, httpErr := SubmitSinglePart("POST", string(outJSON), outAddr, s.PzAuth)
	return httpErr
}
