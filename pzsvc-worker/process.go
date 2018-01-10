package main

import (
	"github.com/venicegeo/pzsvc-exec/pzsvc-worker/config"
	"github.com/venicegeo/pzsvc-exec/pzsvc-worker/input"
)

func mainWorkerProcess(cfg config.WorkerConfig) (err error) {
	err = input.FetchInputs(cfg.Session, cfg.Inputs)
	if err != nil {
		return
	}

	return
}
