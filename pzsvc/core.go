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
	"mime/multipart"
	"net/http"
	"time"
)

// RequestKnownJSON submits an http request where the response is assumed to be JSON
// for which the format is known.  Given an object of the appropriate format for
// said response JSON, an address to call and an authKey to send, it will submit
// the get request, demarshal the result into the given object, and return. It
// returns the response buffer, in case it is needed for debugging purposes.
func RequestKnownJSON(method, bodyStr, address, authKey string, outpObj interface{}) (*bytes.Buffer, error) {
// TODO: could very easily refactor this to cover PUT/POST/DELETE as well

	resp, err := SubmitSinglePart(method, bodyStr, address, authKey)
	if resp == nil {
		return nil, fmt.Errorf("GetPzObj: no response")
	}
	if err != nil {
		resp.Body.Close()
		return nil, err
	}

	respBuf := &bytes.Buffer{}

	_, err = respBuf.ReadFrom(resp.Body)
	resp.Body.Close()
	if err != nil {
		return respBuf, err
	}

	err = json.Unmarshal(respBuf.Bytes(), outpObj)
	return respBuf, err
}


// SubmitMultipart sends a multi-part POST call, including an optional uploaded file,
// and returns the response.  Primarily intended to support Ingest calls.
func SubmitMultipart(bodyStr, address, filename, authKey string, fileData []byte) (*http.Response, error) {

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
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("Failed to POST multipart to " + address + " Status: " + resp.Status)
	}
	return resp, err
}

// SubmitSinglePart sends a single-part GET/POST/PUT/DELETE call to Pz and returns the
// Includes the necessary headers.
func SubmitSinglePart(method, bodyStr, url, authKey string) (*http.Response, error) {

	var fileReq *http.Request
	var err error

	if bodyStr != "" {
		fileReq, err = http.NewRequest(method, url, bytes.NewBuffer([]byte(bodyStr)))
		if err != nil {
			return nil, err
		}
		fileReq.Header.Add("Content-Type", "application/json")
	} else {
		fileReq, err = http.NewRequest(method, url, nil)
	}
	
	fileReq.Header.Add("Authorization", authKey)

	client := &http.Client{}
	resp, err := client.Do(fileReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("Failed in " + method + " call to " + url + ".  Status : " + resp.Status)
	}

	return resp, err
}

// GetJobResponse will repeatedly poll the job status on the given job Id
// until job completion, then acquires and returns the DataResult.  
func GetJobResponse(jobID, pzAddr, authKey string) (*DataResult, error) {

	if jobID == "" {
		return nil, fmt.Errorf(`JobID not provided after ingest.  Cannot acquire dataID.`)
	}

	for i := 0; i < 180; i++ { // will wait up to 3 minutes

		var respObj JobResp
		respBuf, err := RequestKnownJSON("GET", "", pzAddr + "/job/" + jobID, authKey, &respObj)
		if err != nil {
			return nil, err
		}

		if	respObj.Status == "Submitted" ||
			respObj.Status == "Running" ||
			respObj.Status == "Pending" ||
			( respObj.Status == "Error" && respObj.Message == "Job Not Found." ) ||
			( respObj.Status == "Success" && respObj.Result == nil ) {
			time.Sleep(time.Second)
		} else {
			if respObj.Status == "Success" {
				return respObj.Result, nil
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

// GetJobID is a simple function to extract the job ID from
// the standard response to job-creating Pz calls
func GetJobID(resp *http.Response) (string, error) {

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
