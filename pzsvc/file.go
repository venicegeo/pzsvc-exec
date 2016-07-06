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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
//	"path/filepath"
	"time"
)

func locString(subFold, fname string ) string {
	if subFold == "" {
		return fmt.Sprintf(`./%s`, fname)
	}
	return fmt.Sprintf(`./%s/%s`, subFold, fname)	
}

// submitGet is essentially the standard http.Get() call with
// an additional authKey parameter for Pz access. 
func submitGet(payload, authKey string) (*http.Response, error) {
	fileReq, err := http.NewRequest("GET", payload, nil)
	if err != nil {
		return nil, err
	}

	fileReq.Header.Add("Authorization", authKey)

	client := &http.Client{}
	return client.Do(fileReq)
}

// submitMultipart sends a multi-part POST call, including an optional uploaded file,
// and returns the response.  Primarily intended to support Ingest calls.
func submitMultipart(bodyStr, address, filename, authKey string, fileData []byte) (*http.Response, error) {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	err := writer.WriteField("data", bodyStr)
	if err != nil {
		return nil, err
	}

	if fileData != nil {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			return nil, err
		}
		if (part == nil) {
			return nil, fmt.Errorf("Failure in Form File Creation.")
		}

		_, err = io.Copy(part, bytes.NewReader(fileData))
		if err != nil {
			return nil, err
		}
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	fileReq, err := http.NewRequest("POST", address, body)
	if err != nil {
		return nil, err
	}

	fileReq.Header.Add("Content-Type", writer.FormDataContentType())
	fileReq.Header.Add("Authorization", authKey)

	client := &http.Client{}
	resp, err := client.Do(fileReq)
	if err != nil {
		return nil, err
	}
	return resp, err
}

// DownloadBytes retrieves a file from Pz using the file access API and then
// returns the results as a byte slice
func DownloadBytes(dataID, pzAddr, authKey string) ([]byte, error) {

	resp, err := submitGet(pzAddr + "/file/" + dataID, authKey)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// Download retrieves a file from Pz using the file access API
func Download(dataID, subFold, pzAddr, authKey string) (string, error) {

	resp, err := submitGet(pzAddr + "/file/" + dataID, authKey)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", err
	}

	contDisp := resp.Header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(contDisp)
	filename := params["filename"]
	if filename == "" {
		b := make([]byte, 100)
		resp.Body.Read(b)
		
		return "", fmt.Errorf(`File for DataID %s unnamed.  Probable ingest error.  Initial response characters: %s`, dataID, string(b))
	}
	
	out, err := os.Create(locString(subFold, filename))
	if err != nil {
		return "", err
	}

	defer out.Close()
	io.Copy(out, resp.Body)

	return filename, nil
}

