package workerexec

import (
	"encoding/json"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/worker/config"
	"github.com/venicegeo/pzsvc-exec/worker/log"
)

type piazzaOutputter struct {
	sendExecResultData func(pzsvc.Session, string, string, string, pzsvc.PiazzaStatus, []byte) *pzsvc.PzCustomError
}

func newPiazzaOutputter() *piazzaOutputter {
	return &piazzaOutputter{
		sendExecResultData: pzsvc.SendExecResultData,
	}
}

func (dpo piazzaOutputter) OutputToPiazza(cfg config.WorkerConfig, outData workerOutputData) error {
	serializedOutData, _ := json.Marshal(outData)
	workerlog.Info(cfg, "sending serialized output: "+string(serializedOutData))
	var jobStatus pzsvc.PiazzaStatus
	if len(outData.Errors) == 0 {
		jobStatus = pzsvc.PiazzaStatusSuccess
	} else {
		jobStatus = pzsvc.PiazzaStatusError
	}
	pzsvcErr := dpo.sendExecResultData(*cfg.Session, cfg.PiazzaBaseURL, cfg.PiazzaServiceID, cfg.JobID, jobStatus, serializedOutData)
	if pzsvcErr != nil {
		return pzsvcErr.Log(*cfg.Session, "failed to send result data")
	}
	return nil
}
