package input

import (
	"fmt"
	"io"
	"net/http"
	"time"
	"os"
	"strconv"

	"github.com/venicegeo/pzsvc-exec/worker/config"
)

func getClientTimeout() time.Duration {
	defaultTimeout := 180
	if envTimeout := os.Getenv("HTTP_TIMEOUT"); envTimeout != "" {
		defaultTimeout , _ = strconv.Atoi(envTimeout)
	}
	return (time.Duration(defaultTimeout) * time.Second)
}

func getClientRetries() int {
	defaultRetries := 1
	if envRetries := os.Getenv("HTTP_RETRIES"); envRetries != "" {
		defaultRetries , _ = strconv.Atoi(envRetries)
	}
	return defaultRetries
}

var httpClient = http.Client{
	Timeout: getClientTimeout(),
}

type asyncDownloader interface {
	DownloadInputAsync(source config.InputSource) chan error
}

type defaultAsyncDownloader struct{
	Retries int
}

var asyncDownloaderInstance asyncDownloader = defaultAsyncDownloader{
	Retries: getClientRetries(),
}

func (dl defaultAsyncDownloader) DownloadInputAsync(source config.InputSource) chan error {
	errChan := make(chan error)

	go func() {
		var err error
		var resp *http.Response
		defer close(errChan)

		targetFile, err := fileCheckerInstance.CheckAndOpen(source.FileName, 0777)
		if err != nil {
			errChan <- err
			return
		}
		defer targetFile.Close()

		for i := 0; i <= dl.Retries; i++ {
			resp, err = httpClient.Get(source.URL)
			if err == nil && resp.StatusCode != http.StatusOK {
				err = fmt.Errorf("unexpected status downloading input (%v)", resp.StatusCode)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to download URL %s on the %d attempt: %v. Timing out after %d retries.\n", source.URL, i+1, err, dl.Retries)
				continue
			}
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
