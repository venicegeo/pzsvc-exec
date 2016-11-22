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
func DownloadByID(dataID, filename, subFold, pzAddr, authKey string) (string, LoggedError) {
	fName, err := DownloadByURL(pzAddr+"/file/"+dataID, filename, subFold, authKey)
	if err == nil && fName == "" {

		return "", LogSimpleErr(`File for DataID `+dataID+` unnamed.  Probable ingest error.`, nil)
	}
	return fName, err
}

// DownloadByURL retrieves a file from the given URL
func DownloadByURL(url, filename, subFold, authKey string) (string, LoggedError) {

	var (
		params map[string]string
		err    error
		pErr   *Error
	)
	resp, pErr := SubmitSinglePart("GET", "", url, authKey)
	if resp != nil {
		defer resp.Body.Close()
	}
	if pErr != nil {
		return "", pErr.Log("Download error: ")
	}
	if filename == "" {
		contDisp := resp.Header.Get("Content-Disposition")
		_, params, err = mime.ParseMediaType(contDisp)
		if err != nil {
			return "", LogSimpleErr("Download: could not read Content-Disposition header: ", err)
		}
		filename = params["filename"]
		if filename == "" {
			return "", LogSimpleErr(`Input file from URL "`+url+`" was not given a name.`, nil)
		}
	}
	out, err := os.Create(locString(subFold, filename))
	if err != nil {
		return "", LogSimpleErr("Download: could not create file "+filename+": ", err)
	}

	defer out.Close()
	io.Copy(out, resp.Body)

	return filename, nil
}

// Ingest ingests the given bytes to Piazza.
func Ingest(fName, fType, pzAddr, sourceName, version, authKey string,
	ingData []byte,
	props map[string]string) (string, LoggedError) {

	var (
		fileData []byte
		resp     *http.Response
		pErr     *Error
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
		return "", LogSimpleErr("Internal Error.  Failure when marshalling IngestReq: ", err)
	}

	if fileData != nil {
		resp, pErr = SubmitMultipart(string(bbuff), (pzAddr + "/data/file"), fName, authKey, fileData)
	} else {
		resp, pErr = SubmitSinglePart("POST", string(bbuff), (pzAddr + "/data"), authKey)
	}
	if pErr != nil {
		return "", pErr.Log("Failure submitting Ingest request")
	}

	jobID, pErr := GetJobID(resp)
	if pErr != nil {
		return "", pErr.Log("Failure pulling Job ID for Ingest request")
	}

	result, pErr := GetJobResponse(jobID, pzAddr, authKey)
	if pErr != nil {
		return "", pErr.Log("Failure getting job result for Ingest call")
	}
	return result.DataID, nil
}

// IngestFile ingests the given file to Piazza
func IngestFile(fName, subFold, fType, pzAddr, sourceName, version, authKey string,
	props map[string]string) (string, LoggedError) {

	path := locString(subFold, fName)

	fData, err := ioutil.ReadFile(path)
	if err != nil {
		return "", LogSimpleErr(`Error reading file `+fName+` for Ingest: `, err)
	}
	if len(fData) == 0 {
		return "", LogSimpleErr(`File "`+fName+`" read as empty.`, nil)
	}
	return Ingest(fName, fType, pzAddr, sourceName, version, authKey, fData, props)
}
