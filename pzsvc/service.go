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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// FindMySvc Searches Pz for a service matching the input information.  If it finds
// one, it returns the service ID.  If it does not, returns an empty string.  Currently
// only able to search on service name.  Will be much more viable as a long-term answer
// if/when it's able to search on both service name and submitting user.
func FindMySvc(svcName, pzAddr, authKey string) (string, error) {

	query := pzAddr + "/service?per_page=1000&keyword=" + url.QueryEscape(svcName)
	
	resp, err := submitGet(query, authKey)
	if err != nil {
		return "", err
	}

	respBuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var respObj SvcWrapper
	err = json.Unmarshal(respBuf, &respObj)
	if err != nil {
		return "", err
	}

	for _, checkServ := range respObj.Data {
		if checkServ.ResMeta.Name == svcName {
			return checkServ.ServiceID, nil
		}
	}

	return "", nil
}

// SubmitSinglePart sends a single-part POST or a PUT call to Pz and returns the
// response.  May work on some other methods, but not yet tested for them.  Includes
// the necessary headers.
func SubmitSinglePart(method, bodyStr, address, authKey string) (*http.Response, error) {

	fileReq, err := http.NewRequest(method, address, bytes.NewBuffer([]byte(bodyStr)))
	if err != nil {
		return nil, err
	}

	// The following header block is necessary for proper Pz function (as of 4 May 2016).
	fileReq.Header.Add("Content-Type", "application/json")
	fileReq.Header.Add("size", "30")
	fileReq.Header.Add("from", "0")
	fileReq.Header.Add("key", "stamp")
	fileReq.Header.Add("order", "true")
	fileReq.Header.Add("Authorization", authKey)

	client := &http.Client{}
	resp, err := client.Do(fileReq)
	if err != nil {
		return nil, err
	}

	return resp, err
}

// ManageRegistration Handles Pz registration for a service.  It checks the current
// service list to see if it has been registered already.  If it has not, it performs
// initial registration.  If it has not, it re-registers.  Best practice is to do this
// every time your service starts up.  For those of you code-reading, the filter is
// still somewhat rudimentary.  It will improve as better tools become available.
func ManageRegistration(svcName, svcDesc, svcURL, pzAddr, svcVers, authKey string, attributes map[string]string) error {
	
	fmt.Println("Finding")
	svcID, err := FindMySvc(svcName, pzAddr, authKey)
	if err != nil {
		return err
	}

	svcClass := ClassType{"UNCLASSIFIED"} // TODO: this will have to be updated at some point.
	metaObj := ResMeta{ svcName, svcDesc, svcClass, svcVers, make(map[string]string) }
	for key, val := range attributes {
		metaObj.Metadata[key] = val
	}
	svcObj := Service{ svcID, svcURL, "", "POST", metaObj }
	svcJSON, err := json.Marshal(svcObj)

	if svcID == "" {
		fmt.Println("Registering")
		_, err = SubmitSinglePart("POST", string(svcJSON), pzAddr+"/service", authKey)
	} else {
		fmt.Println("Updating")
		_, err = SubmitSinglePart("PUT", string(svcJSON), pzAddr+"/service/"+svcID, authKey)
	}
	if err != nil {
		return err
	}

	return nil
}

// ExecIn is a structure designed to contain all of the information necessary
// to call pzsvc-exec.
type ExecIn struct {
	FuncStr		string
	InFiles		[]string
	OutGeoJSON	[]string
	OutGeoTIFF	[]string
	OutTxt		[]string
	AlgoURL		string
	AuthKey		string
}

// CallPzsvcExec is a function designed to simplify calls to pzsvc-exec.
// Fill out the inpObj properly, and it'll go through the contact process,
// returning the OutFiles mapping (as that is generally what people are
// interested in, one way or the other)
func CallPzsvcExec(inpObj *ExecIn) (map[string]string, error){
	type execOut struct {
		InFiles		map[string]string
		OutFiles	map[string]string
		ProgReturn	string
		Errors		[]string
	}
	var formVal url.Values

	formVal = make(map[string][]string)
	formVal.Set("cmd", inpObj.FuncStr)
	formVal.Set("inFiles", sliceToCommaSep(inpObj.InFiles))
	formVal.Set("outGeoJson", sliceToCommaSep(inpObj.OutGeoJSON))
	formVal.Set("outTiffs", sliceToCommaSep(inpObj.OutGeoTIFF))
	formVal.Set("outTxt", sliceToCommaSep(inpObj.OutTxt))
	formVal.Set("authKey", inpObj.AuthKey)
	fmt.Println(inpObj.FuncStr)

	resp, err := http.PostForm(inpObj.AlgoURL, formVal)
	if err != nil {
		return nil, fmt.Errorf(`PostForm: %s`, err.Error())
	}
	
	respBuf := &bytes.Buffer{}
	_, err = respBuf.ReadFrom(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(`ReadFrom: %s`, err.Error())
	}
fmt.Println(respBuf.String())
	var respObj execOut
	err = json.Unmarshal(respBuf.Bytes(), &respObj)
	if err != nil {
		fmt.Printf(`Unmarshalling error.  Json as follows:\n%s`, respBuf.String())
		return nil, fmt.Errorf(`Unmarshalling error: %s.  Json: %s`, err.Error(), respBuf.String())
	}
	
	return respObj.OutFiles, nil
}

// takes a string slice, and turns it into a comma-separated list of strings,
// suitable for JSON.
func sliceToCommaSep(inSlice []string) string {
	sliLen := len(inSlice)
	if (sliLen == 0){
		return ""
	}
	accum := inSlice[0]
	for i := 1; i < sliLen; i++ {
		accum = accum + "," + inSlice[i]
	}
	return accum
}