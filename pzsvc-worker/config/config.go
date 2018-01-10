package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/venicegeo/pzsvc-exec/pzse"
	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

// InputSource encapsulates the location and sourcing of a file
type InputSource struct {
	FileName string
	URL      string
}

// ParseInputSource takes a colon-separates input source string and turns it
// into an InputSource value
func ParseInputSource(sourceString string) (*InputSource, error) {
	parts := strings.SplitN(sourceString, ":", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("Invalid input source string: %s", sourceString)
	}
	return &InputSource{
		FileName: parts[0],
		URL:      parts[1],
	}, nil
}

// WorkerConfig encapsulates all configuration necessary for the  worker process
type WorkerConfig struct {
	Session         pzsvc.Session `json:"-"`
	PiazzaBaseURL   string
	PiazzaAPIKey    string
	CLICommandExtra string
	UserID          string
	Inputs          []InputSource
	Outputs         []string
	PzSEConfig      pzse.ConfigType
}

// ReadPzSEConfig reads the pzsvc-exec.config data from the given path
func (wc *WorkerConfig) ReadPzSEConfig(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &wc.PzSEConfig)
	return err
}

// Serialize turns the configuration into something readable (JSON)
func (wc WorkerConfig) Serialize() string {
	data, _ := json.Marshal(wc)
	return string(data)
}
