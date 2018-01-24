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
	"net/url"
	"os"
	"strconv"
	"time"
	"strings"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/venicegeo/pzsvc-exec/pzse"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
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

	pzsvc.LogInfo(s, "Found target service.  ServiceID: "+svcID+".")

	// Initialize the CF Client
	clientConfig := &cfclient.Config{
		ApiAddress: os.Getenv("CF_API"),
		Username:   os.Getenv("CF_USER"),
		Password:   os.Getenv("CF_PASS"),
	}
	client, err := cfclient.NewClient(clientConfig)
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

func pollForJobs(s pzsvc.Session, configObj pzse.ConfigType, svcID string, configPath string, cfClient *cfclient.Client) {
	var (
		err error
	)
	s.SessionID = "Polling"

	// Get the application name
	vcapJSONContainer := make(map[string]interface{})
	err = json.Unmarshal([]byte(os.Getenv("VCAP_APPLICATION")), &vcapJSONContainer)
	if err != nil {
		pzsvc.LogSimpleErr(s, "Cannot proceed: Error in reading VCAP Application properties: ", err)
		return
	}
	appID, ok := vcapJSONContainer["application_id"].(string)
	if !ok {
		pzsvc.LogSimpleErr(s, "Cannot Read Application Name from VCAP Application properties: string type assertion failed", nil)
		return
	}
	pzsvc.LogInfo(s, "Found application name from VCAP Tree: "+appID)

	// Read the # of simultaneous Tasks that are allowed to be run by the Dispatcher
	taskLimit := 5
	if envTaskLimit := os.Getenv("TASK_LIMIT"); envTaskLimit != "" {
		taskLimit, _ = strconv.Atoi(envTaskLimit)
	}

	// Polling Loop
	for {
		// First, check to see if there is room for tasks. If we've reached the task limit, then do not poll Piazza for jobs.
		query := url.Values{}
		query.Add("states", "RUNNING")
		tasks, err := cfClient.TasksByAppByQuery(appID, query)
		if err != nil {
			pzsvc.LogSimpleErr(s, "Cannot poll CF tasks", err)
		}

		if len(tasks) > taskLimit {
			pzsvc.LogInfo(s, "Maximum Tasks reached for App. Will not poll for work until current work has completed.")
			continue
		}

		var pzJobObj struct {
			Data WorkOutData `json:"data"`
		}
		pzJobObj.Data = WorkOutData{SvcData: WorkSvcData{JobID: "", Data: WorkInData{DataInputs: WorkDataInputs{Body: WorkBody{Content: ""}}}}}

		byts, pErr := pzsvc.RequestKnownJSON("POST", "", s.PzAddr+"/service/"+svcID+"/task", s.PzAuth, &pzJobObj)
		if pErr != nil {
			pErr.Log(s, "Dispatcher: error getting new task:" + string(byts))
			time.Sleep(time.Duration(5) * time.Second)
			continue
		}

		inpStr := pzJobObj.Data.SvcData.Data.DataInputs.Body.Content
		jobID := pzJobObj.Data.SvcData.JobID
		if inpStr != "" {
			pzsvc.LogInfo(s, "New Task Grabbed.  JobID: "+jobID)

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
					pzsvc.SendExecResultNoData(s, s.PzAddr, svcID, jobID, pzsvc.PiazzaStatusFail)
					time.Sleep(5 * time.Second)
					continue
				}
			}

			// Form the CLI for the Algorithm Task
			workerCommand := fmt.Sprintf("worker --cliExtra '%s' --userID '%s' --config '%s' --serviceID '%s' --output '%s' --jobID '%s'", jobInputContent.Command, jobInputContent.UserID, configPath, svcID, jobInputContent.OutGeoJs[0], jobID)
			// For each input image, add that image ref as an argument to the CLI.
			// If AWS images, track the total file size to appropriately size the PCF task container.
			var fileSizeTotal int
			for i := range jobInputContent.InExtFiles {
				workerCommand += fmt.Sprintf(" -i '%s:%s'", jobInputContent.InExtNames[i], jobInputContent.InExtFiles[i])
				if strings.Contains(jobInputContent.InExtFiles[i], "amazonaws") {
					fileSize, err := pzsvc.GetS3FileSizeInMegabytes(jobInputContent.InExtFiles[i])
					if err == nil {
						fileSizeTotal += fileSize
					} else {
						err.Log(s, "Tried to get File Size from S3 File " + jobInputContent.InExtFiles[i] + " but encountered an error.")
					}
				}
			}
			diskInMegabyte := 6142
			if fileSizeTotal != 0 {
				// Allocate 2G for the filesystem and executables (with some buffer), then add the image sizes
				diskInMegabyte = 2048 + fileSizeTotal
				pzsvc.LogInfo(s, "Obtained S3 File Sizes for input files; will use Dynamic Disk Space of " + string(diskInMegabyte) + " in Task container.")
			} else {
				pzsvc.LogInfo(s, "Could not get the S3 File Sizes for input files. Will use the default Disk Space when running Task.")
			}

			taskRequest := cfclient.TaskRequest{
				Command:          workerCommand,
				Name:             jobID,
				DropletGUID:      appID,
				MemoryInMegabyte: 3072,
				DiskInMegabyte:   diskInMegabyte,
			}

			pzsvc.LogAudit(s, s.UserID, "Creating CF Task for Job "+jobID+" : "+workerCommand, s.AppName, string(displayByt), pzsvc.INFO)

			// Send Run-Task request to CF
			_, err := cfClient.CreateTask(taskRequest)
			if err != nil {
				pzsvc.LogAudit(s, s.UserID, "Audit failure", s.AppName, "Could not Create PCF Task for Job. Job Failed: "+err.Error(), pzsvc.ERROR)
				pzsvc.SendExecResultNoData(s, s.PzAddr, svcID, jobID, pzsvc.PiazzaStatusFail)
				time.Sleep(5 * time.Second)
				continue
			}

			pzsvc.LogAudit(s, s.UserID, "Task Created for CF Job", s.AppName, string(displayByt), pzsvc.INFO)

			time.Sleep(5 * time.Second)
		} else {
			// This is way too chatty. I don't think it's needed at this point in time. 
			// pzsvc.LogInfo(s, "No Jobs found during Poll; Trying again shortly.")
			time.Sleep(5 * time.Second)
		}
	}
}
