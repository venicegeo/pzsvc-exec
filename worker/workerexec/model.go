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

package workerexec

// workerOutputData populates and provides the format for pzsvc-exec's output
// Reimplementation of pzse.OutStruct
type workerOutputData struct {
	InFiles    map[string]string `json:"InFiles,omitempty"`
	OutFiles   map[string]string `json:"OutFiles,omitempty"`
	ProgStdOut string            `json:"ProgStdOut,omitempty"`
	ProgStdErr string            `json:"ProgStdErr,omitempty"`
	Errors     []string          `json:"Errors,omitempty"`
	HTTPStatus int               `json:"HTTPStatus,omitempty"`
}

func (d *workerOutputData) AddErrors(errors ...error) {
	for _, err := range errors {
		d.Errors = append(d.Errors, err.Error())
	}
}
