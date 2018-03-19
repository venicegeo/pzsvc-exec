package input

import (
	"fmt"
	"io"
	"net/http"
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

		targetFile, err := fileCheckerInstance.CheckAndOpen(source.FileName, 0777)
		if err != nil {
			errChan <- err
			return
		}
		defer targetFile.Close()

		resp, err := httpClient.Get(source.URL)
		if err == nil && resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("Unexpected HTTP status: %v", resp.StatusCode)
		}
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		_, err = io.Copy(targetFile, resp.Body)

		if err != nil {
			errChan <- err
			return
		}
	}()

	return errChan
}
