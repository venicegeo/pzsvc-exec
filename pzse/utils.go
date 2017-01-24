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
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

func handleFList(s pzsvc.Session, fList, nameList []string, lFunc rangeFunc, fType, action string, output *OutStruct, fileRec map[string]string) error {
	re := regexp.MustCompile(`^[\w\-\.]*$`)
	for i, f := range fList {
		name := ""
		if len(nameList) > i {
			name = nameList[i]
		}
		if !re.Match([]byte(name)) {
			pzsvc.LogAlert(s, `Illegal filename "`+name+`" entered for `+action+`.  Possible attempted security breach.`)
			addOutputError(output, `handleFlist error: Filename "`+name+`" contains illegal characters and is not permitted.`, http.StatusBadRequest)
			return fmt.Errorf("failure on file handling: illegal filename")
		}
		outStr, err := lFunc(f, name, fType)
		if err != nil {
			addOutputError(output, "Error in "+action+" of "+name+".", http.StatusBadRequest)
			return fmt.Errorf("failure on file handling: request error: %s", err.Error())
		} else if outStr == "" {
			pzsvc.LogSimpleErr(s, `handleFlist error: type "`+fType+`", input "`+f+`" blank result.`, nil)
			addOutputError(output, "Blank Result Error in "+action+" of "+name+".", http.StatusBadRequest)
			return fmt.Errorf("failure on file handling: empty output")
		} else {
			fileRec[f] = outStr
		}
	}
	return nil
}

func addOutputError(output *OutStruct, errString string, httpStat int) {
	output.Errors = append(output.Errors, errString)
	output.HTTPStatus = httpStat
}

func splitOrNil(inString, knife string) []string {
	if inString == "" {
		return nil
	}
	return strings.Split(inString, knife)
}

// GetVersion spits out the best available version for the
// current software, based on the contents fo the config file
func GetVersion(s pzsvc.Session, configObj *ConfigType) string {
	vCmdSlice := splitOrNil(configObj.VersionCmd, " ")
	if vCmdSlice != nil {
		vCmd := exec.Command(vCmdSlice[0], vCmdSlice[1:]...)

		// if stdout exists, get that.
		// if it doesn't, and there wasnt' an error, and stderr exists, get that.
		// trim leading/trailign whitespace regardless

		verB, err := vCmd.CombinedOutput()
		verStr := strings.TrimSpace(string(verB))
		pzsvc.LogInfo(s, `Called VersionCmd `+configObj.VersionCmd+`.  Results: `+verStr)
		if err != nil {
			pzsvc.LogSimpleErr(s, "VersionCmd failed: ", err)
		}
		if verStr != "" {
			return verStr
		}
	}
	return configObj.VersionStr
}

