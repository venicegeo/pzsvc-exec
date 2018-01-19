package workerlog

import (
	"fmt"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/worker/config"
)

// Info is a wrapper around pzsvc.LogInfo that includes worker config details
func Info(cfg config.WorkerConfig, message string) {
	pzsvc.LogInfo(*cfg.Session, applyWorkerPrefix(cfg, message))
}

// Warn is a wrapper around pzsvc.LogWarn that includes worker config details
func Warn(cfg config.WorkerConfig, message string) {
	pzsvc.LogWarn(*cfg.Session, applyWorkerPrefix(cfg, message))
}

// Alert is a wrapper around pzsvc.LogAlert that includes worker config details
func Alert(cfg config.WorkerConfig, message string) {
	pzsvc.LogAlert(*cfg.Session, applyWorkerPrefix(cfg, message))
}

// SimpleErr is a wrapper around pzsvc.LogSimpleErr that includes worker config details
func SimpleErr(cfg config.WorkerConfig, message string, err error) {
	pzsvc.LogSimpleErr(*cfg.Session, applyWorkerPrefix(cfg, message), err)
}

func applyWorkerPrefix(cfg config.WorkerConfig, message string) string {
	return fmt.Sprintf("{Worker, jobID=%s} %s", cfg.JobID, message)
}
