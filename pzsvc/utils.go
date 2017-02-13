// Copyright 2016, RadiantBlue Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pzsvc

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogFunc is the function used to add entries to the log

func baseLogFunc(logString string) {
	fmt.Println(logString)
}

var (
	// LogFunc is the logging function in use by this instance.
	// It exists as a way to easily control where your logs go.
	LogFunc = baseLogFunc
)

// LoggedError is a duplicate of the "error" interface.  Its real point is to
// indicate, when it is returned from a function, that the error it represents
// has already been entered into the log and does not need to be entered again.
// The string contained in the LoggedError should be a relatively simple
// description of the error, suitable for returning to the caller of a REST
// interface.
type LoggedError error

// various constants representing the levels of severity for a given audit message
const (
	FATAL    = 0
	CRITICAL = 2
	ERROR    = 3
	WARN     = 4
	NOTICE   = 5
	INFO     = 6
	DEBUG    = 7
)

// logMessage receives a string to put to the logs.  It formats it correctly
// and puts it in the right place.  This function exists partially in order
// to simplify the task of modifying log behavior in the future.
func logMessage(s Session, severity int, msg string) {
	funcPtr, file, line, _ := runtime.Caller(2)
	fname := runtime.FuncForPC(funcPtr).Name()
	if s.LogRootDir != "" {
		splits := strings.SplitAfter(file, s.LogRootDir)
		if len(splits) > 1 {
			file = s.LogRootDir + splits[len(splits)-1]
		}
	}
	time := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")

	hostName, _ := os.Hostname()
	outMsg := fmt.Sprintf(`<%d>1 %s %s %s - ID%d [pzsource@48851 file="%s" line="%d" function="%s"] %s`,
		8+severity, time, hostName, s.AppName, os.Getpid(), file, line, fname, msg)
	LogFunc(outMsg)
}

// LogSimpleErr posts a logMessage call for simple error messages, and produces a pzsvc.Error
// from the result.  The point is mostly to maintain uniformity of appearance and behavior.
func LogSimpleErr(s Session, message string, err error) LoggedError {
	if err != nil {
		message += err.Error()
	}
	logMessage(s, 3, message)
	return fmt.Errorf(message)
}

// LogInfo posts a logMessage call for standard, non-error messages.  The
// point is mostly to maintain uniformity of appearance and behavior.
func LogInfo(s Session, message string) {
	logMessage(s, 6, message)
}

// LogWarn posts a logMessage call for messages that suggest that something
// may be going wrong, especially in the case of expected user error, but
// that the program can make reasonable assumptions as to what was actually
// intended and carry on.  The point of this function is mostly to maintain
// uniformity of appearance and behavior.
func LogWarn(s Session, message string) {
	logMessage(s, 4, message)
}

// LogAlert posts a logMessage call for messages that suggest that someone
// may be attempting to breach the security of the program, or point to the
// possibility of a significant security vulnerability.  The point of this
// function is mostly to maintain uniformity of appearance and behavior.
func LogAlert(s Session, message string) {
	logMessage(s, 5, "ALERT:"+message)
}

// LogAudit posts a logMessage call for messages that are generated to
// conform to Audit requirements.  This function is intended to maintain
// uniformity of appearance and behavior, and also to ease maintainability
// when routing requirements change.
func LogAudit(s Session, actor, action, actee, msg string, severity int) {
	if s.LogAudit {
		outMsg := fmt.Sprintf(`[pzaudit@48851 actor="%s" action="%s" actee="%s"] %s`, actor, action, actee, msg)
		logMessage(s, severity, outMsg)
	}
}

// LogAuditResponse is LogAudit for those cases where it needs to include an HTTP response
// body, and that body is not beign conveniently read and outputted by some other function.
// It reads the response, logs the result, and replaces the consumed response body with a
// fresh one made from the read buffer, so that it doesn't interfere with any other function
// that woudl wish to access the body.
func LogAuditResponse(s Session, actor, action, actee string, resp *http.Response, severity int) {
	if s.LogAudit {
		bbuff, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(bbuff))
		trimPld := strings.Replace(string(bbuff), "\n", "", -1)
		LogAudit(s, actor, action, actee, trimPld, severity)
	}
}

// Error is intended as a somewhat more full-featured way of handlign the
// error niche
type Error struct {
	hasLogged  bool   // whether or not this Error has been logged
	LogMsg     string // message to enter into logs
	SimpleMsg  string // simplified message to return to user via rest endpoint
	request    string // http request body associated with the error (if any)
	response   string // http response body assocaited with the error (if any)
	url        string // url associated with the error (if any)
	httpStatus int    // http status associated with the error (if any)
}

// OverwriteRequest exists because some requests contain auth information.  For security
// reasons, that needs to be stripped out before logging, but the logic that knows
// it needs to be removed only exists outside of the pzsvc library.
func (err *Error) OverwriteRequest(inReq string) {
	err.request = inReq
}

// GenExtendedMsg is used to generate extended log messages from Error objects
// for the cases where that's appropriate
func (err Error) GenExtendedMsg() string {
	lineBreak := "\n/**************************************/\n"
	outBody := "Http Error: " + err.LogMsg + lineBreak
	if err.url != "" {
		outBody += "\nURL: " + err.url + "\n"
	}
	if err.request != "" {
		outBody += "\nRequest: " + err.request + "\n"
	}
	if err.response != "" {
		outBody += "\nResponse: " + err.response + "\n"
	}
	if http.StatusText(err.httpStatus) != "" {
		outBody += "\nHTTP Status: " + http.StatusText(err.httpStatus) + "\n"
	}
	outBody += lineBreak
	return outBody
}

// Log is intended as the base way to generate logging information for an Error
// object.  It constructs an extended error if necessary, gathers the filename
// and line number data, and sends it to logMessage for formatting and output.
// It also ensures that any given error will only be logged once, and will be
// logged at the lowest level that calls for it.  In particular, the general
// expectation is that the message will be generated at a relatively low level,
// and then logged with additional context at some higher position.  Given our
// general level of complexity, that strikes a decent balance between providing
// enough detail to figure out the cause of an error and keepign thigns simple
// enough to readily understand.
func (err *Error) Log(s Session, msgAdd string) LoggedError {
	if !err.hasLogged {
		if msgAdd != "" {
			err.LogMsg = msgAdd + ": " + err.LogMsg
		}
		outMsg := err.LogMsg
		if err.request != "" || err.response != "" {
			outMsg = err.GenExtendedMsg()
		}
		logMessage(s, 3, outMsg)
		err.hasLogged = true
	} else {
		logMessage(s, 3, "Meta-error.  Tried to log same message for a second time.")
	}
	if err.SimpleMsg != "" {
		return fmt.Errorf(err.SimpleMsg)
	}
	return fmt.Errorf(err.LogMsg)

}

// SliceToCommaSep takes a string slice, and turns it into a comma-separated
// list of strings, suitable for JSON.
func SliceToCommaSep(inSlice []string) string {
	sliLen := len(inSlice)
	if sliLen == 0 {
		return ""
	}
	accum := inSlice[0]
	for i := 1; i < sliLen; i++ {
		accum = accum + "," + inSlice[i]
	}
	return accum
}

// PsuUUID makes a psuedo-UUID.  It may not achieve cryptographic levels of
// randomness, and it won't respond properly to standard ways of pulling data
// out of UUIDs, but it works just fine at generating effectively unique IDs
// for practical purposes.
func PsuUUID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
