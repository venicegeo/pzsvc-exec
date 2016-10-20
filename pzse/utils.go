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
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func handleFList(fList, nameList []string, lFunc rangeFunc, fType string, output *OutStruct, fileRec map[string]string, w http.ResponseWriter) {
	for i, f := range fList {
		name := ""
		if len(nameList) > i {
			name = nameList[i]
		}
		outStr, err := lFunc(f, name, fType)
		if err != nil {
			handleError(output, "handleFlist error: ", err, w, http.StatusBadRequest)
		} else if outStr == "" {
			handleError(output, "handleFlist error: ", errors.New(`type "`+fType+`", input "`+f+`" blank result.`), w, http.StatusBadRequest)
		} else {
			fileRec[f] = outStr
		}
	}
}

func handleError(output *OutStruct, addString string, err error, w http.ResponseWriter, httpStat int) {
	if err != nil {
		var outErrStr string
		_, _, line, ok := runtime.Caller(1)
		if ok == true {
			outErrStr = addString + `(pzsvc-exec/main.go, ` + strconv.Itoa(line) + `): ` + err.Error()
		} else {
			outErrStr = addString + `: ` + err.Error()
		}
		output.Errors = append(output.Errors, outErrStr)
		output.HTTPStatus = httpStat
	}
	return
}

func splitOrNil(inString, knife string) []string {
	if inString == "" {
		return nil
	}
	return strings.Split(inString, knife)
}

// GetVersion spits out the best available version for the
// current software, based on the contents fo the config file
func GetVersion(configObj *ConfigType) string {
	vCmdSlice := splitOrNil(configObj.VersionCmd, " ")
	if vCmdSlice != nil {
		vCmd := exec.Command(vCmdSlice[0], vCmdSlice[1:]...)
		verB, err := vCmd.Output()
		if err != nil {
			fmt.Println("error: VersionCmd failed: " + err.Error())
		}
		if string(verB) != "" {
			return string(verB)
		}
	}
	return configObj.VersionStr
}

// CheckConfig takes an input config file, checks it over for issues,
// and outputs any issues or concerns to std.out.  It returns whether
// or not the config file permits autoregistration, and whether or not
// it permits file upload/download.
func CheckConfig(configObj *ConfigType) (bool, bool, bool) {
	canReg := true
	canFile := true
	hasAuth := true
	if configObj.CliCmd == "" {
		fmt.Println(`Config: Warning: CliCmd is blank.  This is a major security vulnerability.`)
	}

	if configObj.PzAddr == "" {
		fmt.Println(`Config: PzAddr not specified.  Client will have to provide Piazza Address for upload/download.  Autoregistration and file upload/download disabled.`)
		canFile = false
		hasAuth = false
		canReg = false
	} else if configObj.AuthEnVar == "" {
		fmt.Println(`Config: AuthEnVar was not specified.  Client will have to provide authKey for upload/download.  Autoregistration disabled.`)
		hasAuth = false
		canReg = false
	} else if configObj.SvcName == "" {
		fmt.Println(`Config: SvcName not specified.  Autoregistration disabled.`)
		canReg = false
	} else if configObj.URL == "" {
		fmt.Println(`Config: URL not specified for this service.  Autoregistration disabled.`)
		canReg = false
	}

	if !canFile {
		if configObj.VersionCmd != "" {
			fmt.Println(`Config: VersionCmd was specified, but is much less useful without autoregistration.`)
		}
		if configObj.VersionStr != "" {
			fmt.Println(`Config: VersionStr was specified, but is much less useful without without autoregistration.`)
		}
		if configObj.AuthEnVar != "" {
			fmt.Println(`Config: AuthEnVar was specified, but is meaningless without upload/download/autoregistration.`)
		}
	} else {
		if configObj.VersionCmd == "" && configObj.VersionStr == "" {
			fmt.Println(`Config: neither VersionCmd nor VersionStr was specified.  Version will be left blank.`)
		}
		if configObj.VersionCmd != "" && configObj.VersionStr != "" {
			fmt.Println(`Config: Both VersionCmd and VersionStr were specified.  Redundant.  Default to VersionCmd.`)
		}
	}

	if !canReg {
		if configObj.SvcName != "" {
			fmt.Println(`Config: SvcName was specified, but is meaningless without autoregistration.`)
		}
		if configObj.URL != "" {
			fmt.Println(`Config: URL was specified, but is meaningless without autoregistration.`)
		}
	} else {
		if configObj.Description == "" {
			fmt.Println(`Config: Description not specified.  When autoregistering, descriptions are strongly encouraged.`)
		}
	}

	if configObj.Port <= 0 {
		fmt.Println(`Config: Port not specified, or incorrect format.  Default to 8080.`)
	}

	return canReg, canFile, hasAuth
}

// PrintHelp prints out a basic helpfile to make things easier on direct users
func PrintHelp(w http.ResponseWriter) {
	fmt.Fprintln(w, `The pzsvc-exec service endpoints are as follows:`)
	fmt.Fprintln(w, `- '/': entry point.  Displays base command if any, and suggests other endpoints.`)
	fmt.Fprintln(w, `- '/execute': The meat of the program.  Downloads files, executes on them, and uploads the results.`)
	fmt.Fprintln(w, `See the Service Request Format section of the Readme for interface details.`)
	fmt.Fprintln(w, `(Readme available at https://github.com/venicegeo/pzsvc-exec).`)
	fmt.Fprintln(w, `- '/description': When enabled, provides a description of this particular pzsvc-exec instance.`)
	fmt.Fprintln(w, `- '/attributes': When enabled, provides a list of key/value attributes for this pzsvc-exec instance.`)
	fmt.Fprintln(w, `- '/version': When enabled, provides version number for the application served by this pzsvc-exec instance.`)
	fmt.Fprintln(w, `- '/help': This screen.`)
}
