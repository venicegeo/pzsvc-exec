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

type rangeFunc func(string, string, string) (string, error)
