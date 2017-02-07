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
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"time"
)

// locString simplifies certain local processes that wish to interact with
// files that may or may not be in a subfolder.
func locString(subFold, fname string) string {
	if subFold == "" {
		return fmt.Sprintf(`./%s`, fname)
	}
	return fmt.Sprintf(`./%s/%s`, subFold, fname)
}

// DownloadByID retrieves a file from Pz using the file access API
func DownloadByID(s Session, dataID, filename string) (string, LoggedError) {
	fName, err := DownloadByURL(s, s.PzAddr+"/file/"+dataID, filename, s.PzAuth, false)
	if err == nil && fName == "" {

		return "", LogSimpleErr(s, `File for DataID `+dataID+` unnamed.  Probable ingest error.`, nil)
	}
	return fName, err
}

// DownloadByURL retrieves a file from the given URL, which may or may not have anything
// to do with Piazza
func DownloadByURL(s Session, url, filename, authKey string, retryOn202 bool) (string, LoggedError) {

	var (
		params map[string]string
		err    error
		pErr   *Error
		resp   *http.Response
		x      int
	)
	LogAudit(s, s.UserID, "file download request for "+filename, url, "", INFO)
	for x = 0; x < 60; x++ {
		resp, pErr = SubmitSinglePart("GET", "", url, authKey)
		if !retryOn202 || resp == nil || !(resp.StatusCode == 202) {
			break
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		LogAudit(s, url, "received 202"+filename, s.UserID, "Will wait minute, then retry.", NOTICE)
		time.Sleep(60 * time.Second)
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	if pErr != nil {
		return "", pErr.Log(s, "Download error: ")
	}
	LogAudit(s, url, "file download response for "+filename, s.UserID, "", INFO)
	if filename == "" {
		contDisp := resp.Header.Get("Content-Disposition")
		_, params, err = mime.ParseMediaType(contDisp)
		if err != nil {
			return "", LogSimpleErr(s, "Download: could not read Content-Disposition header: ", err)
		}
		filename = params["filename"]
		if filename == "" {
			return "", LogSimpleErr(s, `Input file from URL "`+url+`" was not given a name.`, nil)
		}
	}
	LogAudit(s, s.UserID, "local file creation and writing", filename, "", INFO) //file creation/manipulation
	out, err := os.Create(locString(s.SubFold, filename))
	if err != nil {
		return "", LogSimpleErr(s, "Download: could not create file "+filename+": ", err)
	}

	defer out.Close()
	io.Copy(out, resp.Body)

	return filename, nil
}

// Ingest ingests the given bytes to Piazza.
func Ingest(s Session, fName, fType, sourceName, version string,
	ingData []byte,
	props map[string]string) (string, LoggedError) {

	var (
		fileData []byte
		resp     *http.Response
		pErr     *Error
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
			//dType.MimeType = "image/tiff"
			fileData = ingData
		}
	case "geojson":
		{
			dType.MimeType = "application/vnd.geo+json"
			fileData = ingData
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
