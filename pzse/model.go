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

import "github.com/venicegeo/pzsvc-lib"

// ConfigType represents and contains the information from a pzsvc-exec config file.
type ConfigType struct {
	CliCmd      string
	VersionCmd  string
	VersionStr  string
	PzAddr      string
	AuthEnVar   string
	SvcName     string
	URL         string
	Port        int
	Description string
	Attributes  map[string]string
	NumProcs    int
}

// OutStruct populates and provides the format for pzsvc-exec's output
type OutStruct struct {
	InFiles    map[string]string
	OutFiles   map[string]string
	ProgStdOut string
	ProgStdErr string
	Errors     []string
	HTTPStatus int
}

// ConfigParseOut is a handy struct to organize all of the outputs
// for pzse.ConfigParse() and prevent potential confusion.
type ConfigParseOut struct {
	AuthKey  string
	PortStr  string
	Version  string
	CanFile  bool
	ProcPool pzsvc.Semaphore
}

// InpStruct is the format that pzsvc-exec demarshals input data into
type InpStruct struct {
	Command    string   `json:"cmd"`
	InPzFiles  []string `json:"inPzFiles"`    // slice: Pz dataIds
	InExtFiles []string `json:"inExtFiles"`   // slice: external URL
	InPzNames  []string `json:"inPzNames"`    // slice: name for the InPzFile of the same index
	InExtNames []string `json:"inExtNames"`   // slice: name for the InExtFile of the same index
	OutTiffs   []string `json:"outTiffs"`     // slice: filenames of GeoTIFFs to be ingested
	OutTxts    []string `json:"outTxts"`      // slice: filenames of text files to be ingested
	OutGeoJs   []string `json:"outGeoJson"`   // slice: filenames of GeoJSON files to be ingested
	ExtAuth    string   `json:"inExtAuthKey"` // string: auth key for accessing external files
	PzAuth     string   `json:"pzAuthKey"`    // string: auth key for accessing Piazza
	PzAddr     string   `json:"pzAddr"`       // string: URL for the targeted Pz instance
}

type rangeFunc func(string, string, string) (string, error)
