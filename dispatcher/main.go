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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/venicegeo/pzsvc-exec/pzse"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/cloudfoundry-community/go-cfclient"
)

func main() {
	// Initialization Block
	s := pzsvc.Session{AppName: "Dispatcher", SessionID: "Startup", LogRootDir: "pzsvc-exec"}
	pzsvc.LogAudit(s, s.AppName, "startup", s.AppName, "", pzsvc.INFO)

	if len(os.Args) < 2 {
		pzsvc.LogSimpleErr(s, "error: Insufficient parameters.  You must specify a config file.", nil)
		os.Exit(1)
	}

	// First argument after the base call should be the path to the config file.
	// ReadFile returns the contents of the file as a byte buffer.
	configPath := os.Args[1]
	configBuf, err := ioutil.ReadFile(configPath)
	if err != nil {
		pzsvc.LogSimpleErr(s, "Dispatcher error in reading config: ", err)
		return
	}
	var configObj pzse.ConfigType
	err = json.Unmarshal(configBuf, &configObj)
	if err != nil {
		pzsvc.LogSimpleErr(s, "Dispatcher error in unmarshalling config: ", err)
		return
	}

	s.LogAudit = configObj.LogAudit
	if configObj.LogAudit {
		pzsvc.LogInfo(s, "Config: Audit logging enabled.")
	} else {
		pzsvc.LogInfo(s, "Config: Audit logging disabled.")
	}

	s.PzAddr = configObj.PzAddr
	if configObj.PzAddrEnVar != "" {
		newAddr := os.Getenv(configObj.PzAddrEnVar)
		if newAddr != "" {
			s.PzAddr = newAddr
		}
	}
	if s.PzAddr == "" {
		pzsvc.LogSimpleErr(s, "Config: Cannot work tasks.  Must have either a valid PzAddr, or a valid and populated PzAddrEnVar.", nil)
		return
	}

	if configObj.SvcName == "" {
		pzsvc.LogSimpleErr(s, "Config: Cannot work tasks without service name.", nil)
		return
	}

	if configObj.APIKeyEnVar == "" {
		pzsvc.LogSimpleErr(s, "Config: Cannot work tasks without valid APIKeyEnVar.", nil)
		return
	}
	apiKey := os.Getenv(configObj.APIKeyEnVar)
	if apiKey == "" {
		pzsvc.LogSimpleErr(s, "No API key at APIKeyEnVar.  Cannot work.", nil)
		return
	}
	s.PzAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(apiKey+":"))


	svcID := ""
	for i := 0; svcID == "" && i < 10; i++ {
		svcID, err = pzsvc.FindMySvc(s, configObj.SvcName)
		if err != nil {
			pzsvc.LogSimpleErr(s, "Dispatcher could not find Pz Service ID.  Initial Error: ", err)
			return
		}
		if svcID == "" && i < 9 {
			pzsvc.LogInfo(s, "Could not find service.  Will sleep and wait.")
			time.Sleep(15 * time.Second)
		}
	}
	if svcID == "" {
		pzsvc.LogSimpleErr(s, "Dispatcher could not find Pz Service ID.  Ensure the Service exists and is registered, and try again.", err)
		return
	}

	pzsvc.LogInfo(s, "Found target service.  ServiceID: "+ svcID + ".")

	// Initialize the CF Client
	clientConfig := &cfclient.Config{
		ApiAddress:   os.Getenv("CF_API"),
		Username:     os.Getenv("CF_USER"),
		Password:     os.Getenv("CF_PASS")
	}
	client, err := cfclient.NewClient(c)
	if err != nil {
		pzsvc.LogSimpleErr(s, "Error in Inflating Cloud Foundry API Client: ", err)
		return
	}

	pzsvc.LogInfo(s, "Cloud Foundry Client initialized. Beginning Polling.")

	pollForJobs(s, configObj, svcID, configPath, client)
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

