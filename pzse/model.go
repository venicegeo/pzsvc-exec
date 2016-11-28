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

import "github.com/venicegeo/pzsvc-exec/pzsvc"

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
	CanUpload   bool
	CanDownlPz  bool
	CanDownlExt bool
}

// OutStruct populates and provides the format for pzsvc-exec's output
type OutStruct struct {
	InFiles    map[string]string `json:"InFiles,omitempty"`
	OutFiles   map[string]string `json:"OutFiles,omitempty"`
	ProgStdOut string            `json:"ProgStdOut,omitempty"`
	ProgStdErr string            `json:"ProgStdErr,omitempty"`
	Errors     []string          `json:"Errors,omitempty"`
	HTTPStatus int               `json:"HTTPStatus,omitempty"`
}

// ConfigParseOut is a handy struct to organize all of the outputs
// for pzse.ConfigParse() and prevent potential confusion.
type ConfigParseOut struct {
	AuthKey  string
	PortStr  string
	Version  string
	ProcPool pzsvc.Semaphore
}

// InpStruct is the format that pzsvc-exec demarshals input data into
type InpStruct struct {
	Command    string   `json:"cmd,omitempty"`
	InPzFiles  []string `json:"inPzFiles,omitempty"`    // slice: Pz dataIds
	InExtFiles []string `json:"inExtFiles,omitempty"`   // slice: external URL
	InPzNames  []string `json:"inPzNames,omitempty"`    // slice: name for the InPzFile of the same index
	InExtNames []string `json:"inExtNames,omitempty"`   // slice: name for the InExtFile of the same index
	OutTiffs   []string `json:"outTiffs,omitempty"`     // slice: filenames of GeoTIFFs to be ingested
	OutTxts    []string `json:"outTxts,omitempty"`      // slice: filenames of text files to be ingested
	OutGeoJs   []string `json:"outGeoJson,omitempty"`   // slice: filenames of GeoJSON files to be ingested
	ExtAuth    string   `json:"inExtAuthKey,omitempty"` // string: auth key for accessing external files
	PzAuth     string   `json:"pzAuthKey,omitempty"`    // string: auth key for accessing Piazza
	PzAddr     string   `json:"pzAddr,omitempty"`       // string: URL for the targeted Pz instance
}

type rangeFunc func(string, string, string) (string, error)
