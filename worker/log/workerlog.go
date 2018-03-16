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

package workerlog

import (
	"fmt"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/worker/config"
)

// Info is a wrapper around pzsvc.LogInfo that includes worker config details
func Info(cfg config.WorkerConfig, message string) {
	if cfg.MuteLogs {
		return
	}
	pzsvc.LogInfo(*cfg.Session, applyWorkerPrefix(cfg, message))
}

// Warn is a wrapper around pzsvc.LogWarn that includes worker config details
func Warn(cfg config.WorkerConfig, message string) {
	if cfg.MuteLogs {
		return
	}
	pzsvc.LogWarn(*cfg.Session, applyWorkerPrefix(cfg, message))
}

// Alert is a wrapper around pzsvc.LogAlert that includes worker config details
func Alert(cfg config.WorkerConfig, message string) {
	if cfg.MuteLogs {
		return
	}
	pzsvc.LogAlert(*cfg.Session, applyWorkerPrefix(cfg, message))
}

// SimpleErr is a wrapper around pzsvc.LogSimpleErr that includes worker config details
func SimpleErr(cfg config.WorkerConfig, message string, err error) {
	if cfg.MuteLogs {
		return
	}
	pzsvc.LogSimpleErr(*cfg.Session, applyWorkerPrefix(cfg, message), err)
}

func applyWorkerPrefix(cfg config.WorkerConfig, message string) string {
	return fmt.Sprintf("{Worker, jobID=%s} %s", cfg.JobID, message)
}
