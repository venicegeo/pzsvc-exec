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
//"net/http"
)

func getTestConfigWorkable() ConfigType {
	return ConfigType{
		CliCmd:      "ls",
		VersionCmd:  "",
		VersionStr:  "0.0",
		PzAddr:      "aaa",
		AuthEnVar:   "TESTENV",
		SvcName:     "testSvc",
		URL:         "www.testSvc.nope",
		Port:        0,
		Description: "Insert Description Here",
		Attributes:  map[string]string{"testAttr": "testAtt"},
		NumProcs:    3,
		CanUpload:   true,
		CanDownlPz:  true,
		CanDownlExt: true}
}

func getTestConfigList() ([6]ConfigType, [6]ConfigParseOut, string) {
	cliCmd := "ls -l"
	versionCmd := "echo vers1"
	versionStr := "vers2"
	pzAddr := "aaa"
	authEnVar := "APP"
	svcName := "testSvc"
	url := "www.testSvc.nope"
	port := 8081
	desc := "Insert Description Here"
	attr := map[string]string{"testAttr": "testAtt"}
	numProcs := 0 //We'll test this part, but we'll do it on the full run-through, rather than the config-tester
	var configList [6]ConfigType
	var configParseList [6]ConfigParseOut
	configList[0] = ConfigType{}
	configParseList[0] = ConfigParseOut{"", ":8080", "", nil}

	configList[1] = ConfigType{cliCmd, versionCmd, versionStr, "", authEnVar, svcName, url, 0, desc, attr, numProcs, false, false, false}
	configParseList[1] = ConfigParseOut{"", ":8080", "vers1\n", nil}

	configList[2] = ConfigType{cliCmd, versionCmd, "", pzAddr, "", "", url, port, desc, attr, numProcs, true, true, true}
	configParseList[2] = ConfigParseOut{"", ":8081", "vers1\n", nil}

	configList[3] = ConfigType{cliCmd, versionCmd, "", pzAddr, authEnVar, "", url, port, desc, attr, numProcs, true, true, true}
	configParseList[3] = ConfigParseOut{"pzsvc-exec", ":8081", "vers1\n", nil}

	configList[4] = ConfigType{cliCmd, "", versionStr, pzAddr, authEnVar, svcName, "", port, desc, attr, numProcs, true, true, true}
	configParseList[4] = ConfigParseOut{"pzsvc-exec", ":8081", "vers2", nil}

	configList[5] = ConfigType{cliCmd, "echo", "", versionStr, authEnVar, svcName, url, port, "", attr, numProcs, true, true, true}
	configParseList[5] = ConfigParseOut{"pzsvc-exec", ":8081", "\n", nil}

	return configList, configParseList, authEnVar
}
