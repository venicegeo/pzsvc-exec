package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/worker/config"
	"github.com/venicegeo/pzsvc-exec/worker/log"
	"github.com/venicegeo/pzsvc-exec/worker/workerexec"

	cli "gopkg.in/urfave/cli.v1"
)

var cliApp *cli.App

func init() {
	cliApp = cli.NewApp()
	cliApp.Name = "pzsvc-worker"
	cliApp.Usage = "run a one-off piazza job in its own process"
	cliApp.Action = runCmd

	cliApp.Flags = []cli.Flag{
		cli.StringFlag{Name: "config", Usage: "JSON pzsvc-exec configuration file (required)"},
		cli.StringFlag{Name: "cliExtra", Usage: "supplemental command arguments to run the Piazza job"},
		cli.StringFlag{Name: "piazzaBaseURL", Usage: "base URL for querying Piazza API (required if not PZ_ADDR)"},
		cli.StringFlag{Name: "piazzaAPIKey", Usage: "API key for use for communicating with Piazza (required if not in vcap)"},
		cli.StringFlag{Name: "userID", Usage: "key authentication string (required)"},
		cli.StringFlag{Name: "serviceID", Usage: "piazza service ID (algorithm name) (required)"},
		cli.StringFlag{Name: "jobID", Usage: "job ID for this run, used for logging"},
		cli.StringSliceFlag{Name: "input, i", Usage: "input source specification (as \"filename:URL\")"},
		cli.StringSliceFlag{Name: "output, o", Usage: "output file name (usable multiple times; at least one required)"},
	}
}

func runCmd(ctx *cli.Context) error {
	cfg := config.WorkerConfig{
		Session:         &pzsvc.Session{AppName: "pzsvc-worker", SessionID: "startup", LogRootDir: "pzsvc-exec"},
		CLICommandExtra: ctx.String("cliExtra"),
		PiazzaBaseURL:   ctx.String("piazzaBaseURL"),
		PiazzaAPIKey:    ctx.String("piazzaAPIKey"),
		PiazzaServiceID: ctx.String("serviceID"),
		UserID:          ctx.String("userID"),
		JobID:           ctx.String("jobID"),
		Inputs:          []config.InputSource{},
		Outputs:         ctx.StringSlice("output"),
		PzSEConfig:      pzsvc.Config{},
	}
	workerlog.Info(cfg, "startup")

	if ctx.String("config") == "" {
		return cli.NewExitError("pzsvc-exec config file is required", 1)
	}
	if err := cfg.ReadPzSEConfig(ctx.String("config")); err != nil {
		return cli.NewExitError(err, 1)
	}

	if cfg.PiazzaServiceID == "" {
		return cli.NewExitError("Service ID is required", 1)
	}

	if cfg.PiazzaBaseURL == "" {
		cfg.PiazzaBaseURL = os.Getenv(cfg.PzSEConfig.PzAddrEnVar)
	}
	if cfg.PiazzaBaseURL == "" {
		return cli.NewExitError("Piazza base URL is required", 1)
	}
	cfg.Session.PzAddr = cfg.PiazzaBaseURL

	if cfg.PiazzaAPIKey == "" {
		cfg.PiazzaAPIKey = os.Getenv((cfg.PzSEConfig.APIKeyEnVar))
	}
	if cfg.PiazzaAPIKey == "" {
		return cli.NewExitError("Piazza API key is required", 1)
	}
	cfg.Session.PzAuth = "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.PiazzaAPIKey+":"))

	if len(cfg.Outputs) == 0 {
		return cli.NewExitError("1 or more output files are required", 1)
	}

	for _, sourceString := range ctx.StringSlice("input") {
		inFile, err := config.ParseInputSource(sourceString)
		if err != nil {
			return cli.NewExitError(err, 1)
		}
		cfg.Inputs = append(cfg.Inputs, *inFile)
	}

	workerlog.Info(cfg, fmt.Sprintf("config validated: %s", cfg.Serialize()))

	workerlog.Info(cfg, "Starting actual worker execution")
	err := workerexec.WorkerExec(cfg)
	if err != nil {
		workerlog.SimpleErr(cfg, "execution error, quitting with status 1", err)
		return cli.NewExitError(err, 1)
	}

	workerlog.Info(cfg, "worker done, exiting")
	return nil
}

func main() {
	cliApp.Run(os.Args)
}
