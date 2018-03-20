package poll

import (
	"encoding/json"
	"errors"
	"os"
)

func getVCAPApplicationID() (string, error) {
	// Get the application name
	vcapJSONContainer := make(map[string]interface{})
	err := json.Unmarshal([]byte(os.Getenv("VCAP_APPLICATION")), &vcapJSONContainer)
	if err != nil {
		return "", errors.New("Error in reading VCAP Application properties: " + err.Error())
	}
	appID, ok := vcapJSONContainer["application_id"].(string)
	if !ok {
		return "", errors.New("Cannot Read Application Name from VCAP Application properties: string type assertion failed")
	}
	return appID, nil
}
