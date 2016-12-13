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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

// ParseConfig parses the config file on starting up
func ParseConfig(s pzsvc.Session, configObj *ConfigType) ConfigParseOut {

	canReg := CheckConfig(s, configObj)
	canPzFile := configObj.CanUpload || configObj.CanDownlPz

	var authKey string
	if configObj.AuthEnVar != "" && (canReg || canPzFile) {
		authKey = os.Getenv(configObj.AuthEnVar)
		if authKey == "" {
			errStr := "No auth key at AuthEnVar."
			if canReg {
				errStr += "  Registration disabled."
			}
			if canPzFile {
				errStr += "  Client will have to provide authKey for Pz file interactions."
			}
			pzsvc.LogInfo(s, errStr)
			canReg = false
		}
	}

	if configObj.Port <= 0 {
		configObj.Port = 8080
	}
	portStr := ":" + strconv.Itoa(configObj.Port)

	version := GetVersion(s, configObj)

	if canReg {
		pzsvc.LogInfo(s, "About to manage registration.")
		err := pzsvc.ManageRegistration(s,
			configObj.SvcName,
			configObj.Description,
			configObj.URL+"/execute",
			configObj.PzAddr,
			version,
			authKey,
			configObj.Attributes)
		if err != nil {
			pzsvc.LogSimpleErr(s, "pzsvc-exec error in managing registration: ", err)
		} else {
			pzsvc.LogInfo(s, "Registration managed.")
		}
	}

	var procPool = pzsvc.Semaphore(nil)
	if configObj.NumProcs > 0 {
		procPool = make(pzsvc.Semaphore, configObj.NumProcs)
	}

	return ConfigParseOut{authKey, portStr, version, procPool}
}

