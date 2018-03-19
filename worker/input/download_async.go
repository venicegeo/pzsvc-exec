package input

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/venicegeo/pzsvc-exec/worker/config"
)

var httpClient = http.Client{
	Timeout: 30 * time.Second,
}

type asyncDownloader interface {
	DownloadInputAsync(source config.InputSource) chan error
}

type defaultAsyncDownloader struct{}

var asyncDownloaderInstance asyncDownloader = defaultAsyncDownloader{}

func (dl defaultAsyncDownloader) DownloadInputAsync(source config.InputSource) chan error {
	errChan := make(chan error)

	go func() {
		var err error
		defer close(errChan)

		_, fStatErr := os.Stat(source.FileName)
		if fStatErr == nil {
			err = fmt.Errorf("File already exists: %v", source.FileName)
		} else if !os.IsNotExist(fStatErr) {
			err = fmt.Errorf("Error statting file: %v; %v", source.FileName, fStatErr)
		}
		if err != nil {
			errChan <- err
			return
		}

		resp, err := httpClient.Get(source.URL)
		if err == nil && resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("Unexpected HTTP status: %v", resp.StatusCode)
		}
		if err != nil {
			errChan <- err
			return
		}

		err = fileSaverInstance.CopyTo(source.FileName, resp.Body, 0777)

		if err != nil {
			errChan <- err
			return
		}
	}()

	return errChan
}
