// Copyright 2018, RadiantBlue Technologies, Inc.
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
	"encoding/base64"
	"os"
	"strconv"
)

// Config represents and contains the information from a pzsvc-exec config file.
type Config struct {
	CliCmd        string            // The first segment of the command to send to the CLI.  Security vulnerability when blank.
	VersionStr    string            // The version number of the underlying CLI.  Redundant with VersionCmd
	VersionCmd    string            // The command to run to determine the version number of the underlying CLI.  Redundant with VersionStr
	PzAddr        string            // Address of local Piazza instance.  Used for Piazza file access.  Necessary for autoregistration, task worker.
	PzAddrEnVar   string            // Environment variable holding Piazza address.  Used to populate/overwrite PzAddr if present
	APIKeyEnVar   string            // The environment variable containing the api key for the local Piazza instance.  Used for the same things.
	SvcName       string            // The name to give for this service when registering.  Necessary for autoregistration, task worker.
	URL           string            // URL to give when registering.  Required when registering and not using task manager.
	Port          int               // Port to publish this service on.  Defaults to 8080.
	PortEnVar     string            // Environment variable to check for port.  Mutually exclusive with "Port"
	Description   string            // Description to return when asked.
	Attributes    map[string]string // Service attributes.  Used to improve searching/sorting of services.
	NumProcs      int               // Number of jobs a single instance of this service can handle simultaneously
	CanUpload     bool              // True if this service is permitted to upload files
	CanDownlPz    bool              // True if this service is permitted to download files from Piazza
	CanDownlExt   bool              // True if this service is permitted to download files from an external source
	RegForTaskMgr bool              // True if autoregistration should be as a service using the Pz task manager
	MaxRunTime    int               // Time in seconds before a running job should be considered to have failed.  Used for task worker registration.
	LocalOnly     bool              // True if service should only accept connections from localhost (used with task worker)
	LogAudit      bool              // True to log all auditable events
	LimitUserData bool              // True to limit the information availabel to the individual user
	ExtRetryOn202 bool              // If true, will retry when receiving a 202 response from external file download links
	DocURL        string            // URL to provide to autoregistration and to documentation endpoint for info about the service
	//JwtSecAuthURL string            // URL for taskworker to decrypt JWT.  If nonblank, will assume that all jobs are JWT format, and will require decrypting.
}

// ConfigParseOut is a handy struct to organize all of the outputs
// for pzse.ConfigParse() and prevent potential confusion.
type ConfigParseOut struct {
	PortStr  string
	Version  string
	ProcPool Semaphore
}

// ParseConfigAndRegister parses the config file on starting up, manages
// registration for it on the given Pz instance if registration management
// is called for, and returns a few useful derived values
func ParseConfigAndRegister(s Session, configObj *Config) (ConfigParseOut, Session) {
	canReg := checkConfig(s, configObj)
	canPzFile := configObj.CanUpload || configObj.CanDownlPz

	s.PzAddr = configObj.PzAddr
	if configObj.PzAddrEnVar != "" {
		newAddr := os.Getenv(configObj.PzAddrEnVar)
		if newAddr != "" {
			s.PzAddr = newAddr
			LogInfo(s, `Config: PzAddr updated to `+configObj.PzAddr+` based on PzAddrEnVar.`)
		} else if s.PzAddr != "" {
			LogInfo(s, `Config: PzAddrEnVar specified in config, but no such env var exists.  Reverting to specified PzAddr.`)
		} else {
			logStr := `Config: PzAddrEnVar specified in config, but no such env var exists, and PzAddr not specified.`
			if canReg {
				logStr += `  Autoregistration disabled.`
				canReg = false
			}
			if canPzFile {
				logStr += `  Client will have to provide Piazza Address for uploads and Piazza downloads.`
			}
			LogInfo(s, logStr)
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
			LogInfo(s, errStr)
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
			LogInfo(s, "Config: Could not find/interpret PortVar properly.  Reverting to port "+strconv.Itoa(configObj.Port))
		}
	}

	portStr := ":" + strconv.Itoa(configObj.Port)
	if configObj.LocalOnly {
		LogInfo(s, "Local Only specified.  Limiting incoming requests to localhost.")
		portStr = "localhost" + portStr
	}

	version := getVersion(s, configObj)

	if canReg {
		LogInfo(s, "About to manage registration.")

		svcClass := ClassType{Classification: "UNCLASSIFIED"} // TODO: this will have to be updated at some point.
		metaObj := ResMeta{Name: configObj.SvcName,
			Description: configObj.Description,
			ClassType:   svcClass,
			Version:     version,
			Metadata:    make(map[string]string)}
		for key, val := range configObj.Attributes {
			metaObj.Metadata[key] = val
		}

		svcObj := Service{
			ContractURL:   configObj.DocURL,
			Method:        "POST",
			ResMeta:       metaObj,
			Timeout:       configObj.MaxRunTime,
			IsTaskManaged: configObj.RegForTaskMgr}
		if configObj.URL != "" {
			svcObj.URL = configObj.URL + "/execute"
		}

		err := ManageRegistration(s, svcObj)
		if err != nil {
			LogSimpleErr(s, "pzsvc-exec error in managing registration: ", err)
		} else {
			LogInfo(s, "Registration managed.")
		}
	}

	var procPool = Semaphore(nil)
	if configObj.NumProcs > 0 {
		procPool = make(Semaphore, configObj.NumProcs)
	}

	s.AppName = configObj.SvcName
	s.LogRootDir = "pzsvc-exec"
	s.LogAudit = configObj.LogAudit

	return ConfigParseOut{portStr, version, procPool}, s
}

