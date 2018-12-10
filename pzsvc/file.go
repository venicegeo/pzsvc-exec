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
	"fmt"
	"io/ioutil"
	"net/http"
)

// locString simplifies certain local processes that wish to interact with
// files that may or may not be in a subfolder.
func locString(subFold, fname string) string {
	if subFold == "" {
		return fmt.Sprintf(`./%s`, fname)
	}
	return fmt.Sprintf(`./%s/%s`, subFold, fname)
}

// Ingest ingests the given bytes to Piazza.
func Ingest(s Session, fName, fType, sourceName, version string,
	ingData []byte,
	props map[string]string) (string, LoggedError) {

	var (
		fileData []byte
		resp     *http.Response
		pErr     *PzCustomError
		targAddr string
	)

	desc := fmt.Sprintf("%s uploaded by %s.", fType, sourceName)
	rMeta := ResMeta{
		Name:        fName,
		Format:      fType,
		ClassType:   ClassType{"UNCLASSIFIED"},
		Version:     version,
		Description: desc,
		Metadata:    make(map[string]string)}

	for key, val := range props {
		rMeta.Metadata[key] = val
	}

	dType := DataType{Type: fType}

	switch fType {
	case "raster":
		{
			fileData = ingData
		}
	case "geojson":
		{
			dType.MimeType = "application/vnd.geo+json"
			dType.GeoJContent = string(ingData)
			fileData = nil
		}
	case "text":
		{
			dType.MimeType = "application/text"
			dType.Content = string(ingData)
			fileData = nil
		}
	}

	dRes := DataDesc{"", dType, rMeta, nil}
	jType := IngestReq{dRes, true, "ingest"}
	bbuff, err := json.Marshal(jType)
	if err != nil {
		return "", LogSimpleErr(s, "Internal Error.  Failure when marshalling IngestReq: ", err)
	}

	if fileData != nil {
		targAddr = s.PzAddr + "/data/file"
		LogInfo(s, "beginning file upload")
		LogAudit(s, s.UserID, "file upload http request", targAddr, string(bbuff), INFO)
		resp, pErr = SubmitMultipart(string(bbuff), targAddr, fName, s.PzAuth, fileData)
	} else {
		targAddr = s.PzAddr + "/data"
		LogAudit(s, s.UserID, "file upload http request", targAddr, string(bbuff), INFO)
		resp, pErr = SubmitSinglePart("POST", string(bbuff), targAddr, s.PzAuth)
	}
	if pErr != nil {
		return "", pErr.Log(s, "Failure submitting Ingest request")
	}
	LogAuditResponse(s, targAddr, "file upload http response", s.UserID, resp, INFO)

	jobID, pErr := GetJobID(resp)
	if pErr != nil {
		return "", pErr.Log(s, "Failure pulling Job ID for Ingest request")
	}

	result, pErr := GetJobResponse(s, jobID)
	if pErr != nil {
		return "", pErr.Log(s, "Failure getting job result for Ingest call")
	}
	return result.DataID, nil
}

// IngestFile ingests the given file to Piazza
func IngestFile(s Session, fName, fType, sourceName, version string,
	props map[string]string) (string, LoggedError) {

	path := locString(s.SubFold, fName)

	LogAudit(s, s.UserID, "read file for ingest", path, "", INFO)
	fData, err := ioutil.ReadFile(path)
	if err != nil {
		return "", LogSimpleErr(s, `Error reading file `+fName+` for Ingest: `, err)
	}
	if len(fData) == 0 {
		return "", LogSimpleErr(s, `File "`+fName+`" read as empty.`, nil)
	}
	return Ingest(s, fName, fType, sourceName, version, fData, props)
}
