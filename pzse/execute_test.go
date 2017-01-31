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

package pzse

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

func TestParseConfiguration(t *testing.T) {
	configs, planOuts := getTestConfigList()
	holdAddrEnv := os.Getenv("addrEnv")
	os.Setenv("addrEnv", "pz-addr")
	holdPortEnv := os.Getenv("portEnv")
	os.Setenv("portEnv", "8081")
	holdAPIKeyEnv := os.Getenv("apiKeyEnv")
	os.Setenv("apiKeyEnv", "pz-auth")
	holdBlankEnv := os.Getenv("blankEnv")
	os.Setenv("blankEnv", "")
	s := pzsvc.Session{}
	for i, config := range configs {
		planOut := planOuts[i]
		var runOut ConfigParseOut
		runOut, s = ParseConfigAndRegister(s, &config)
		if planOut.PortStr != runOut.PortStr {
			t.Error(`TestParseConfiguration: PortStr mismatch on run #` + strconv.Itoa(i) +
				`.  actual: ` + runOut.PortStr + `.  expected: ` + planOut.PortStr + `.`)
		}
		if planOut.Version != runOut.Version {
			t.Error(`TestParseConfiguration: Version mismatch on run #` + strconv.Itoa(i) +
				`.  actual: ` + runOut.Version + `.  expected: ` + planOut.Version + `.`)
		}
	}
	os.Setenv("addrEnv", holdAddrEnv)
	os.Setenv("portEnv", holdPortEnv)
	os.Setenv("apiKeyEnv", holdAPIKeyEnv)
	os.Setenv("blankEnv", holdBlankEnv)
}

func TestExecute(t *testing.T) {
	config := getTestConfigWorkable()
	var s pzsvc.Session
	var parsConfig ConfigParseOut
	parsConfig, s = ParseConfigAndRegister(s, &config)
	s.PzAddr = "testAddr"
	s.PzAuth = "testAuth"
	testResList := []string{"test", "test", `{"data":{"jobId":"testID"}}`, `{"data":{"status":"Success", "Result":{"message":"testStatus", "dataId":"testId"}}}`}
	pzsvc.SetMockClient(testResList, 200)

	r := http.Request{}
	r.Method = "POST"
	inpObj := InpStruct{Command: "-l",
		InExtFiles: []string{"https://avatars0.githubusercontent.com/u/15457149?v=3&s=200"},
		InExtNames: []string{"icon.png"},
		OutTiffs:   []string{"icon.png"},
		PzAuth:     "pzAuth",
		PzAddr:     "pzAddr",
		ExtAuth:    "extAuth",
		UserID:     "user"}

	byts, err := json.Marshal(inpObj)
	if err != nil {
		t.Error(`TestExecute: failed to marshal static object.  errStr: ` + err.Error())
	}

	r.Body = pzsvc.GetMockReadCloser(string(byts))
	outObj, _ := Execute(&r, s, config, parsConfig.ProcPool, parsConfig.Version)
	if outObj.Errors != nil {
		for _, errStr := range outObj.Errors {
			t.Error(`TestExecute: Generated Error: ` + errStr)
		}
	}

	r.Method = "GET"
	outObj, _ = Execute(&r, s, config, parsConfig.ProcPool, parsConfig.Version)
	if outObj.Errors == nil {
		t.Error(`TestExecute: Did not error on GET.`)
	}
	r.Method = "POST"
	/*
		r.Body = pzsvc.GetMockReadCloser("}")
		outObj, _ = Execute(&r, s, config, parsConfig.ProcPool, parsConfig.Version)
		if outObj.Errors == nil {
			objbyt, _ := json.Marshal(outObj)
			t.Error(`TestExecute: Did not error on bad json: ` + string(objbyt))
		}
	*/
	s.PzAddr = ""
	inpObj.PzAddr = ""
	r.Body = pzsvc.GetMockReadCloser(string(byts))
	outObj, _ = Execute(&r, s, config, parsConfig.ProcPool, parsConfig.Version)
	if outObj.Errors == nil {
		t.Error(`TestExecute: Did not error on unfilled need for PzAddr.`)
	}
	s.PzAddr = "testAddr"
	inpObj.PzAddr = "pzAddr"
	byts, _ = json.Marshal(inpObj)

	s.PzAuth = ""
	inpObj.PzAuth = ""
	byts, _ = json.Marshal(inpObj)
	r.Body = pzsvc.GetMockReadCloser(string(byts))
	outObj, _ = Execute(&r, s, config, parsConfig.ProcPool, parsConfig.Version)
	if outObj.Errors == nil {
		t.Error(`TestExecute: Did not error on unfilled need for PzAuth.`)
	}
	s.PzAuth = "testAuth"
	inpObj.PzAuth = "pzAuth"
	byts, _ = json.Marshal(inpObj)

}
