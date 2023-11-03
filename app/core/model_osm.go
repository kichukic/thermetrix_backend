package core

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type OSMObject struct {
	PlaceId     uint     `json:"place_id"`
	License     string   `json:"license"`
	OSMType     string   `json:"osm_type"`
	OSMID       uint     `json:"osm_id"`
	BoundingBox []string `json:"boundingbox"`
	Lat         string   `json:"lat"`
	Lon         string   `json:"lon"`
	DisplayName string   `json:"display_name"`
	Class       string   `json:"class"`
	Type        string   `json:"type"`
	Importance  float64  `json:"importance"`
}

type OSMObjects []OSMObject

func GetOSMObjects(baseUrl, postalCode string) ([]OSMObject, error) {
	osmObjects := OSMObjects{}
	if baseUrl == "" {
		//baseUrl = "http://fleetserver.works4dev.de/nominatim/search?format=json&postalcode="
		baseUrl = "https://nominatim.openstreetmap.org/search?format=json&postalcode="
	}
	if postalCode != "" {
		codeLength := len(postalCode)
		if codeLength >= 5 {
			index := strings.Index(postalCode, " ")
			if index != codeLength-4 {
				postalCode = postalCode[0:codeLength-3] + " " + postalCode[codeLength-3:codeLength]
			}
		}
	}
	reqUrl := baseUrl + postalCode
	resp, err := http.Get(reqUrl)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &osmObjects)
	if err != nil {
		return nil, err
	}

	return osmObjects, nil
}

func GetOSMLatLon(baseUrl, postalCode string) (float64, float64) {
	lat, lon := 0.0, 0.0
	addresses, err := GetOSMObjects(baseUrl, postalCode)
	if err == nil && len(addresses) > 0 {
		lat, _ = strconv.ParseFloat(addresses[0].Lat, 64)
		lon, _ = strconv.ParseFloat(addresses[0].Lon, 64)
	}
	return lat, lon
}
