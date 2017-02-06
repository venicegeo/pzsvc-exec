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
	"testing"

	"github.com/venicegeo/pzsvc-exec/pzsvc"
)

func TestCallPzsvcExec(t *testing.T) {

	var err error
	s := pzsvc.Session{}
	inpObj := InpStruct{}
	outStrs := []string{
		`}`,
		`{"Errors":["Yep.  It's an error"]}`,
		`{}`,
	}
	pzsvc.SetMockClient(outStrs, 250)

	_, err = CallPzsvcExec(s, &inpObj, "aaaa")
	if err == nil {
		t.Error(`TestCallPzsvcExec: passed on bad JSON.`)
	}
	_, err = CallPzsvcExec(s, &inpObj, "aaaa")
	if err == nil {
		t.Error(`TestCallPzsvcExec: passed on returned error.`)
	}
	_, err = CallPzsvcExec(s, &inpObj, "aaaa")
	if err != nil {
		t.Error(`TestCallPzsvcExec: failed on clean response.`)
	}
}