// checkConfig takes an input config file, checks it over for issues,
// and outputs any issues or concerns to std.out.  It returns whether
// or not the config file permits autoregistration.
func checkConfig(s Session, configObj *Config) bool {
	canReg := true
	canPzFile := configObj.CanUpload || configObj.CanDownlPz
	if configObj.CliCmd == "" {
		LogAlert(s, `Config: Warning: CliCmd is blank.  This is a major security vulnerability.`)
	}

	if configObj.PzAddr == "" && configObj.PzAddrEnVar == "" {
		errStr := `Config: Did not specify PzAddr or PzAddrEnVar.  Autoregistration disabled.`
		if canPzFile {
			errStr += `  Client will have to provide Piazza Address for uploads and Piazza downloads.`
		}
		LogInfo(s, errStr)
		canReg = false
	} else if configObj.APIKeyEnVar == "" {
		errStr := `Config: APIKeyEnVar was not specified.  Autoregistration disabled.`
		if canPzFile {
			errStr += `  Client will have to provide authKey for uploads and Piazza downloads.`
		}
		LogInfo(s, errStr)
		canReg = false
	} else if configObj.SvcName == "" {
		LogInfo(s, `Config: SvcName not specified.  Autoregistration disabled.`)
		canReg = false
	} else if configObj.URL == "" && configObj.RegForTaskMgr == false {
		LogInfo(s, `Config: URL not specified.  URL required unless registering for task manager.  Autoregistration disabled.`)
		canReg = false
	} else if configObj.RegForTaskMgr && configObj.MaxRunTime == 0 {
		LogInfo(s, `Config: Cannot register for task manager use without MaxRunTime (in seconds).  Registration disabled.`)
		canReg = false
	}

	if !canReg {
		if configObj.VersionCmd != "" {
			LogInfo(s, `Config: VersionCmd was specified, but is much less useful without autoregistration.`)
		}
		if configObj.VersionStr != "" {
			LogInfo(s, `Config: VersionStr was specified, but is much less useful without without autoregistration.`)
		}
		if configObj.APIKeyEnVar != "" {
			if canPzFile {
				LogInfo(s, `Config: APIKeyEnVar was specified, but PzAddr was not.  APIKeyEnVar useless without a Pz instance to authenticate against.`)
			} else {
				LogInfo(s, `Config: APIKeyEnVar was specified, but is meaningless without autoregistration or Pz file interactions.`)
			}
		}
		if configObj.SvcName != "" {
			LogInfo(s, `Config: SvcName was specified, but is meaningless without autoregistration.`)
		}
		if configObj.URL != "" {
			LogInfo(s, `Config: URL was specified, but is meaningless without autoregistration.`)
		}
		if configObj.RegForTaskMgr {
			LogInfo(s, `Config: RegForTaskMgr was specified as true, but is meaningless without autoregistration.`)
		}
		if configObj.MaxRunTime == 0 {
			LogInfo(s, `Config: MaxRunTime was specified, but is meaningless without autoregistration.`)
		}
	} else {
		if configObj.PzAddr == "" && configObj.PzAddrEnVar == "" {
			LogInfo(s, `Config: Both PzAddr and PzAddrEnVar were specified.  Redundant.  Default to PzAddrEnVar.`)
		}
		if configObj.VersionCmd == "" && configObj.VersionStr == "" {
			LogInfo(s, `Config: Neither VersionCmd nor VersionStr were specified.  Version will be left blank.`)
		}
		if configObj.VersionCmd != "" && configObj.VersionStr != "" {
			LogInfo(s, `Config: Both VersionCmd and VersionStr were specified.  Redundant.  Default to VersionCmd.`)
		}
		if configObj.Description == "" {
			LogInfo(s, `Config: Description not specified.  When autoregistering, descriptions are strongly encouraged.`)
		}
		if configObj.MaxRunTime != 0 && !configObj.RegForTaskMgr {
			LogInfo(s, `Config: MaxRunTime not meaningful unless registering for task manager use.`)
		}
	}

	if configObj.Port <= 0 && configObj.PortEnVar == "" {
		LogInfo(s, `Config: Neither Port nor PortEnVar were properly specified.  Default to Port 8080.`)
	}
	if configObj.Port > 0 && configObj.PortEnVar != "" {
		LogInfo(s, `Config: Both Port and PortEnVar were specified.  Redundant.  Default to PortEnVar.`)
	}

	return canReg
}
