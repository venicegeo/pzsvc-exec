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
	"fmt"

	"github.com/venicegeo/pzsvc-lib"
)

// CallPzsvcExec is a function designed to simplify calls to pzsvc-exec.
// Fill out the inpObj properly, and it'll go through the contact process,
// returning the OutFiles mapping (as that is generally what people are
// interested in, one way or the other)
func CallPzsvcExec(inpObj *InpStruct, algoURL string) (*OutStruct, error) {

	var respObj OutStruct
	byts, err := pzsvc.ReqByObjJSON("POST", algoURL, "", inpObj, &respObj)
	if err != nil {
		return nil, pzsvc.TraceErr(err)
	}

	if len(respObj.Errors) != 0 {
		return nil, fmt.Errorf(`pzsvc-exec errors: %s`, pzsvc.SliceToCommaSep(respObj.Errors))
	}

	fmt.Println("pzsvc-exec returned. Output: " + string(byts))

	return &respObj, nil
}
