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
	"io/ioutil"
	"os"
	"testing"
)

func TestDownloadByID(t *testing.T) {
	outStrs := []string{`{"test":"blah"}`, `{"test":"blah"}`, `{"test":"blah"}`}
	SetMockClient(outStrs, 250)
	url := "http://testURL.net"
	authKey := "testAuthKey"
	dataID := "1234ID"
	fileName := "tempTestFile.tmp"
	subFold := "folderName"
	s := Session{SubFold: subFold, PzAddr: url, PzAuth: authKey, LogRootDir: "Test"}

	os.Mkdir(subFold, 0777)
	_, err := DownloadByID(s, dataID, fileName)
	if err != nil {
		t.Error(`TestDownloadByID: failed on subfolder-yes call: ` + err.Error())
	}
	os.RemoveAll(subFold)

	s.SubFold = ""
	_, err = DownloadByID(s, dataID, fileName)
	if err != nil {
		t.Error(`TestDownloadByID: failed on subfolder-no call: ` + err.Error())
	}
	os.Remove(locString("", fileName))

	_, err = DownloadByID(s, dataID, "")
	if err == nil {
		t.Error(`TestDownloadByID: passed a filename-no call: ` + err.Error())
	}
}

func TestIngestFile(t *testing.T) {
	outStrs := []string{
		`{"Data":{"JobID":"testID1"}}`,
		`{"Data":{"Status":"Success", "Result":{"Message":"testStatus1"}}}`,
		`{"Data":{"JobID":"testID2"}}`,
		`{"Data":{"Status":"Success", "Result":{"Message":"testStatus2"}}}`,
		`{"Data":{"JobID":"testID3"}}`,
		`{"Data":{"Status":"Success", "Result":{"Message":"testStatus3"}}}`}
	SetMockClient(outStrs, 250)
	url := "http://testURL.net"
	authKey := "testAuthKey"
	fileName := "tempTestFile.tmp"
	subFold := "folderName"
	s := Session{SubFold: subFold, PzAddr: url, PzAuth: authKey, LogAudit: true}

	os.Mkdir(subFold, 0777)
	err := ioutil.WriteFile("./"+subFold+"/"+fileName, []byte(fileName), 0666)
	if err != nil {
		t.Error(`TestIngestFile: error on file creation: ` + err.Error())
	}
	_, err = IngestFile(s, fileName, "text", "tester", "0.0", map[string]string{"prop1": "1", "prop2": "2"})
	if err != nil {
		t.Error(`TestIngestFile: error on text ingest: ` + err.Error())
	}
	_, err = IngestFile(s, fileName, "geojson", "tester", "0.0", map[string]string{"prop1": "1", "prop2": "2"})
	if err != nil {
		t.Error(`TestIngestFile: error on geojson ingest: ` + err.Error())
	}
	_, err = IngestFile(s, fileName, "raster", "tester", "0.0", map[string]string{"prop1": "1", "prop2": "2"})
	if err != nil {
		t.Error(`TestIngestFile: error on raster ingest: ` + err.Error())
	}
	os.RemoveAll(subFold)
}
