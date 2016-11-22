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
	"strconv"
)

// LogFunc is the function used to add entries to the log
var LogFunc func(string)

func baseLogFunc(logString string) {
	fmt.Println(logString)
}

// LogMessage receives a string to put to the logs.  It formats it correctly
// and puts it in the right place.  This function exists partially in order
// to simplify the task of modifying log behavior in the future.
func logMessage(message, fileName string, lineNo int, isError bool) {
	if LogFunc == nil {
		LogFunc = baseLogFunc
	}
	var outBody string
	if isError {
		outBody = "ERROR - "
	} else {
		outBody = "INFO - "
	}
	if fileName != "" {
		outBody += ("[" + fileName + " " + strconv.Itoa(lineNo) + "] ")
	}
	outBody += message

}

// LogInfo posts a logMessage call for standard, non-error messages.  The
// point is mostly to maintain uniformity of appearance and behavior.
func LogInfo(message string) {
	_, file, line, _ := runtime.Caller(1)
	logMessage(message, file, line, false)
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
func (err *Error) Log(msgAdd string) LoggedError {
	_, file, line, _ := runtime.Caller(1)
	if !err.hasLogged {
		if msgAdd != "" {
			err.LogMsg = msgAdd + ": " + err.LogMsg
		}
		outMsg := err.LogMsg
		_, file, line, _ := runtime.Caller(1)
		if err.request != "" || err.response != "" {
			outMsg = err.GenExtendedMsg()
		}
		logMessage(outMsg, file, line, true)
		err.hasLogged = true
	} else {
		logMessage("Meta-error.  Tried to log same message for a second time.", file, line, false)
	}
	return fmt.Errorf(err.Error())
}

// LogSimpleErr posts a logMessage call for simple error messages, and produces a pzsvc.Error
// from the result.  The point is mostly to maintain uniformity of appearance and behavior.
func LogSimpleErr(message string, err error) LoggedError {
	message += err.Error()
	_, file, line, _ := runtime.Caller(1)
	logMessage(message, file, line, true)
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
