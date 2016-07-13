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
//	"bytes"
//	"encoding/json"
//	"fmt"
//	"io/ioutil"
//	"net/http"
//	"net/url"
	"strconv"
)

// EvTypeStruct contains all information necessary to register, modify, and/or
// describe an event type.
//TODO: full write-up and move this to model.go once complete
type EvTypeStruct struct {
	EventTypeID		string				`json:"eventTypeID,omitempty"`
	Mapping			map[string]string	`json:"mapping,omitempty"`
	Name			string				`json:"name,omitempty"`
}

// Pagination covers the pagination block of many Pz objects.
type Pagination struct {
	Count			int					`json:"count,omitempty"`
	Order			string				`json:"order,omitempty"`
	Page			string				`json:"page,omitempty"`
	PerPage			string				`json:"perPage,omitempty"`
	SortBy			string				`json:"sortBy,omitempty"`
}

// EvTypeRespStruct is the format for the response to the "Get Event Types" call.
type EvTypeRespStruct struct {
	Data			[]EvTypeStruct		`json:"data, omitempty"`
	Pag				*Pagination			`json:"pagination, omitempty"`
}

// TriggerStruct contains all information necessary to register, modify, and/or
// describe a trigger.
type TriggerStruct struct {
}

// ProdLineStruct contains all information necessary to describe a product line.
// It is primarily intended for output purposes.
type ProdLineStruct struct {
}

// FindEventType searches through all available event types in the
// given Piazza (Pz) instance
func FindEventType(evType EvTypeStruct, pzAddr, pzAuth string) (string, error) {
	for i := 0; ; i++{
		var evTypeColl EvTypeRespStruct
		_, err := RequestKnownJSON("GET", "", pzAddr + "/eventType?perPage=20;page=" + strconv.Itoa(i), pzAuth, &evTypeColl)
		if err != nil {
			return "", err
		}
		if (len(evTypeColl.Data) == 0) {
			return "", nil
		}
		for _, nextEvT := range evTypeColl.Data {
			if areEvTypesSame (evType, nextEvT) {
				return nextEvT.EventTypeID, nil
			}
		}
	}
}

// areEvTypesSame returns whether two event types are functionally identical, and
// does so in a fairly efficient manner.  It is inaccurate when the first eventType
// has been manually defined to contain a mapping to "".  (JSON conversions do not
// generate such mappings.)  It is inefficient when dealing with sets of event
// types that often share names but differ in mappings, and have large numbers of
// mappings. 
func areEvTypesSame (type1, type2 EvTypeStruct) (bool) {
	isSame := (type1.Name == type2.Name) && (len(type1.Mapping) == len(type2.Mapping))
	if !isSame {
		return false
	}
	for key, val := range type1.Mapping {
		isSame = isSame && type2.Mapping[key] == val
	}
	return isSame
}

// RegEventType registers a new event type to the given Pz instance
// based on the available information, and returns the Pz Id for that
// event type
func RegEventType(evType EvTypeStruct, pzAddr, authKey string) (string, error) {
	// - pretty standard "build and call" function.  Dry run it a few times,
	//   cheat off of similar functions elsewhere in the library, and you're
	//   good to go.
	return "", nil
}

// GetEventType takes an event type ID, then makes calls to Pz as necessary to
// find and return all pertinent information on that Event Type ID.
func GetEventType(evTypeID, pzAddr, authKey string) (*EvTypeStruct, error) {
	// - Slightly different format than RegEventType, but fundamentally the
	//   same sort of thing.  Is this necessary for v1?
	return nil, nil
}

// FindMyTriggers returns the IDs of the triggers this user has created.
func FindMyTriggers(pzAddr, authKey string) ([]string, error) {
	// not sure how well-supported this is right now.  Presumably support
	// for it is in the works.  Not actually necessary for v1.
	return nil, nil
}

// GetProductLineByTriggerID will, when given the necessary information,
// make calls into the given Pz instance as necessary to put together
// all pertinent data on the product line in question
func GetProductLineByTriggerID(triggerID, pzAddr, authKey string) (*ProdLineStruct, error) {
	// not necessary for v1.  Requires deterining what is actually necessary
	// for full description of product line.
	return nil, nil
}

// RegTrigger creates a new trigger in the given Pz instance, populated as
// indicated in the triggerData parameter.  Returns the triggerId.
func RegTrigger(triggerData TriggerStruct, pzAddr, authKey string) (string, error) {
	// in order to know what to do here, we need to know what sort of a structure
	// to build.  In order to do that, it would be handy to know what sort
	// of stuff we can get fed to an event, and/or 
	return "", nil
}

// BuildEventHandler finds or creates each fo the necessary pieces in the
// given Pz instance to have Pz respond to an incoming event by calling
// the correct function with the correct inputs.
func BuildEventHandler(handlerJSON, pzAddr, authKey string) (string, error) {
// demarshal JSON into appropriate struct.
// call FindEventType with data from struct.
// - if there, call FindTrigger
// - else, call RegEventType with data froms truct


/*
	what the user has going in...
	- They know what database they want things from.
	- They know what filters they want to apply.
	- They know some details about the service they want to call on it

	by the time it gets here...
	- The exact proccessing command(s) will have been established.

	things derive for ourselves...
	- the layer ID for geoserver ingest (added later)
	- the eventId
	- the triggerId (importance?)

*/
	return "", nil
}




// Further development:
//- trigger expiration?
