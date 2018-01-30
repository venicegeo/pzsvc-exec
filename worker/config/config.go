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

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

// InputSource encapsulates the location and sourcing of a file
type InputSource struct {
	FileName string
	URL      string
}

// ParseInputSource takes a colon-separates input source string and turns it
// into an InputSource value
func ParseInputSource(sourceString string) (*InputSource, error) {
	parts := strings.SplitN(sourceString, ":", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("Invalid input source string: %s", sourceString)
	}
	return &InputSource{
		FileName: parts[0],
		URL:      parts[1],
	}, nil
}

// WorkerConfig encapsulates all configuration necessary for the  worker process
type WorkerConfig struct {
	Session         *pzsvc.Session `json:"-"`
	PiazzaBaseURL   string
	PiazzaAPIKey    string
	PiazzaServiceID string
	CLICommandExtra string
	UserID          string
	JobID           string
	Inputs          []InputSource
	Outputs         []string
	PzSEConfig      pzsvc.Config
}

// ReadPzSEConfig reads the pzsvc-exec.config data from the given path
func (wc *WorkerConfig) ReadPzSEConfig(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &wc.PzSEConfig)
	return err
}

// Serialize turns the configuration into something readable (JSON)
func (wc WorkerConfig) Serialize() string {
	data, _ := json.Marshal(wc)
	return string(data)
}

// InputsAsMap returns a string:string map representing the worker inputs
func (wc WorkerConfig) InputsAsMap() map[string]string {
	converted := map[string]string{}
	for _, input := range wc.Inputs {
		converted[input.FileName] = input.URL
	}
	return converted
}
