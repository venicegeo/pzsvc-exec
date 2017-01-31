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

func getTestConfigWorkable() ConfigType {
	return ConfigType{
		CliCmd:      "ls",
		VersionCmd:  "",
		VersionStr:  "0.0",
		PzAddr:      "aaa",
		APIKeyEnVar: "TESTENV",
		SvcName:     "",
		URL:         "www.testSvc.nope",
		Port:        0,
		Description: "Insert Description Here",
		Attributes:  map[string]string{"testAttr": "testAtt"},
		NumProcs:    3,
		CanUpload:   true,
		CanDownlPz:  true,
		CanDownlExt: true}
}

func getTestConfigList() ([6]ConfigType, [6]ConfigParseOut) {

	DefaultConfig := ConfigType{
		CliCmd:        "ls -l",
		VersionCmd:    "echo vers1",
		VersionStr:    "vers2",
		PzAddr:        "aaa",
		PzAddrEnVar:   "addrEnv",
		APIKeyEnVar:   "apiKeyEnv",
		SvcName:       "testSvc",
		URL:           "www.testSvc.nope",
		Port:          8081,
		PortEnVar:     "portEnv",
		Description:   "Insert Description Here",
		Attributes:    map[string]string{"testAttr": "testAtt"},
		NumProcs:      0, //We'll test this part, but we'll do it on the full run-through, rather than the config-tester
		CanUpload:     true,
		CanDownlPz:    true,
		CanDownlExt:   true,
		RegForTaskMgr: true,
		MaxRunTime:    50,
		LocalOnly:     false,
		LogAudit:      true,
		LimitUserData: false,
		ExtRetryOn202: true,
		DocURL:        "fakeURL",
	}
	var configList [6]ConfigType
	var configParseList [6]ConfigParseOut
	configList[0] = ConfigType{}
	configParseList[0] = ConfigParseOut{":8080", "", nil}

	//configList[1] = ConfigType{cliCmd, versionCmd, versionStr, "", apiKeyEnVar, svcName, url, 0, desc, attr, numProcs, false, false, false}
	configList[1] = DefaultConfig
	configList[1].PzAddr = ""
	configList[1].PzAddrEnVar = "blankEnv"
	configList[1].Port = 0
	configList[1].PortEnVar = "blankEnv"
	configList[1].CanUpload = false
	configList[1].CanDownlPz = false
	configList[1].CanDownlExt = false
	configParseList[1] = ConfigParseOut{":8080", "vers1", nil}

	//configList[2] = ConfigType{cliCmd, versionCmd, "", pzAddr, "", "", url, port, desc, attr, numProcs, true, true, true}
	configList[2] = DefaultConfig
	configList[2].PzAddrEnVar = "blankEnv"
	configList[2].VersionStr = ""
	configList[2].APIKeyEnVar = "blankEnv"
	configList[2].SvcName = ""
	configParseList[2] = ConfigParseOut{":8081", "vers1", nil}

	//configList[3] = ConfigType{cliCmd, versionCmd, "", pzAddr, apiKeyEnVar, "", url, port, desc, attr, numProcs, true, true, true}
	configList[3] = DefaultConfig
	configList[3].VersionStr = ""
	configList[3].SvcName = ""
	configParseList[3] = ConfigParseOut{":8081", "vers1", nil}

	//configList[4] = ConfigType{cliCmd, "", versionStr, pzAddr, apiKeyEnVar, svcName, "", port, desc, attr, numProcs, true, true, true}
	configList[4] = DefaultConfig
	configList[4].VersionCmd = ""
	configList[4].URL = ""
	configList[4].LocalOnly = true
	configParseList[4] = ConfigParseOut{"localhost:8081", "vers2", nil}

	//configList[5] = ConfigType{cliCmd, "echo", "", versionStr, apiKeyEnVar, svcName, url, port, "", attr, numProcs, true, true, true}
	configList[5] = DefaultConfig
	configList[5].VersionCmd = "echo"
	configList[5].VersionStr = ""
	configList[5].PzAddr = DefaultConfig.VersionStr
	configList[5].Description = ""
	configParseList[5] = ConfigParseOut{":8081", "", nil}

	return configList, configParseList
}
