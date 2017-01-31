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

package pzse

import (
	"errors"
	"testing"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

// PrintHelp prints out a basic helpfile to make things easier on direct users
func TestPrintHelp(t *testing.T) {
	w, _, _ := pzsvc.GetMockResponseWriter()
	PrintHelp(w)
}

func TestHandleFlist(t *testing.T) {

	outObj := OutStruct{}
	var err error

	flistTestFunc := func(dataID, fname, fType string) (string, error) {
		if dataID == "err" {
			return "", errors.New("error")
		}
		return dataID, nil
	}

	err = handleFList(pzsvc.Session{}, []string{"blah"}, []string{"../blah"}, flistTestFunc, "unspecified", "", &outObj, map[string]string{"": ""})
	if err == nil {
		t.Error(`TestHandleFlist: failed to catch attempted temp folder breakout`)
	}
	err = handleFList(pzsvc.Session{}, []string{"err"}, []string{"blah"}, flistTestFunc, "unspecified", "", &outObj, map[string]string{"": ""})
	if err == nil {
		t.Error(`TestHandleFlist: failed to catch file handle error`)
	}
	err = handleFList(pzsvc.Session{}, []string{""}, []string{"blah"}, flistTestFunc, "unspecified", "", &outObj, map[string]string{"": ""})
	if err == nil {
		t.Error(`TestHandleFlist: failed to catch file handle empty`)
	}
}
