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
	"io/ioutil"
	"os"

	"github.com/venicegeo/pzsvc-exec/dispatcher/cfwrapper"
	"github.com/venicegeo/pzsvc-exec/dispatcher/poll"
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
	var configObj pzsvc.Config
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

	// Check for the Service ID. If it exists, then grab the ID. If it doesn't exist, then Register it.
	svcID, err := newPzSvcDiscoverer().discoverSvcID(&s, &configObj)
	if err != nil {
		pzsvc.LogSimpleErr(s, "Dispatcher could not find Piazza Service ID. Error: ", err)
		return
	}
	pzsvc.LogInfo(s, "Found target service.  ServiceID: "+svcID+".")

	// Initialize the CF Client
	clientConfig := &cfwrapper.Config{
		ApiAddress: os.Getenv("CF_API"),
		Username:   os.Getenv("CF_USER"),
		Password:   os.Getenv("CF_PASS"),
	}
	clientFactory, err := cfwrapper.NewFactory(&s, clientConfig)

	if err != nil {
		pzsvc.LogSimpleErr(s, "Error in initializing CF Client factory: ", err)
		return
	}

	pzsvc.LogInfo(s, "Cloud Foundry Client initialized. Beginning Polling.")

	pollLoop, err := poll.NewLoop(&s, configObj, svcID, configPath, clientFactory)
	if err != nil {
		pzsvc.LogSimpleErr(s, "Error in initializing dispatch polling loop: ", err)
		return
	}

	for err = range pollLoop.Start() {
		pzsvc.LogSimpleErr(s, "Polling loop encountered an error on this iteration:: ", err)
	}
}