// GetJobResponse will repeatedly poll the job status on the given job Id
// until job completion, then acquires and returns the DataResult.  
func GetJobResponse(jobID, pzAddr, authKey string) (*DataResult, error) {

	if jobID == "" {
		return nil, fmt.Errorf(`JobID not provided after ingest.  Cannot acquire dataID.`)
	}

	for i := 0; i < 180; i++ { // will wait up to 3 minutes
		resp, err := submitGet(pzAddr + "/job/" + jobID, authKey)
		if resp == nil {
			return nil, fmt.Errorf("getDataID: no response")
		}
		if err != nil {
			resp.Body.Close()
			return nil, err
		}

		respBuf := &bytes.Buffer{}

		_, err = respBuf.ReadFrom(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var respObj JobResp
		err = json.Unmarshal(respBuf.Bytes(), &respObj)
		if err != nil {
			return nil, err
		}

		if	respObj.Status == "Submitted" ||
			respObj.Status == "Running" ||
			respObj.Status == "Pending" ||
			( respObj.Status == "Success" && respObj.Result.DataID == "" ) {
			time.Sleep(1000 * time.Millisecond)
		} else {
			if respObj.Status == "Success" {
				return &respObj.Result, nil
			}
			if respObj.Status == "Fail" {
				return nil, errors.New("Piazza failure when acquiring DataId.  Response json: " + respBuf.String())
			}
			if respObj.Status == "Error" {
				return nil, errors.New("Piazza error when acquiring DataId.  Response json: " + respBuf.String())
			}
			return nil, errors.New("Unknown status when acquiring DataId.  Response json: " + respBuf.String())
		}
	}

	return nil, fmt.Errorf("Never completed.  JobId: %s", jobID)
}

// Ingest ingests the given bytes to Pz.  
func Ingest(fName, fType, pzAddr, sourceName, version, authKey string,
			ingData []byte,
			props map[string]string) (string, error) {

	var fileData []byte
	var resp *http.Response

	desc := fmt.Sprintf("%s uploaded by %s.", fType, sourceName)
	rMeta := ResMeta{fName, desc, ClassType{"UNCLASSIFIED"}, version, make(map[string]string)} //TODO: implement classification
	for key, val := range props {
		rMeta.Metadata[key] = val
	}

	dType := DataType{"", fType, "", nil}

	switch fType {
		case "raster" : {
			//dType.MimeType = "image/tiff"
			fileData = ingData
		}
		case "geojson" : {
			dType.MimeType = "application/vnd.geo+json"
			fileData = ingData
		}
		case "text" : {
			dType.MimeType = "application/text"
			dType.Content = string(ingData)
			fileData = nil
		}
	}

	dRes := DataResource{dType, rMeta, "", nil}
	jType := IngJobType{"ingest", true, dRes}
	bbuff, err := json.Marshal(jType)
	if err != nil {
		return "", err
	}

	if (fileData != nil) {
		resp, err = submitMultipart(string(bbuff), (pzAddr + "/data/file"), fName, authKey, fileData)
	} else {
		resp, err = SubmitSinglePart("POST", string(bbuff), (pzAddr + "/data"), authKey)
	}
	if err != nil {
		return "", err
	}

	jobID, err := getJobID(resp)
	if err != nil {
		return "", err
	}

	result, err := GetJobResponse(jobID, pzAddr, authKey)
	if err != nil {
		return "", err
	}
	
	return result.DataID, err
}

// IngestFile ingests the given file
func IngestFile(fName, subFold, fType, pzAddr, sourceName, version, authKey string,
				props map[string]string) (string, error) {

//	absPath, err := filepath.Abs(locString(subFold, fName))
	path := locString(subFold, fName)
fmt.Println(path)
/*	if err != nil {
		return "", err
	}*/
	fData, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(fData) == 0 {
		return "", fmt.Errorf(`pzsvc.IngestFile: File "%s" read as empty`, fName)
	}
	return Ingest(fName, fType, pzAddr, sourceName, version, authKey, fData, props)
}

// GetFileMeta retrieves the metadata for a given dataID in the S3 bucket
func GetFileMeta(dataID, pzAddr, authKey string) (*DataResource, error) {

	call := fmt.Sprintf(`%s/data/%s`, pzAddr, dataID)
	resp, err := submitGet(call, authKey)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	respBuf := &bytes.Buffer{}
	_, err = respBuf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	var respObj IngJobType
	err = json.Unmarshal(respBuf.Bytes(), &respObj)
	if err != nil {
		return nil, err
	}

	return &respObj.Data, nil
}

// UpdateFileMeta updates the metadata for a given dataID in the S3 bucket
func UpdateFileMeta(dataID, pzAddr, authKey string, newMeta map[string]string ) error {
	
	var meta struct { Metadata map[string]string `json:"metadata"` }
	meta.Metadata = newMeta
	jbuff, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	
	_, err = SubmitSinglePart("POST", string(jbuff), fmt.Sprintf(`%s/data/%s`, pzAddr, dataID), authKey)
	return err
}

func DeployToGeoServer(dataID, pzAddr, authKey string) (string, error) {
	outJSON := fmt.Sprintf(`{"dataId":"%s","deploymentType":"geoserver","type":"access"}`, dataID)
	resp, err := SubmitSinglePart("POST", outJSON, fmt.Sprintf(`%s/deployment`, pzAddr), authKey)
	if err != nil {
		return "", err
	}
		
	jobID, err := getJobID(resp)
	if err != nil {
		return "", err
	}

	result, err := GetJobResponse(jobID, pzAddr, authKey)
	if err != nil {
		return "", err
	}

	return result.Deployment.DeplID, nil
}



func getJobID (resp *http.Response) (string, error) {

	respBuf := &bytes.Buffer{}
	_, err := respBuf.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}
// need to decide exactly how we're going to treat these errors
	var respObj JobResp
	err = json.Unmarshal(respBuf.Bytes(), &respObj)
	if err != nil {
		fmt.Println("error:", err)
	}

	return respObj.JobID, nil
}
