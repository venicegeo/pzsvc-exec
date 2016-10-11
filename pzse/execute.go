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
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/venicegeo/pzsvc-lib"
)

// ParseConfig parses the config file on starting up
func ParseConfig(configObj *ConfigType) ConfigParseOut {

	canReg, canFile, hasAuth := CheckConfig(configObj)

	var authKey string
	if hasAuth {
		authKey = os.Getenv(configObj.AuthEnVar)
		if authKey == "" {
			fmt.Println("Error: no auth key at AuthEnVar.  Registration disabled, and client will have to provide authKey.")
			hasAuth = false
			canReg = false
		}
	}

	if configObj.Port <= 0 {
		configObj.Port = 8080
	}
	portStr := ":" + strconv.Itoa(configObj.Port)

	version := GetVersion(configObj)

	if canReg {
		fmt.Println("About to manage registration.")
		err := pzsvc.ManageRegistration(configObj.SvcName,
			configObj.Description,
			configObj.URL+"/execute",
			configObj.PzAddr,
			version,
			authKey,
			configObj.Attributes)
		if err != nil {
			fmt.Println("pzsvc-exec error in managing registration: ", err.Error())
		}
		fmt.Println("Registration managed.")
	}

	var procPool = pzsvc.Semaphore(nil)
	if configObj.NumProcs > 0 {
		procPool = make(pzsvc.Semaphore, configObj.NumProcs)
	}

	return ConfigParseOut{authKey, portStr, version, canFile, procPool}
}

// Execute does the primary work for pzsvc-exec.  Given a request and various
// blocks of config data, it creates a temporary folder to work in, downloads
// any files indicated in the request (if the configs support it), executes
// the command indicated by the combination of request and configs, uploads
// any files indicated by the request (if the configs support it) and cleans
// up after itself
func Execute(w http.ResponseWriter, r *http.Request, configObj ConfigType, pzAuth, version string, canFile bool, procPool pzsvc.Semaphore) OutStruct {

	// Makes sure that you only have a certain number of execution tasks firing at once.
	// pzsvc-exec calls can get pretty resource-intensive, and this keeps them from
	// trampling each other into messy deadlock
	procPool.Lock()
	defer procPool.Unlock()

	var output OutStruct
	output.InFiles = make(map[string]string)
	output.OutFiles = make(map[string]string)
	output.HTTPStatus = http.StatusOK

	if r.Method != "POST" {
		handleError(&output, "", fmt.Errorf(r.Method+" not supported.  Please us POST."), w, http.StatusMethodNotAllowed)
		return output
	}

	cmdParam := r.FormValue("cmd")
	cmdParamSlice := splitOrNil(cmdParam, " ")
	cmdConfigSlice := splitOrNil(configObj.CliCmd, " ")
	cmdSlice := append(cmdConfigSlice, cmdParamSlice...)

	inPzFileIDs := splitOrNil(r.FormValue("inFiles"), ",")
	inExtFileURLs := splitOrNil(r.FormValue("inFileURLs"), ",")
	inPzFileNames := splitOrNil(r.FormValue("inPzFileNames"), ",")
	inExtFileNames := splitOrNil(r.FormValue("inExtFileNames"), ",")
	outTiffs := splitOrNil(r.FormValue("outTiffs"), ",")
	outTxts := splitOrNil(r.FormValue("outTxts"), ",")
	outGeoJs := splitOrNil(r.FormValue("outGeoJson"), ",")

	urlAuth := r.FormValue("inUrlAuthKey")
	if r.FormValue("authKey") != "" {
		pzAuth = r.FormValue("authKey")
	}

	if !canFile && (len(inPzFileIDs)+len(outTiffs)+len(outTxts)+len(outGeoJs) != 0) {
		handleError(&output, "", fmt.Errorf("Cannot complete.  File up/download not enabled in config file."), w, http.StatusForbidden)
		return output
	}

	if pzAuth == "" && (len(inPzFileIDs)+len(outTiffs)+len(outTxts)+len(outGeoJs) != 0) {
		handleError(&output, "", fmt.Errorf("Cannot complete.  Auth Key not available."), w, http.StatusForbidden)
		return output
	}

	runID, err := pzsvc.PsuUUID()
	handleError(&output, "psuUUID error: ", err, w, http.StatusInternalServerError)

	err = os.Mkdir("./"+runID, 0777)
	handleError(&output, "os.Mkdir error: ", err, w, http.StatusInternalServerError)
	defer os.RemoveAll("./" + runID)

	err = os.Chmod("./"+runID, 0777)
	handleError(&output, "os.Chmod error: ", err, w, http.StatusInternalServerError)

	// this is done to enable use of handleFList, which lets us
	// reduce a fair bit of code duplication in plowing through
	// our upload/download lists.  handleFList gets used a fair
	// bit more after the execute call.
	pzDownlFunc := func(dataID, fname, fType string) (string, error) {
		return pzsvc.DownloadByID(dataID, fname, runID, configObj.PzAddr, pzAuth)
	}
	handleFList(inPzFileIDs, inPzFileNames, pzDownlFunc, "", &output, output.InFiles, w)

	extDownlFunc := func(url, fname, fType string) (string, error) {
		return pzsvc.DownloadByURL(url, fname, runID, urlAuth)
	}
	handleFList(inExtFileURLs, inExtFileNames, extDownlFunc, "", &output, output.InFiles, w)

	if len(cmdSlice) == 0 {
		handleError(&output, "", errors.New(`No cmd or CliCmd.  Please provide "cmd" param.`), w, http.StatusBadRequest)
		return output
	}

	fmt.Println(`Executing "` + configObj.CliCmd + ` ` + cmdParam + `".`)

	// we're calling this from inside a temporary subfolder.  If the
	// program called exists inside the initial pzsvc-exec folder, that's
	// probably where it's called from, and we need to acccess it directly.
	_, err = os.Stat(fmt.Sprintf("./%s", cmdSlice[0]))
	if err == nil || !(os.IsNotExist(err)) {
		// ie, if there's a file in the start folder named the same thing
		// as the base command
		cmdSlice[0] = ("../" + cmdSlice[0])
	}

	clc := exec.Command(cmdSlice[0], cmdSlice[1:]...)
	clc.Dir = runID

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	clc.Stdout = &stdout
	clc.Stderr = &stderr

	err = clc.Run()
	handleError(&output, "clc.Run error: ", err, w, http.StatusBadRequest)

	output.ProgStdOut = stdout.String()
	output.ProgStdErr = stderr.String()

	fmt.Println(`Program stdout: ` + output.ProgStdOut)
	fmt.Println(`Program stderr: ` + output.ProgStdErr)

	attMap := make(map[string]string)
	attMap["algoName"] = configObj.SvcName
	attMap["algoVersion"] = version
	attMap["algoCmd"] = configObj.CliCmd + " " + cmdParam
	attMap["algoProcTime"] = time.Now().UTC().Format("20060102.150405.99999")

	// this is the other spot that handleFlist gets used, and works on the
	// same principles.

	ingFunc := func(fName, dummy, fType string) (string, error) {
		return pzsvc.IngestFile(fName, runID, fType, configObj.PzAddr, configObj.SvcName, version, pzAuth, attMap)
	}

	handleFList(outTiffs, nil, ingFunc, "raster", &output, output.OutFiles, w)
	handleFList(outTxts, nil, ingFunc, "text", &output, output.OutFiles, w)
	handleFList(outGeoJs, nil, ingFunc, "geojson", &output, output.OutFiles, w)

	return output
}
