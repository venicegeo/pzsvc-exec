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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

// ParseConfigAndRegister parses the config file on starting up, manages
// registration for it on the given Pz instance if registration management
// is called for, and returns a few useful derived values
func ParseConfigAndRegister(s pzsvc.Session, configObj *ConfigType) (ConfigParseOut, pzsvc.Session) {

	canReg := CheckConfig(s, configObj)
	canPzFile := configObj.CanUpload || configObj.CanDownlPz

	s.PzAddr = configObj.PzAddr
	if configObj.PzAddrEnVar != "" {
		newAddr := os.Getenv(configObj.PzAddrEnVar)
		if newAddr != "" {
			s.PzAddr = newAddr
			pzsvc.LogInfo(s, `Config: PzAddr updated to `+configObj.PzAddr+` based on PzAddrEnVar.`)
		} else if s.PzAddr != "" {
			pzsvc.LogInfo(s, `Config: PzAddrEnVar specified in config, but no such env var exists.  Reverting to specified PzAddr.`)
		} else {
			logStr := `Config: PzAddrEnVar specified in config, but no such env var exists, and PzAddr not specified.`
			if canReg {
				logStr += `  Autoregistration disabled.`
				canReg = false
			}
			if canPzFile {
				logStr += `  Client will have to provide Piazza Address for uploads and Piazza downloads.`
			}
			pzsvc.LogInfo(s, logStr)
		}
	}

	if configObj.APIKeyEnVar != "" && (canReg || canPzFile) {
		apiKey := os.Getenv(configObj.APIKeyEnVar)
		if apiKey == "" {
			errStr := "No api key at APIKeyEnVar."
			if canReg {
				errStr += "  Registration disabled."
			}
			if canPzFile {
				errStr += "  Client will have to provide authKey for Pz file interactions."
			}
			pzsvc.LogInfo(s, errStr)
			canReg = false
		} else {
			s.PzAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(apiKey+":"))
		}
	}

	if configObj.Port <= 0 {
		configObj.Port = 8080
	}
	if configObj.PortEnVar != "" {
		newPort, err := strconv.Atoi(os.Getenv(configObj.PortEnVar))
		if err == nil && newPort > 0 {
			configObj.Port = newPort
		} else {
			pzsvc.LogInfo(s, "Config: Could not find/interpret PortVar properly.  Reverting to port "+strconv.Itoa(configObj.Port))
		}
	}

	portStr := ":" + strconv.Itoa(configObj.Port)
	if configObj.LocalOnly {
		pzsvc.LogInfo(s, "Local Only specified.  Limiting incoming requests to localhost.")
		portStr = "localhost" + portStr
	}

	version := GetVersion(s, configObj)

	if canReg {
		pzsvc.LogInfo(s, "About to manage registration.")

		svcClass := pzsvc.ClassType{Classification: "UNCLASSIFIED"} // TODO: this will have to be updated at some point.
		metaObj := pzsvc.ResMeta{Name: configObj.SvcName,
			Description: configObj.Description,
			ClassType:   svcClass,
			Version:     version,
			Metadata:    make(map[string]string)}
		for key, val := range configObj.Attributes {
			metaObj.Metadata[key] = val
		}

		svcObj := pzsvc.Service{
			ContractURL:   configObj.DocURL,
			Method:        "POST",
			ResMeta:       metaObj,
			Timeout:       configObj.MaxRunTime,
			IsTaskManaged: configObj.RegForTaskMgr}
		if configObj.URL != "" {
			svcObj.URL = configObj.URL + "/execute"
		}

		err := pzsvc.ManageRegistration(s, svcObj)
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

	s.AppName = configObj.SvcName
	s.LogRootDir = "pzsvc-exec"
	s.LogAudit = configObj.LogAudit

	return ConfigParseOut{portStr, version, procPool}, s
}

