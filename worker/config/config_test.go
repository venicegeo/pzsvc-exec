package config

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInputSource_NoError(t *testing.T) {
	src, err := ParseInputSource("testFile.txt:http://example.localdomain/mockTestFile.txt")
	assert.Nil(t, err)
	assert.Equal(t, "testFile.txt", src.FileName)
	assert.Equal(t, "http://example.localdomain/mockTestFile.txt", src.URL)
}

func TestParseInputSource_BadFormat(t *testing.T) {
	src, err := ParseInputSource("asdf1234")
	assert.Nil(t, src)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Invalid input source string")
}

func TestReadPzSEConfig(t *testing.T) {
	// from 'examplecfg.txt' in base repo dir
	testConfig := `{
	    "CliCmd":"python ../bfalg-ndwi.py --outdir .",
	    "VersionStr":"",
	    "VersionCmd":"python ./bfalg-ndwi.py --version",
	    "PzAddr":"",
	    "PzAddrEvVar":"PZ_ADDR",
	    "APIKeyEnVar":"PZ_API_KEY",
	    "SvcName":"BF_Algo_NDWI_PY",
	    "URL":"https://pzsvc-exec.stage.geointservices.io",
	    "Port":8080,
	    "PortEnVar":"PORT",
	    "Description":"Shoreline Detection using the NDWI Algorithm via Beachfront's own python script.",
	    "Attributes":{
	        "SvcType":"beachfront",
	        "Interface":"pzsvc-ndwi",
	        "ImgReq - CloudCover":"4%",
	        "ImgReq - Bands":"3,6",
	        "ImgReq - Coastline":"Yes"
	    },
	    "NumProcs":3,
	    "CanUpload":true,
	    "CanDownlPz":true,
	    "CanDownlExt":true,
	    "RegForTaskMgr":false,
	    "MaxRunTime":0,
	    "LocalOnly":false,
	    "LogAudit":false
	}`
	f, err := ioutil.TempFile("", "test-pzse-config")
	if err != nil {
		assert.Fail(t, "could not create temporary pzse config file: ", err)
		return
	}
	f.WriteString(testConfig)
	f.Close()
	defer os.Remove(f.Name())

	wc := WorkerConfig{}
	err = wc.ReadPzSEConfig(f.Name())

	assert.Nil(t, err)
	assert.Equal(t, "python ../bfalg-ndwi.py --outdir .", wc.PzSEConfig.CliCmd)
}

func TestInputsAsMap(t *testing.T) {
	wc := WorkerConfig{Inputs: []InputSource{
		InputSource{FileName: "testFile1.txt", URL: "http://example1.localdomain/test1.txt"},
		InputSource{FileName: "testFile2.jp2", URL: "http://example2.localdomain/test2.jp2"},
	}}

	inputs := wc.InputsAsMap()

	assert.Len(t, inputs, 2)
	assert.Equal(t, "http://example1.localdomain/test1.txt", inputs["testFile1.txt"])
	assert.Equal(t, "http://example2.localdomain/test2.jp2", inputs["testFile2.jp2"])
}