func pollForJobs(s pzsvc.Session, configObj pzse.ConfigType, svcID string, configPath string, cfClient Client) {
	var (
		err       error
	)
	s.SessionID = "Polling"

	// Get the application name
	vcapJsonContainer := make(map[string]interface{})
	err := json.Unmarshal(os.Getenv("VCAP_APPLICATION"), &vcapJsonContainer)
	if pErr != nil {
		pzsvc.LogSimpleErr(s, "Cannot proceed: Error in reading VCAP Application properties: ", err)
		return
	}
	appName, err := vcapJsonContainer["application_name"].(string)
	if pErr != nil {
		pzsvc.LogSimpleErr(s, "Cannot Read Application Name from VCAP Application properties: ", err)
		return
	}

	for {
		var pzJobObj struct {
			Data WorkOutData `json:"data"`
		}
		pzJobObj.Data = WorkOutData{SvcData: WorkSvcData{JobID: "", Data: WorkInData{DataInputs: WorkDataInputs{Body: WorkBody{Content: ""}}}}}

		byts, pErr := pzsvc.RequestKnownJSON("POST", "", s.PzAddr+"/service/"+svcID+"/task", s.PzAuth, &pzJobObj)
		if pErr != nil {
			pErr.Log(s, "Dispatcher: error getting new task:")
			time.Sleep(time.Duration(5) * time.Second)
			continue
		}

		inpStr := pzJobObj.Data.SvcData.Data.DataInputs.Body.Content
		jobID := pzJobObj.Data.SvcData.JobID
		if inpStr != "" {
			pzsvc.LogInfo(s, "New Task Grabbed.  JobID: "+jobID)

			var outpByts []byte

			var respObj pzse.OutStruct
			var jobInputContent pzse.InpStruct
			var displayByt []byte
			err = json.Unmarshal([]byte(inpStr), &jobInputContent)
			if err == nil {
				if jobInputContent.ExtAuth != "" {
					jobInputContent.ExtAuth = "*****"
				}
				if jobInputContent.PzAuth != "" {
					jobInputContent.PzAuth = "*****"
				}
				displayByt, err = json.Marshal(jobInputContent)
				if err != nil {
					pzsvc.LogAudit(s, s.UserID, "Audit failure", s.AppName, "Could not Marshal.  Job Canceled.", pzsvc.ERROR)
					sendExecResult(s, s.PzAddr, s.PzAuth, svcID, jobID, "Fail", nil)
					time.Sleep(5 * time.Second)
					continue
				}
			}

			pzsvc.LogAudit(s, s.UserID, "Creating CF Task for Job" + jobID, s.AppName, string(displayByt), pzsvc.INFO)

			// Form the CLI for the Algorithm Task
			workerCommand := fmt.Sprintf("worker --cliCmd \"%s\" --userId \"%s\" --config \"%s\" --serviceId \"%s\"", jobInputContent.Command, jobInputContent.UserID, configPath, svcID)
			// For each input image, add that image ref as an argument to the CLI
			for i, imageFile := range jobInputContent.InExtFiles {
				workerCommand += fmt.Sprintf(" -i %s:%s", jobInputContent.InExtNames[i], jobInputContent.InExtFiles[i])
			}

			taskRequest := TaskRequest{
				Command: workerCommand
				Name: jobID
				MemoryInMegabyte: 0
				DiskInMegabyte: 0
				DropletGUID: jobID
			}

			// Send Run-Task request to CF
			task, error := cfClient.CreateTask(taskRequest)
			if err != nil {
				pzsvc.LogAudit(s, s.UserID, "Audit failure", s.AppName, "Could not Create PCF Task for Job. Job Failed.", pzsvc.ERROR)
				sendExecResult(s, s.PzAddr, s.PzAuth, svcID, jobID, "Fail", nil)
				time.Sleep(5 * time.Second)
				continue
			}

			pzsvc.LogAudit(s, s.UserID, "Task Created for CF Job" + , s.AppName, string(displayByt), pzsvc.INFO)

			time.Sleep(5 * time.Second)
		} else {
			pzsvc.LogInfo(s, "No Task.  Sleeping now.  input: "+string(byts))
			time.Sleep(5 * time.Second)
		}
	}

}

func sendExecResult(s pzsvc.Session, pzAddr, pzAuth, svcID, jobID, status string, resJSON []byte) {
	outAddr := pzAddr + `/service/` + svcID + `/task/` + jobID

	pzsvc.LogInfo(s, "Sending Exec Results.  Status: "+status+".")
	if resJSON != nil {
		dataID, err := pzsvc.Ingest(s, "Output", "text", "Dispatcher", "", resJSON, nil)
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
