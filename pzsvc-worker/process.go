package main

import (
	"os/exec"
	"strings"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/pzsvc-worker/config"
	"github.com/venicegeo/pzsvc-exec/pzsvc-worker/input"
)

func mainWorkerProcess(cfg config.WorkerConfig) (err error) {
	err = input.FetchInputs(cfg.Session, cfg.Inputs)
	if err != nil {
		return
	}

	command := strings.Join([]string{cfg.PzSEConfig.CliCmd, cfg.CLICommandExtra}, " ")
	err = runCommand(cfg.Session, command)
	if err != nil {
		return
	}

	return
}

func runCommand(session pzsvc.Session, command string) error {
	pzsvc.LogInfo(session, "Executing command: "+command)

	cmd := exec.Command("sh", "-c", command)

	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		pzsvc.LogAlert(session, "failed executing command; stdout+stderr below; error: "+err.Error())
		pzsvc.LogAlert(session, string(stdoutStderr))
		return err
	}
	pzsvc.LogInfo(session, "completed running command")
	return nil
}
