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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/venicegeo/pzsvc-exec/pzse"
	"github.com/venicegeo/pzsvc-lib"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("error: Insufficient parameters.  You must specify a config file.")
		return
	}

	// First argument after the base call should be the path to the config file.
	// ReadFile returns the contents of the file as a byte buffer.
	configBuf, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("pzsvc-exec error in reading config: " + err.Error())
	}
	var configObj pzse.ConfigType
	err = json.Unmarshal(configBuf, &configObj)
	if err != nil {
		fmt.Println("pzsvc-exec error in unmarshalling config: " + err.Error())
	}
	canReg, canFile, hasAuth := pzse.CheckConfig(&configObj)

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

	version := pzse.GetVersion(configObj)

	var procPool = pzsvc.Semaphore(nil)
	if configObj.NumProcs > 0 {
		procPool = make(pzsvc.Semaphore, configObj.NumProcs)
	}

	if canReg {
		fmt.Println("About to manage registration.")
		err = pzsvc.ManageRegistration(configObj.SvcName,
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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		switch r.URL.Path {
		case "/":
			{
				fmt.Fprintf(w, "Hello.  This is pzsvc-exec")
				if configObj.SvcName != "" {
					fmt.Fprintf(w, ", serving %s", configObj.SvcName)
				}
				fmt.Fprintf(w, ".\nWere you possibly looking for the /help or /execute endpoints?")
			}
		case "/execute":
			{
				// the other options are shallow and informational.  This is the
				// place where the work gets done.
				output := pzse.Execute(w, r, configObj, authKey, version, canFile, procPool)
				pzsvc.PrintJSON(w, output, output.HTTPStatus)
			}
		case "/description":
			if configObj.Description == "" {
				fmt.Fprintf(w, "No description defined")
			} else {
				fmt.Fprintf(w, configObj.Description)
			}
		case "/attributes":
			if configObj.Attributes == nil {
				fmt.Fprintf(w, "{ }")
			} else {
				pzsvc.PrintJSON(w, configObj.Attributes, http.StatusOK)
			}
		case "/help":
			pzse.PrintHelp(w)
		case "/version":
			fmt.Fprintf(w, version)
		default:
			fmt.Fprintln(w, "Endpoint undefined.  Try /help?")
		}
	})

	log.Fatal(http.ListenAndServe(portStr, nil))
}
