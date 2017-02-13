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

package pzsvc

import (
	"encoding/json"
	"testing"
)

func TestManageRegistration(t *testing.T) {
	s := Session{PzAddr: "http://testURL.net", PzAuth: "testAuthKey"}
	svcName := "testJobID"
	svcDesc := "testDesc"
	svcURL := "http://testSvcURL.net"
	svcVers := "0.0"

	metaObj := ResMeta{Name: svcName,
		Description: svcDesc,
		ClassType:   ClassType{Classification: "Unclassified"},
		Version:     svcVers,
		Metadata:    map[string]string{"prop1": "1", "prop2": "2", "prop3": "3"}}
	targService := Service{ServiceID: "123", URL: svcURL, Method: "POST", ResMeta: metaObj}
	svcL1 := SvcList{Data: []Service{Service{ServiceID: "123", URL: svcURL, Method: "POST", ResMeta: metaObj}}}
	svcJSON, _ := json.Marshal(svcL1)

	profileStr := `{"type":"user-profile","data":{"userProfile":{"username":"PzTest","distinguishedName":"PzTestLong","createdOn":"aaa"}}}`

	outStrs := []string{`---}`, profileStr, string(svcJSON), `{"Data":[]}`}
	SetMockClient(outStrs, 250)

	err := ManageRegistration(s, targService)
	if err == nil {
		t.Error(`TestManageRegistration: passed on bad json`)
	}
	err = ManageRegistration(s, targService)
	if err != nil {
		t.Error(`TestManageRegistration: failed on full registration.  Error: `, err.Error())
	}
	err = ManageRegistration(s, targService)
	if err != nil {
		t.Error(`TestManageRegistration: failed on empty registration.  Error: `, err.Error())
	}
	SetMockClient([]string{string(svcJSON)}, 500)
	err = ManageRegistration(s, targService)
	if err == nil {
		t.Error(`TestManageRegistration: passed on http error code`)
	}
}
