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

func runCommand(cfg config.WorkerConfig, command string) (out commandOutput) {
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