// Execute does the primary work for pzsvc-exec.  Given a request and various
// blocks of config data, it creates a temporary folder to work in, downloads
// any files indicated in the request (if the configs support it), executes
// the command indicated by the combination of request and configs, uploads
// any files indicated by the request (if the configs support it) and cleans
// up after itself
func Execute(r *http.Request, s pzsvc.Session, configObj ConfigType, procPool pzsvc.Semaphore, version string) (OutStruct, pzsvc.Session) {
	// Makes sure that you only have a certain number of execution tasks firing at once.
	// pzsvc-exec calls can get pretty resource-intensive, and this keeps them from
	// trampling each other into messy deadlock
	procPool.Lock()
	defer procPool.Unlock()

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

	s.SessionID = "FailedOnInit"

	if s.AppName == "" {
		s.AppName = "pzsvc-exec"
	}
	if r.Method != "POST" {
		addOutputError(&output, r.Method+" not supported.  Please us POST.", http.StatusMethodNotAllowed)
		return output, s
	}

	if byts, pErr = pzsvc.ReadBodyJSON(&inpObj, r.Body); err != nil {
		pErr.Log(s, "Could not read request body.  Initial error:")
		addOutputError(&output, "Could not read request body.  Please use JSON format.", http.StatusBadRequest)
		return output, s
	}
	pzsvc.LogInfo(s, "input received.  "+strconv.Itoa(len(byts))+"bytes.")

	s.SessionID, err = pzsvc.PsuUUID()
	if err != nil {
		pzsvc.LogSimpleErr(s, "psuUUID error: ", err)
		addOutputError(&output, "pzsvc-exec internal error.  Check logs for further information.", http.StatusInternalServerError)
		return output, s
	}
	s.SubFold = s.SessionID // they're the same here, but as far as the pzsvc library is concerned, they're different concepts

	if inpObj.PzAddr != "" {
		s.PzAddr = inpObj.PzAddr
	}
	if inpObj.PzAuth != "" {
		s.PzAuth = inpObj.PzAuth
		inpObj.PzAuth = "******" // we shouldnt' log auth data
	}
	if inpObj.ExtAuth != "" {
		s.ExtAuth = inpObj.ExtAuth
		inpObj.ExtAuth = "******" // we still shouldnt' log auth data
	}
	if inpObj.UserID != "" {
		s.UserID = inpObj.UserID
	} else if s.PzAddr != "" && s.PzAuth != "" {
		var profile pzsvc.UserProfileResp
		query := s.PzAddr + "/profile"
		pzsvc.LogAudit(s, s.AppName, "http request - looking for profile", query, "", pzsvc.INFO)
		byts, pErr = pzsvc.RequestKnownJSON("GET", "", query, s.PzAuth, &profile)
		pzsvc.LogAudit(s, query, "http response to profile request", s.AppName, string(byts), pzsvc.INFO)
		if pErr != nil {
			err = pErr.Log(s, "Error finding profile for session")
			addOutputError(&output, "pzsvc-exec internal error.  Check logs for further information.", http.StatusInternalServerError)
			return output, s
		}
		s.UserID = profile.Data.UserProfile.DistinguishedName
	} else {
		s.UserID = "anon user"
	}

	byts, _ = json.Marshal(inpObj)
	pzsvc.LogInfo(s, `pzsvc-exec call initiated.  Input: `+string(byts))
	pzsvc.LogAudit(s, s.UserID, "pzsvc-exec execute call", s.AppName, string(byts), pzsvc.INFO)

	cmdParamSlice := splitOrNil(inpObj.Command, " ")
	cmdConfigSlice := splitOrNil(configObj.CliCmd, " ")
	cmdSlice := append(cmdConfigSlice, cmdParamSlice...)

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
		unlogErr := pzsvc.CheckAuth(s)
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

	pzsvc.LogAudit(s, s.AppName, "creating temp dir "+s.SubFold, "local hard drive", "", pzsvc.INFO)
	err = os.Mkdir("./"+s.SubFold, 0777)
	if err != nil {
		pzsvc.LogSimpleErr(s, "os.Mkdir error: ", err)
		addOutputError(&output, "pzsvc-exec internal error.  Check logs for further information.", http.StatusInternalServerError)
	}
	defer os.RemoveAll("./" + s.SubFold)
	defer pzsvc.LogAudit(s, s.AppName, "deleting temp dir "+s.SubFold, "local hard drive", "", pzsvc.INFO)

	// this is done to enable use of handleFList, which lets us
	// reduce a fair bit of code duplication in plowing through
	// our upload/download lists.  handleFList gets used a fair
	// bit more after the execute call.
	pzDownlFunc := func(dataID, fname, fType string) (string, error) {
		pzsvc.LogAudit(s, s.UserID, "pz File Download", dataID, "", pzsvc.INFO)
		return pzsvc.DownloadByID(s, dataID, fname)
	}
	err = handleFList(s, inpObj.InPzFiles, inpObj.InPzNames, pzDownlFunc, "unspecified", "Pz download", &output, output.InFiles)
	if err != nil {
		return output, s
	}

	extDownlFunc := func(url, fname, fType string) (string, error) {
		pzsvc.LogAudit(s, s.UserID, "external File Download", url, "", pzsvc.INFO)
		return pzsvc.DownloadByURL(s, url, fname, s.ExtAuth, configObj.ExtRetryOn202)
	}
	handleFList(s, inpObj.InExtFiles, inpObj.InExtNames, extDownlFunc, "unspecified", "URL download", &output, output.InFiles)
	if err != nil {
		return output, s
	}

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
		cmdSlice[0] = ("../" + cmdSlice[0])
	}

	clc := exec.Command(cmdSlice[0], cmdSlice[1:]...)
	clc.Dir = s.SubFold

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	clc.Stdout = &stdout
	clc.Stderr = &stderr

	pzsvc.LogInfo(s, "Executing `"+configObj.CliCmd+" "+inpObj.Command+"`.")
	pzsvc.LogAudit(s, s.UserID, "Executing `"+configObj.CliCmd+" "+inpObj.Command+"`.", "cmdLine", "", pzsvc.INFO)
	err = clc.Run()

	output.ProgStdOut = stdout.String()
	output.ProgStdErr = stderr.String()
	pzsvc.LogInfo(s, `Program stdout: `+stdout.String())
	pzsvc.LogInfo(s, `Program stderr: `+stderr.String())

	if err != nil {
		pzsvc.LogSimpleErr(s, "clc.Run error: ", err)
		addOutputError(&output, "pzsvc-exec failed on cmd `"+inpObj.Command+"`.  If that was correct, check logs for further details.", http.StatusBadRequest)
		return output, s
	}

	attMap := make(map[string]string)
	attMap["algoName"] = configObj.SvcName
	attMap["algoVersion"] = version
	attMap["algoCmd"] = configObj.CliCmd + " " + inpObj.Command
	attMap["algoProcTime"] = time.Now().UTC().Format("20060102.150405.99999")

	// this is the other spot that handleFlist gets used, and works on the
	// same principles.

	ingFunc := func(fName, dummy, fType string) (string, error) {
		pzsvc.LogAudit(s, s.UserID, "Pz File Ingest", fName, "", pzsvc.INFO)
		return pzsvc.IngestFile(s, fName, fType, configObj.SvcName, version, attMap)
	}

	// Not checking for errors here because at this point it's mostly redundant, and possibly
	// getting out some info is better than getting out no info.
	handleFList(s, inpObj.OutTiffs, inpObj.OutTiffs, ingFunc, "raster", "upload", &output, output.OutFiles)
	handleFList(s, inpObj.OutTxts, inpObj.OutTxts, ingFunc, "text", "upload", &output, output.OutFiles)
	handleFList(s, inpObj.OutGeoJs, inpObj.OutGeoJs, ingFunc, "geojson", "upload", &output, output.OutFiles)

	return output, s
}
