// Copyright 2016, RadiantBlue Technologies, Inc.
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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/venicegeo/pzsvc-exec/pzse"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

func main() {

	s := pzsvc.Session{AppName: "pzsvc-taskworker", SessionID: "startup", LogRootDir: "pzsvc-exec"}
	pzsvc.LogAudit(s, s.AppName, "startup", s.AppName)

	if len(os.Args) < 2 {
		pzsvc.LogSimpleErr(s, "error: Insufficient parameters.  You must specify a config file.", nil)
		return
	}

	// First argument after the base call should be the path to the config file.
	// ReadFile returns the contents of the file as a byte buffer.
	configBuf, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		pzsvc.LogSimpleErr(s, "pzsvc-taskworker error in reading config: ", err)
		return
	}
	var configObj pzse.ConfigType
	err = json.Unmarshal(configBuf, &configObj)
	if err != nil {
		pzsvc.LogSimpleErr(s, "pzsvc-taskworker error in unmarshalling config: ", err)
		return
	}

	s.LogAudit = configObj.LogAudit
	if configObj.LogAudit {
		pzsvc.LogInfo(s, "Config: Audit logging enabled.")
	} else {
		pzsvc.LogInfo(s, "Config: Audit logging disabled.")
	}

	s.PzAddr = configObj.PzAddr
	if configObj.PzAddr == "" {
		pzsvc.LogSimpleErr(s, "Config: Cannot work tasks without valid PzAddr.", nil)
		return
	}

	if configObj.SvcName == "" {
		pzsvc.LogSimpleErr(s, "Config: Cannot work tasks without service name.", nil)
		return
	}

	if configObj.AuthEnVar == "" {
		pzsvc.LogSimpleErr(s, "Config: Cannot work tasks without valid AuthEnVar.", nil)
		return
	}
	s.PzAuth = os.Getenv(configObj.AuthEnVar)
	if s.PzAuth == "" {
		pzsvc.LogSimpleErr(s, "No Auth key at AuthEnVar.  Cannot work.", nil)
		return
	}

	if configObj.NumProcs == 0 {
		pzsvc.LogInfo(s, "Config: No Proc number specified.  Defaulting to one at a time.")
		configObj.NumProcs = 1
	}

	if configObj.Port == 0 {
		pzsvc.LogInfo(s, "Config: No target Port specified.  Defaulting to 8080.")
		configObj.Port = 8080
	}

	svcID, err := pzsvc.FindMySvc(s, configObj.SvcName, configObj.PzAddr, s.PzAuth)
	if err != nil || svcID == "" {
		pzsvc.LogSimpleErr(s, "Taskworker could not find Pz Service ID.  Initial Error: ", err)
		return
	}
	pzsvc.LogInfo(s, "Found target service.  ServiceID: "+svcID)
	for i := 0; i < configObj.NumProcs; i++ {
		go workerThread(s, configObj, svcID)
	}
	select {} //blocks forever

}

// WorkBody exists as part of the response format of the Piazza job manager task request endpoint.
// specifically, it's one layer of the bit we care about.
type WorkBody struct {
	Content string `json:"content"`
}

// WorkDataInputs exists as part of the response format of the Piazza job manager task request endpoint.
// specifically, it's one layer of the bit we care about.
type WorkDataInputs struct {
	Body WorkBody `json:"body"`
}

// WorkInData exists as part of the response format of the Piazza job manager task request endpoint.
// specifically, it's one layer of the bit we care about.
type WorkInData struct {
	DataInputs WorkDataInputs `json:"dataInputs"`
}

// WorkSvcData exists as part of the response format of the Piazza job manager task request endpoint.
// specifically, it's one layer of the bit we care about.
type WorkSvcData struct {
	Data  WorkInData `json:"data"`
	JobID string     `json:"jobId"`
}

// WorkOutData exists as part of the response format of the Piazza job manager task request endpoint.
// specifically, it's one layer of the bit we care about.
type WorkOutData struct {
	SvcData WorkSvcData `json:"serviceData"`
}

func workerThread(s pzsvc.Session, configObj pzse.ConfigType, svcID string) {

	var err error
	workAddr := fmt.Sprintf("http://localhost:%d/execute", configObj.Port)

	s.SessionID, err = pzsvc.PsuUUID()
	if err != nil {
		s.SessionID = "FailedSessionInit"
		pzsvc.LogSimpleErr(s, "psuUUID error: ", err)
		panic("Worker thread failed on uid generation.  Something is very wrong: " + err.Error())
	}
	pzsvc.LogInfo(s, "Worker thread initiated.")

	for {
		var pzJobObj struct {
			Data WorkOutData `json:"data"`
		}
		pzJobObj.Data = WorkOutData{SvcData: WorkSvcData{JobID: "", Data: WorkInData{DataInputs: WorkDataInputs{Body: WorkBody{Content: ""}}}}}

		byts, pErr := pzsvc.RequestKnownJSON("POST", "", configObj.PzAddr+"/service/"+svcID+"/task", s.PzAuth, &pzJobObj)
		if pErr != nil {
			pErr.Log(s, "Taskworker worker thread: error getting new task:")
			continue
		}
		inpStr := pzJobObj.Data.SvcData.Data.DataInputs.Body.Content
		jobID := pzJobObj.Data.SvcData.JobID
		if inpStr != "" {
			pzsvc.LogInfo(s, "New Task Grabbed.  JobID: "+jobID)

			var outpByts []byte
			if configObj.JwtSecAuthURL != "" {
				// TODO: once JWT conversion exists as an option, handle it here.
				// jwtBody = content
				// call JwtSecAuthURL.  send jwtBody.  get response
				// outBody = response (more or less)
			}

			var respObj pzse.OutStruct
			pzsvc.LogAuditBuf(s, s.UserID, "http request - calling pzsvc-exec", inpStr, workAddr)
			outpByts, pErr := pzsvc.RequestKnownJSON("POST", inpStr, workAddr, "", &respObj)
			if pErr != nil {
				pErr.Log(s, "Error calling pzsvc-exec")
				sendExecResult(s, configObj.PzAddr, s.PzAuth, svcID, jobID, "Fail", nil)
			} else {
				pzsvc.LogAuditBuf(s, workAddr, "http response from pzsvc-exec", string(outpByts), s.UserID)
				sendExecResult(s, configObj.PzAddr, s.PzAuth, svcID, jobID, "Success", outpByts)
			}

		} else {
			pzsvc.LogInfo(s, "No Task.  Sleeping now.  input: "+string(byts))
			time.Sleep(60 * time.Second)
		}
	}

}

func sendExecResult(s pzsvc.Session, pzAddr, pzAuth, svcID, jobID, status string, resJSON []byte) {
	outAddr := pzAddr + `/service/` + svcID + `/task/` + jobID

	pzsvc.LogInfo(s, "Sending Exec Results.  Status: "+status+".")
	if resJSON != nil {
		dataID, err := pzsvc.Ingest(s, "Output", "text", "pzsvc-taskworker", "", resJSON, nil)
		if err == nil {
			outStr := `{ "status" : "` + status + `", "result" : { "type" : "data", "dataId" : "` + dataID + `" } }`
			pzsvc.SubmitSinglePart("POST", outStr, outAddr, s.PzAuth)
			return
		}
		pzsvc.LogInfo(s, "Send Exec Results: Ingest failed.")
		status = "Fail"
	}

	outStr := `{ "status" : "` + status + `" }`
	pzsvc.SubmitSinglePart("POST", outStr, outAddr, s.PzAuth)
}
