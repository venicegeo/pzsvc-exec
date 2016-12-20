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

	/*
	   actual pertinent data:
	   - configObj.PzAddr (required)
	   - configObj.Port (default to 8080)
	   - configObj.AuthEnVar (required)
	   - numProcs (required?)(default to 1?)
	   - Stuff out of config as necessary to uniquely identify target service
	*/

	if configObj.PzAddr == "" {
		pzsvc.LogSimpleErr(s, "Config: Cannot work tasks without valid PzAddr.", nil)
		return
	}
	if configObj.AuthEnVar == "" {
		pzsvc.LogSimpleErr(s, "Config: Cannot work tasks without valid AuthEnVar.", nil)
		return
	}
	if configObj.SvcName == "" {
		pzsvc.LogSimpleErr(s, "Config: Cannot work tasks without service name.", nil)
		return
	}
	authKey := os.Getenv(configObj.AuthEnVar)
	if authKey == "" {
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

	svcID, err := getSvcID(configObj.PzAddr, authKey, configObj.SvcName)
	if err != nil {
		pzsvc.LogSimpleErr(s, "Was not able to acquire Pz Service ID: ", err)
		return
	}

	for i := 0; i < configObj.NumProcs; i++ {
		workerThread(s, configObj, authKey, svcID)
	}
	select {} //blocks forever

	//pzsvc.LogAudit(s, s.AppName, "shutdown", s.AppName)

}

func workerThread(s pzsvc.Session, configObj pzse.ConfigType, pzAuth, svcID string) {

	workAddr := fmt.Sprintf("localhost:%d", configObj.Port)
	var outBody *pzse.InpStruct

	for {
		byts := getPzJob(configObj.PzAddr, pzAuth, svcID)
		if byts != nil {
			// strip out Pz wrapper stuff, save everything useful.
			if configObj.JwtSecAuthURL != "" {
				// jwtBody = content
				// call JwtSecAuthURL.  send jwtBody.  get response
				// outBody = response (more or less)
			} else {
				// outBody = content
			}
			pzse.CallPzsvcExec(s, outBody, workAddr)
			// Add appropriate pz wrapper to response.
			// call Pz: configObj.PzAddr, svcId, pzAuth - somewhat different endpoint (GET vs POST?).  send results
		} else {
			time.Sleep(60) // in seconds
		}
	}

}

func getSvcID(pzAddr, pzAuth, svcName string) (string, error) {

	// call in to Pz.  Get service ID.
	// - requires API for service layout of new service
	//
	return "", nil
}

// this probably gets turned into a pzsvc function at some point.
func getPzJob(pzAddr, pzAuth, svcID string) []byte {
	var byts []byte
	//call Pz, get response as byts
	return byts
}
