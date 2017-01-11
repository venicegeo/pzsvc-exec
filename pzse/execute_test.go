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
	configs, planOuts, authEnv := getTestConfigList()
	holdEnv := os.Getenv(authEnv)
	os.Setenv(authEnv, "pzsvc-exec")
	s := pzsvc.Session{}
	for i, config := range configs {
		planOut := planOuts[i]
		runOut := ParseConfigAndRegister(s, &config)
		if planOut.AuthKey != runOut.AuthKey {
			t.Error(`TestParseConfiguration: AuthKey mismatch on run #` + strconv.Itoa(i) +
				`.  actual: ` + runOut.AuthKey + `.  expected: ` + planOut.AuthKey + `.`)
		}
		if planOut.PortStr != runOut.PortStr {
			t.Error(`TestParseConfiguration: PortStr mismatch on run #` + strconv.Itoa(i) +
				`.  actual: ` + runOut.PortStr + `.  expected: ` + planOut.PortStr + `.`)
		}
		if planOut.Version != runOut.Version {
			t.Error(`TestParseConfiguration: Version mismatch on run #` + strconv.Itoa(i) +
				`.  actual: ` + runOut.Version + `.  expected: ` + planOut.Version + `.`)
		}
	}
	os.Setenv(authEnv, holdEnv)
}

func TestExecute(t *testing.T) {
	config := getTestConfigWorkable()
	s := pzsvc.Session{}
	parsConfig := ParseConfigAndRegister(s, &config)
	testResList := []string{"test", "test", `{"data":{"jobId":"testID"}}`, `{"data":{"status":"Success", "Result":{"message":"testStatus", "dataId":"testId"}}}`}
	pzsvc.SetMockClient(testResList, 200)

	r := http.Request{}
	r.Method = "POST"
	inpObj := InpStruct{Command: "-l",
		InExtFiles: []string{"https://avatars0.githubusercontent.com/u/15457149?v=3&s=200"},
		InExtNames: []string{"icon.png"},
		OutTiffs:   []string{"icon.png"},
		PzAuth:     "aaa"}

	byts, err := json.Marshal(inpObj)
	if err != nil {
		t.Error(`TestExecute: failed to marshal static object.  errStr: ` + err.Error())
	}

	r.Body = pzsvc.GetMockReadCloser(string(byts))
	outObj, _ := Execute(&r, config, parsConfig)

	if outObj.Errors != nil {
		for _, errStr := range outObj.Errors {
			t.Error(`TestExecute: Generated Error: ` + errStr)
		}
	}
}