// Execute does the primary work for pzsvc-exec.  Given a request and various
// blocks of config data, it creates a temporary folder to work in, downloads
// any files indicated in the request (if the configs support it), executes
// the command indicated by the combination of request and configs, uploads
// any files indicated by the request (if the configs support it) and cleans
// up after itself
func Execute(w http.ResponseWriter, r *http.Request, configObj ConfigType, cParseRes ConfigParseOut) (OutStruct, pzsvc.Session) {

	// Makes sure that you only have a certain number of execution tasks firing at once.
	// pzsvc-exec calls can get pretty resource-intensive, and this keeps them from
	// trampling each other into messy deadlock
	cParseRes.ProcPool.Lock()
	defer cParseRes.ProcPool.Unlock()

	var (
		output OutStruct
		inpObj InpStruct
		byts   []byte
		err    error
		pErr   *pzsvc.Error
	)
	output.InFiles = make(map[string]string)
	output.OutFiles = make(map[string]string)

	output.HTTPStatus = http.StatusOK

	s := pzsvc.Session{AppName: configObj.SvcName, SessionID: "FailedOnInit", LogRootDir: "pzsvc-exec"}
	if r.Method != "POST" {
		addOutputError(&output, r.Method+" not supported.  Please us POST.", http.StatusMethodNotAllowed)
		return output, s
	}

	if byts, pErr = pzsvc.ReadBodyJSON(&inpObj, r.Body); err != nil {
		pErr.Log(s, "Could not read request body.  Initial error:")
		addOutputError(&output, "Could not read request body.  Please use JSON format.", http.StatusBadRequest)
		return output, s
	}

	s.SessionID, err = pzsvc.PsuUUID()
	if err != nil {
		s.SessionID = "FailedOnInit"
		pzsvc.LogSimpleErr(s, "psuUUID error: ", err)
		addOutputError(&output, "pzsvc-exec internal error.  Check logs for further information.", http.StatusInternalServerError)
		return output, s
	}
	s.SubFold = s.SessionID // they're the same here, but as far as the pzsvc library is concerned, they're different concepts

	s.PzAddr = inpObj.PzAddr
	s.PzAuth = inpObj.PzAuth

	if inpObj.PzAuth != "" {
		inpObj.PzAuth = "******"
		byts, _ = json.Marshal(inpObj)
	}

	pzsvc.LogInfo(s, `pzsvc-exec call initiated.  Input: `+string(byts))

	cmdParamSlice := splitOrNil(inpObj.Command, " ")
	cmdConfigSlice := splitOrNil(configObj.CliCmd, " ")
	cmdSlice := append(cmdConfigSlice, cmdParamSlice...)

	if s.PzAuth == "" {
		s.PzAuth = cParseRes.AuthKey
	}
	if s.PzAddr == "" {
		s.PzAddr = configObj.PzAddr
	}

	needsPz := (len(inpObj.InPzFiles)+len(inpObj.OutTiffs)+len(inpObj.OutTxts)+len(inpObj.OutGeoJs) != 0)

	if needsPz && s.PzAddr == "" {
		addOutputError(&output, "Cannot complete.  No Piazza address provided for file upload/download.", http.StatusForbidden)
		return output, s
	}

	if needsPz && s.PzAuth == "" {
		addOutputError(&output, "Cannot complete.  Auth Key not available.", http.StatusForbidden)
		return output, s
	}

	if needsPz {
		unlogErr := pzsvc.CheckAuth(s.PzAddr, s.PzAuth)
		if unlogErr != nil {
			addOutputError(&output, "Could not confirm auth.", http.StatusForbidden)
			unlogErr.Log(s, "")
			return output, s
		}
	}

	if !configObj.CanDownlExt && (len(inpObj.InExtFiles) != 0) {
		addOutputError(&output, "Cannot complete.  Configuration does not allow external file download.", http.StatusForbidden)
		return output, s
	}
	if !configObj.CanDownlPz && (len(inpObj.InPzFiles) != 0) {
		addOutputError(&output, "Cannot complete.  Configuration does not allow Piazza file download.", http.StatusForbidden)
		return output, s
	}
	if !configObj.CanUpload && (len(inpObj.OutTiffs)+len(inpObj.OutTxts)+len(inpObj.OutGeoJs) != 0) {
		addOutputError(&output, "Cannot complete.  Configuration does not allow file upload.", http.StatusForbidden)
		return output, s
	}

	err = os.Mkdir("./"+s.SubFold, 0777)
	if err != nil {
		pzsvc.LogSimpleErr(s, "os.Mkdir error: ", err)
		addOutputError(&output, "pzsvc-exec internal error.  Check logs for further information.", http.StatusInternalServerError)
	}
	defer os.RemoveAll("./" + s.SubFold)

	err = os.Chmod("./"+s.SubFold, 0777)
	if err != nil {
		pzsvc.LogSimpleErr(s, "os.Chmod error: ", err)
		addOutputError(&output, "pzsvc-exec internal error.  Check logs for further information.", http.StatusInternalServerError)
	}

	// this is done to enable use of handleFList, which lets us
	// reduce a fair bit of code duplication in plowing through
	// our upload/download lists.  handleFList gets used a fair
	// bit more after the execute call.
	pzDownlFunc := func(dataID, fname, fType string) (string, error) {
		return pzsvc.DownloadByID(s, dataID, fname)
	}
	handleFList(s, inpObj.InPzFiles, inpObj.InPzNames, pzDownlFunc, "unspecified", "Pz download", &output, output.InFiles, w)

	extDownlFunc := func(url, fname, fType string) (string, error) {
		return pzsvc.DownloadByURL(s, url, fname, inpObj.ExtAuth)
	}
	handleFList(s, inpObj.InExtFiles, inpObj.InExtNames, extDownlFunc, "unspecified", "URL download", &output, output.InFiles, w)

	if len(cmdSlice) == 0 {
		addOutputError(&output, "No cmd or CliCmd.  Please provide `cmd` param.", http.StatusBadRequest)
		return output, s
	}

	pzsvc.LogInfo(s, "Executing `"+configObj.CliCmd+" "+inpObj.Command+"`.")

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
	clc.Dir = s.SubFold

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	clc.Stdout = &stdout
	clc.Stderr = &stderr

	err = clc.Run()

	if err != nil {
		pzsvc.LogSimpleErr(s, "clc.Run error: ", err)
		addOutputError(&output, "pzsvc-exec failed on cmd `"+inpObj.Command+"`.  If that was correct, check logs for further details.", http.StatusBadRequest)
	}

	output.ProgStdOut = stdout.String()
	output.ProgStdErr = stderr.String()
	pzsvc.LogInfo(s, `Program stdout: `+stdout.String())
	pzsvc.LogInfo(s, `Program stderr: `+stderr.String())

	attMap := make(map[string]string)
	attMap["algoName"] = configObj.SvcName
	attMap["algoVersion"] = cParseRes.Version
	attMap["algoCmd"] = configObj.CliCmd + " " + inpObj.Command
	attMap["algoProcTime"] = time.Now().UTC().Format("20060102.150405.99999")

	// this is the other spot that handleFlist gets used, and works on the
	// same principles.

	ingFunc := func(fName, dummy, fType string) (string, error) {
		return pzsvc.IngestFile(s, fName, fType, configObj.SvcName, cParseRes.Version, attMap)
	}

	handleFList(s, inpObj.OutTiffs, inpObj.OutTiffs, ingFunc, "raster", "upload", &output, output.OutFiles, w)
	handleFList(s, inpObj.OutTxts, inpObj.OutTxts, ingFunc, "text", "upload", &output, output.OutFiles, w)
	handleFList(s, inpObj.OutGeoJs, inpObj.OutGeoJs, ingFunc, "geojson", "upload", &output, output.OutFiles, w)

	return output, s
}
