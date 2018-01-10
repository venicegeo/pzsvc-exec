package main

import (
	"fmt"
	"os"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
	"github.com/venicegeo/pzsvc-exec/pzsvc-worker/config"

	cli "gopkg.in/urfave/cli.v1"
)

var cliApp *cli.App

func init() {
	cliApp = cli.NewApp()
	cliApp.Name = "pzsvc-worker"
	cliApp.Usage = "run a one-off piazza job in its own process"
	cliApp.Action = runCmd

	cliApp.Flags = []cli.Flag{
		cli.StringFlag{Name: "cliCmd", Usage: "command to run the Piazza job"},
		cli.StringFlag{Name: "piazzaBaseURL", Usage: "base URL for querying Piazza API", EnvVar: "PIAZZA_URL"},
		cli.StringFlag{Name: "piazzaAPIKey", Usage: "API key for use for communicating with Piazza"},
		cli.StringFlag{Name: "userID", Usage: "key authentication string"},
		cli.StringSliceFlag{Name: "input, i", Usage: "input source specification (as \"filename:URL\")"},
		cli.StringSliceFlag{Name: "output, o", Usage: "output file name (usable multiple times)"},
	}
}

func runCmd(ctx *cli.Context) error {
	cfg := config.WorkerConfig{
		Session:       pzsvc.Session{AppName: "pzsvc-worker", SessionID: "startup", LogRootDir: "pzsvc-exec"},
		CLICommand:    ctx.String("cliCmd"),
		PiazzaBaseURL: ctx.String("piazzaBaseURL"),
		PiazzaAPIKey:  ctx.String("piazzaAPIKey"),
		UserID:        ctx.String("userID"),
		Inputs:        []config.InputSource{},
		Outputs:       ctx.StringSlice("output"),
	}
	pzsvc.LogInfo(cfg.Session, "startup")

	if cfg.CLICommand == "" {
		return cli.NewExitError("CLI command is required", 1)
	}
	if cfg.PiazzaBaseURL == "" {
		return cli.NewExitError("Piazza base URL is required", 1)
	}
	if cfg.PiazzaAPIKey == "" {
		return cli.NewExitError("Piazza API key is required", 1)
	}
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

	pzsvc.LogInfo(cfg.Session, fmt.Sprintf("config validated: %s", cfg.Serialize()))

	err := mainWorkerProcess(cfg)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	return nil
}

func main() {
	cliApp.Run(os.Args)
}