// CheckConfig takes an input config file, checks it over for issues,
// and outputs any issues or concerns to std.out.  It returns whether
// or not the config file permits autoregistration.
func CheckConfig(s pzsvc.Session, configObj *ConfigType) bool {
	canReg := true
	canPzFile := configObj.CanUpload || configObj.CanDownlPz
	if configObj.CliCmd == "" {
		pzsvc.LogAlert(s, `Config: Warning: CliCmd is blank.  This is a major security vulnerability.`)
	}

	if configObj.PzAddr == "" && configObj.PzAddrEnVar == "" {
		errStr := `Config: Did not specify PzAddr or PzAddrEnVar.  Autoregistration disabled.`
		if canPzFile {
			errStr += `  Client will have to provide Piazza Address for uploads and Piazza downloads.`
		}
		pzsvc.LogInfo(s, errStr)
		canReg = false
	} else if configObj.APIKeyEnVar == "" {
		errStr := `Config: APIKeyEnVar was not specified.  Autoregistration disabled.`
		if canPzFile {
			errStr += `  Client will have to provide authKey for uploads and Piazza downloads.`
		}
		pzsvc.LogInfo(s, errStr)
		canReg = false
	} else if configObj.SvcName == "" {
		pzsvc.LogInfo(s, `Config: SvcName not specified.  Autoregistration disabled.`)
		canReg = false
	} else if configObj.URL == "" && configObj.RegForTaskMgr == false {
		pzsvc.LogInfo(s, `Config: URL not specified.  URL required unless registering for task manager.  Autoregistration disabled.`)
		canReg = false
	} else if configObj.RegForTaskMgr && configObj.MaxRunTime == 0 {
		pzsvc.LogInfo(s, `Config: Cannot register for task manager use without MaxRunTime (in seconds).  Registration disabled.`)
		canReg = false
	}

	if !canReg {
		if configObj.VersionCmd != "" {
			pzsvc.LogInfo(s, `Config: VersionCmd was specified, but is much less useful without autoregistration.`)
		}
		if configObj.VersionStr != "" {
			pzsvc.LogInfo(s, `Config: VersionStr was specified, but is much less useful without without autoregistration.`)
		}
		if configObj.APIKeyEnVar != "" {
			if canPzFile {
				pzsvc.LogInfo(s, `Config: APIKeyEnVar was specified, but PzAddr was not.  APIKeyEnVar useless without a Pz instance to authenticate against.`)
			} else {
				pzsvc.LogInfo(s, `Config: APIKeyEnVar was specified, but is meaningless without autoregistration or Pz file interactions.`)
			}
		}
		if configObj.SvcName != "" {
			pzsvc.LogInfo(s, `Config: SvcName was specified, but is meaningless without autoregistration.`)
		}
		if configObj.URL != "" {
			pzsvc.LogInfo(s, `Config: URL was specified, but is meaningless without autoregistration.`)
		}
		if configObj.RegForTaskMgr {
			pzsvc.LogInfo(s, `Config: RegForTaskMgr was specified as true, but is meaningless without autoregistration.`)
		}
		if configObj.MaxRunTime == 0 {
			pzsvc.LogInfo(s, `Config: MaxRunTime was specified, but is meaningless without autoregistration.`)
		}
	} else {
		if configObj.PzAddr == "" && configObj.PzAddrEnVar == "" {
			pzsvc.LogInfo(s, `Config: Both PzAddr and PzAddrEnVar were specified.  Redundant.  Default to PzAddrEnVar.`)
		}
		if configObj.VersionCmd == "" && configObj.VersionStr == "" {
			pzsvc.LogInfo(s, `Config: Neither VersionCmd nor VersionStr were specified.  Version will be left blank.`)
		}
		if configObj.VersionCmd != "" && configObj.VersionStr != "" {
			pzsvc.LogInfo(s, `Config: Both VersionCmd and VersionStr were specified.  Redundant.  Default to VersionCmd.`)
		}
		if configObj.Description == "" {
			pzsvc.LogInfo(s, `Config: Description not specified.  When autoregistering, descriptions are strongly encouraged.`)
		}
		if configObj.MaxRunTime != 0 && !configObj.RegForTaskMgr {
			pzsvc.LogInfo(s, `Config: MaxRunTime not meaningful unless registering for task manager use.`)
		}

	}

	if configObj.Port <= 0 && configObj.PortEnVar == "" {
		pzsvc.LogInfo(s, `Config: Neither Port nor PortEnVar were properly specified.  Default to Port 8080.`)
	}
	if configObj.Port > 0 && configObj.PortEnVar != "" {
		pzsvc.LogInfo(s, `Config: Both Port and PortEnVar were specified.  Redundant.  Default to PortEnVar.`)
	}

	return canReg
}

// PrintHelp prints out a basic helpfile to make things easier on direct users
func PrintHelp(w http.ResponseWriter) {
	fmt.Fprintln(w, `The pzsvc-exec service endpoints are as follows:`)
	fmt.Fprintln(w, `- '/': entry point.  Displays base command if any, and suggests other endpoints.`)
	fmt.Fprintln(w, `- '/execute': The meat of the program.  Downloads files, executes on them, and uploads the results.`)
	fmt.Fprintln(w, `See the Service Request Format section of the Readme for interface details.`)
	fmt.Fprintln(w, `(Readme available at https://github.com/venicegeo/pzsvc-exec).`)
	fmt.Fprintln(w, `- '/description': When enabled, provides a description of this particular pzsvc-exec instance.`)
	fmt.Fprintln(w, `- '/documentation': When enabled, provides a url containing documentation for this particular pzsvc-exec instance.`)
	fmt.Fprintln(w, `- '/attributes': When enabled, provides a list of key/value attributes for this pzsvc-exec instance.`)
	fmt.Fprintln(w, `- '/version': When enabled, provides version number for the application served by this pzsvc-exec instance.`)
	fmt.Fprintln(w, `- '/help': This screen.`)
}
