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
