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

import (
	"os/exec"

	"github.com/venicegeo/pzsvc-exec/worker/config"
	"github.com/venicegeo/pzsvc-exec/worker/log"
)

type commandOutput struct {
	Stdout []byte
	Stderr []byte
	Error  error
}

type commandRunner interface {
	Run(cfg config.WorkerConfig, command string) commandOutput
}

type defaultCommandRunner struct{}

func (dcr defaultCommandRunner) Run(cfg config.WorkerConfig, command string) (out commandOutput) {
	var err error
	workerlog.Info(cfg, "runCommand: "+command)

	cmd := exec.Command("sh", "-c", command)
	out.Stdout, out.Error = cmd.Output()

	if out.Error != nil {
		if exitErr, ok := out.Error.(*exec.ExitError); ok {
			workerlog.SimpleErr(cfg, "failed executing command; stderr below", exitErr)
			workerlog.Alert(cfg, string(exitErr.Stderr))
			out.Stderr = exitErr.Stderr
		} else {
			workerlog.SimpleErr(cfg, "failed executing command; stderr not available", err)
		}
	} else {
		workerlog.Info(cfg, "runCommandOutput success")
	}
	return
}

var commandRunnerInstance commandRunner = defaultCommandRunner{}
