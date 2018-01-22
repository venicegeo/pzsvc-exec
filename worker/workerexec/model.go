package workerexec

// workerOutputData populates and provides the format for pzsvc-exec's output
// Reimplementation of pzse.OutStruct
type workerOutputData struct {
	InFiles    map[string]string `json:"InFiles,omitempty"`
	OutFiles   map[string]string `json:"OutFiles,omitempty"`
	ProgStdOut string            `json:"ProgStdOut,omitempty"`
	ProgStdErr string            `json:"ProgStdErr,omitempty"`
	Errors     []string          `json:"Errors,omitempty"`
	HTTPStatus int               `json:"HTTPStatus,omitempty"`
}

func (d *workerOutputData) AddErrors(errors ...error) {
	for _, err := range errors {
		d.Errors = append(d.Errors, err.Error())
	}
}
