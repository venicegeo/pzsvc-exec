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
	"crypto/rand"
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

// LogFunc is the function used to add entries to the log
var (
	LogFunc func(string)
)

func baseLogFunc(logString string) {
	fmt.Println(logString)
}

// logMessage receives a string to put to the logs.  It formats it correctly
// and puts it in the right place.  This function exists partially in order
// to simplify the task of modifying log behavior in the future.  Note that
// logMessage will panic if no baseLogFunc has been set.  This is a feature,
// not a bug.  It helps you identify threads that have not been properly
// readied.  If logMessage panics in this way, the appropriate answer is
// to call ReadyLog before the first call to logMessage.
func logMessage(s Session, prefix, message string) {
	_, file, line, _ := runtime.Caller(2)
	if LogFunc == nil {
		LogFunc = baseLogFunc
	}
	if s.LogRootDir != "" {
		splits := strings.SplitAfter(file, s.LogRootDir)
		if len(splits) > 1 {
			file = s.LogRootDir + splits[len(splits)-1]
		}
	}
	outMsg := fmt.Sprintf("%s - [%s:%s %s %d] %s", prefix, s.AppName, s.SessionID, file, line, message)
	LogFunc(outMsg)
}

// LogInfo posts a logMessage call for standard, non-error messages.  The
// point is mostly to maintain uniformity of appearance and behavior.
func LogInfo(s Session, message string) {
	logMessage(s, "INFO", message)
}

// LogAlert posts a logMessage call for messages that suggest that someone
// may be attempting to breach the security of the program, or point to the
// possibility of a significant security vulnerability.  The point of this
// function is mostly to maintain uniformity of appearance and behavior.
func LogAlert(s Session, message string) {
	logMessage(s, "ALERT", message)
}

// LoggedError is a duplicate of the "error" interface.  Its real point is to
// indicate, when it is returned from a function, that the error it represents
// has already been entered intot he log and does not need to be entered again.
// The string contained in the LoggedError should be a relatively simple
// description of the error, suitable for returning to the caller of a REST
// interface.
type LoggedError error

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
		logMessage(s, "ERROR", outMsg)
		err.hasLogged = true
	} else {
		logMessage(s, "ERROR", "Meta-error.  Tried to log same message for a second time.")
	}
	return fmt.Errorf(err.Error())
}

// LogSimpleErr posts a logMessage call for simple error messages, and produces a pzsvc.Error
// from the result.  The point is mostly to maintain uniformity of appearance and behavior.
func LogSimpleErr(s Session, message string, err error) LoggedError {
	if err != nil {
		message += err.Error()
	}
	logMessage(s, "ERROR", message)
	return fmt.Errorf(message)
}

// Error here is intended to let pzsvc.Error objects serve the error interface, and,
// by extension, to let them be passed around as interfaces in palces that aren't
// importing pzsvc-lib and used in a reasonable manner
func (err Error) Error() string {
	if err.SimpleMsg != "" {
		return err.SimpleMsg
	}
	return err.LogMsg
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
