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

	"github.com/venicegeo/pzsvc-exec/pzse"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

func main() {

	s := pzsvc.Session{AppName: "pzsvc-exec", SessionID: "startup", UserID: "local", LogRootDir: "pzsvc-exec"}

	pzsvc.LogAudit(s, s.AppName, "startup", s.AppName)

	if len(os.Args) < 2 {
		pzsvc.LogSimpleErr(s, "error: Insufficient parameters.  You must specify a config file.", nil)
		return
	}

	// First argument after the base call should be the path to the config file.
	// ReadFile returns the contents of the file as a byte buffer.
	pzsvc.LogAudit(s, s.AppName, "read config", os.Args[1])
	configBuf, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		pzsvc.LogSimpleErr(s, "pzsvc-exec error in reading config: ", err)
		return
	}
	pzsvc.LogAudit(s, s.AppName, "read file", "config file "+os.Args[1])
	var configObj pzse.ConfigType
	err = json.Unmarshal(configBuf, &configObj)
	if err != nil {
		pzsvc.LogSimpleErr(s, "pzsvc-exec error in unmarshalling config: ", err)
		return
	}

	s.LogAudit = configObj.LogAudit
	var pRes pzse.ConfigParseOut
	pRes, s = pzse.ParseConfigAndRegister(s, &configObj)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// check config: do we have a security authority set?
		//   if so, check incoming message.  Does it have an auth?
		// identify user here
		// s.UserID = ???

		switch r.URL.Path {
		case "/":
			pzsvc.LogAudit(s, r.RemoteAddr, "hello request", s.AppName)
			fmt.Fprintf(w, "Hello.  This is pzsvc-exec")
			if configObj.SvcName != "" {
				fmt.Fprintf(w, ", serving %s", configObj.SvcName)
			}
			fmt.Fprintf(w, ".\nWere you possibly looking for the /help or /execute endpoints?")
			pzsvc.LogAudit(s, s.AppName, "hello response", r.RemoteAddr)
		case "/execute":
			pzsvc.LogAudit(s, r.RemoteAddr, "execution request", s.AppName)
			// the other options are shallow and informational.  This is the
			// place where the work gets done.
			output, s2 := pzse.Execute(r, s, configObj, pRes.ProcPool, pRes.Version)
			byts := pzsvc.PrintJSON(w, output, output.HTTPStatus)
			pzsvc.LogInfo(s2, `pzsvc-exec call completed.  Output: `+string(byts))
			pzsvc.LogAudit(s, s.AppName, "execution response", r.RemoteAddr+"("+s2.UserID+")")
		case "/description":
			pzsvc.LogAudit(s, r.RemoteAddr, "description request", s.AppName)
			if configObj.Description == "" {
				fmt.Fprintf(w, "No description defined")
			} else {
				fmt.Fprintf(w, configObj.Description)
			}
			pzsvc.LogAudit(s, s.AppName, "description response", r.RemoteAddr)
		case "/documentation":
			pzsvc.LogAudit(s, r.RemoteAddr, "doc URL request", s.AppName)
			if configObj.DocURL == "" {
				fmt.Fprintf(w, "No URL provided")
			} else {
				fmt.Fprintf(w, configObj.DocURL)
			}
			pzsvc.LogAudit(s, s.AppName, "doc URL response", r.RemoteAddr)

		case "/attributes":
			pzsvc.LogAudit(s, r.RemoteAddr, "attributes request", s.AppName)
			if configObj.Attributes == nil {
				fmt.Fprintf(w, "{ }")
			} else {
				pzsvc.PrintJSON(w, configObj.Attributes, http.StatusOK)
			}
			pzsvc.LogAudit(s, s.AppName, "attributes response", r.RemoteAddr)
		case "/help":
			pzsvc.LogAudit(s, r.RemoteAddr, "help request", s.AppName)
			pzse.PrintHelp(w)
			pzsvc.LogAudit(s, s.AppName, "help response", r.RemoteAddr)
		case "/version":
			pzsvc.LogAudit(s, r.RemoteAddr, "version request", s.AppName)
			fmt.Fprintf(w, pRes.Version)
			pzsvc.LogAudit(s, s.AppName, "version response", r.RemoteAddr)
		default:
			pzsvc.LogAudit(s, r.RemoteAddr, "undefined endpoint request: "+r.URL.Path, s.AppName)
			fmt.Fprintln(w, "Endpoint undefined.  Try /help?")
			pzsvc.LogAudit(s, s.AppName, "undefined endpoint response", r.RemoteAddr)
		}
	})

	log.Print(http.ListenAndServe(pRes.PortStr, nil))
	pzsvc.LogAudit(s, s.AppName, "shutdown", s.AppName)
	return

}
