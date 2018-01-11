package workerexec

import (
	"os/exec"
	"strings"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/pzsvc-worker/config"
	"github.com/venicegeo/pzsvc-exec/pzsvc-worker/ingest"
	"github.com/venicegeo/pzsvc-exec/pzsvc-worker/input"
)

// WorkerExec runs the main worker exec subprocess
func WorkerExec(cfg config.WorkerConfig) (err error) {
	err = input.FetchInputs(cfg.Session, cfg.Inputs)
	if err != nil {
		return
	}

	version, err := runCommandOutput(cfg.Session, cfg.PzSEConfig.VersionCmd)
	if err != nil {
		return
	}

	command := strings.Join([]string{cfg.PzSEConfig.CliCmd, cfg.CLICommandExtra}, " ")
	err = runCommandNoOutput(cfg.Session, command)
	if err != nil {
		return
	}

	err = ingest.OutputFilesToPiazza(cfg, command, version)
	if err != nil {
		return
	}

	return
}

func runCommandNoOutput(session pzsvc.Session, command string) error {
	pzsvc.LogInfo(session, "Executing command without output: "+command)

	cmd := exec.Command("sh", "-c", command)

	stdoutStderrData, err := cmd.CombinedOutput()
	if err != nil {
		pzsvc.LogAlert(session, "failed executing command; stdout+stderr below; error: "+err.Error())
		pzsvc.LogAlert(session, string(stdoutStderrData))
		return err
	}
	pzsvc.LogInfo(session, "completed running command")
	return nil
}

func runCommandOutput(session pzsvc.Session, command string) (string, error) {
	pzsvc.LogInfo(session, "Executing command with output: "+command)

	cmd := exec.Command("sh", "-c", command)
	stdout, err := cmd.Output()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			pzsvc.LogAlert(session, "failed executing command; stderr below; error: "+exitErr.Error())
			pzsvc.LogAlert(session, string(exitErr.Stderr))
		} else {
			pzsvc.LogAlert(session, "failed executing command; error: "+err.Error())
		}
	}

	return string(stdout), err
}
