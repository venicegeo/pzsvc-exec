package workerexec

import (
	"os/exec"
	"strings"

	"github.com/venicegeo/pzsvc-exec/worker/config"
	"github.com/venicegeo/pzsvc-exec/worker/ingest"
	"github.com/venicegeo/pzsvc-exec/worker/input"
	"github.com/venicegeo/pzsvc-exec/worker/log"
)

// WorkerExec runs the main worker exec subprocess
func WorkerExec(cfg config.WorkerConfig) (err error) {
	workerlog.Info(cfg, "Fetching inputs")
	err = input.FetchInputs(cfg, cfg.Inputs)
	if err != nil {
		return
	}
	workerlog.Info(cfg, "Inputs fetched")

	workerlog.Info(cfg, "Running version command")
	version, err := runCommandOutput(cfg, cfg.PzSEConfig.VersionCmd)
	if err != nil {
		return
	}
	version = strings.TrimSpace(version)
	workerlog.Info(cfg, "Retrieved algorithm version: "+version)

	command := strings.Join([]string{cfg.PzSEConfig.CliCmd, cfg.CLICommandExtra}, " ")
	workerlog.Info(cfg, "Running algorithm command")
	err = runCommandNoOutput(cfg, command)
	if err != nil {
		return
	}
	workerlog.Info(cfg, "Algorithm command successful")

	workerlog.Info(cfg, "Ingesting output files to Piazza")
	err = ingest.OutputFilesToPiazza(cfg, command, version)
	if err != nil {
		return
	}
	workerlog.Info(cfg, "Ingest successful")

	return
}

func runCommandNoOutput(cfg config.WorkerConfig, command string) error {
	workerlog.Info(cfg, "runCommandNoOutput: "+command)

	cmd := exec.Command("sh", "-c", command)

	stdoutStderrData, err := cmd.CombinedOutput()
	if err != nil {
		workerlog.SimpleErr(cfg, "failed executing command; stdout+stderr below", err)
		workerlog.Alert(cfg, string(stdoutStderrData))
		return err
	}
	workerlog.Info(cfg, "runCommandNoOutput success")
	return nil
}

func runCommandOutput(cfg config.WorkerConfig, command string) (string, error) {
	workerlog.Info(cfg, "runCommandOutput: "+command)

	cmd := exec.Command("sh", "-c", command)
	stdout, err := cmd.Output()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			workerlog.SimpleErr(cfg, "failed executing command; stderr below", exitErr)
			workerlog.Alert(cfg, string(exitErr.Stderr))
		} else {
			workerlog.SimpleErr(cfg, "failed executing command; stderr not available", err)
		}
	} else {
		workerlog.Info(cfg, "runCommandOutput success")
	}

	return string(stdout), err
}
