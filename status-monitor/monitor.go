package statmon

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

type statusReport struct {
	Success   bool
	Message   string
	Timestamp time.Time
}

// StatusMonitor contains functionality for writing non-chatty logs
type StatusMonitor struct {
	interval       time.Duration
	successBuffer  []statusReport
	loggers        []*log.Logger
	reportChan     chan statusReport
	intervalTicker *time.Ticker
}

// New creates a new status monitor and starts its background goroutines
func New(prefix string, aggregateInterval time.Duration, outputFiles ...string) (*StatusMonitor, error) {
	monitor := StatusMonitor{
		successBuffer:  []statusReport{},
		loggers:        []*log.Logger{},
		reportChan:     make(chan statusReport),
		intervalTicker: time.NewTicker(aggregateInterval),
	}

	fOpenErrs := []interface{}{"Status log file open errors:"}
	for _, outputFile := range outputFiles {
		f, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0x666)
		if err != nil {
			fOpenErrs = append(fOpenErrs, err.Error())
			continue
		}
		logger := log.New(f, prefix, log.Ldate|log.Ltime)
		monitor.loggers = append(monitor.loggers, logger)
	}

	if len(monitor.loggers) == 0 {
		return nil, errors.New(fmt.Sprint(fOpenErrs...))
	}

	monitor.printAll(fOpenErrs...)
	monitor.printAll("Starting status logging, aggregate interval: ", aggregateInterval.Seconds(), "seconds")

	go monitor.asyncHandleInterval()
	go monitor.asyncHandleReports()

	return &monitor, nil
}

func (m *StatusMonitor) Success() {
	report := statusReport{Success: true, Timestamp: time.Now()}
	m.reportChan <- report
}

func (m *StatusMonitor) Failure(message string) {
	report := statusReport{Success: false, Message: message, Timestamp: time.Now()}
	m.reportChan <- report
}

func (m *StatusMonitor) Stop() {
	close(m.reportChan)
	m.intervalTicker.Stop()
}

func (m *StatusMonitor) printAll(data ...interface{}) {
	for _, logger := range m.loggers {
		logger.Print(data...)
	}
}

func (m *StatusMonitor) printfAll(format string, data ...interface{}) {
	for _, logger := range m.loggers {
		logger.Printf(format, data...)
	}
}

func (m *StatusMonitor) asyncHandleReports() {
	for report := range m.reportChan {
		if report.Success {
			m.successBuffer = append(m.successBuffer, report)
			continue
		}

		if len(m.successBuffer) > 0 {
			m.printfAll("%d successes queued up before failure", len(m.successBuffer))
			m.successBuffer = []statusReport{}
		}

		m.printAll("Failure:", report.Message)
	}
}

func (m *StatusMonitor) asyncHandleInterval() {
	for _ = range m.intervalTicker.C {
		if len(m.successBuffer) > 0 {
			m.printAll("Interval aggregate: No successes queued up")
			continue
		}

		oldestSuccessTimestamp := m.successBuffer[0].Timestamp
		m.printfAll("Interval aggregate: %v successes queued up; oldest was at: %v",
			len(m.successBuffer), oldestSuccessTimestamp.Format(time.ANSIC))
		m.successBuffer = []statusReport{}
	}
}
