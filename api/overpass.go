package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/s3nkyh/arcticeroute/models"
)

func GetGlaciers(bbox string) ([]models.Glacier, error) {
	query := fmt.Sprintf(`
	[out:json][timeout:25];
	(
	  node["natural"="glacier"](%s);
	  way["natural"="glacier"](%s);
	  relation["natural"="glacier"](%s);
	);
	out center;
	`, bbox, bbox, bbox)

	encodedQuery := url.QueryEscape(query)
	apiURL := fmt.Sprintf("https://overpass-api.de/api/interpreter?data=%s", encodedQuery)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	return parseGlaciers(result), nil
}

func parseGlaciers(data map[string]interface{}) []models.Glacier {
	var glaciers []models.Glacier

	elements, ok := data["elements"].([]interface{})
	if !ok {
		return glaciers
	}

	for _, element := range elements {
		elem := element.(map[string]interface{})

		glacier := models.Glacier{
			ID:   int64(elem["id"].(float64)),
			Type: elem["type"].(string),
		}

		if tags, ok := elem["tags"].(map[string]interface{}); ok {
			if name, ok := tags["name"].(string); ok {
				glacier.Name = name
			} else {
				glacier.Name = "Unnamed Glacier"
			}
		} else {
			glacier.Name = "Unnamed Glacier"
		}

		if center, ok := elem["center"].(map[string]interface{}); ok {
			if lat, ok := center["lat"].(float64); ok {
				glacier.Latitude = lat
			}
			if lon, ok := center["lon"].(float64); ok {
				glacier.Longitude = lon
			}
		} else if lat, ok := elem["lat"].(float64); ok {
			glacier.Latitude = lat
			if lon, ok := elem["lon"].(float64); ok {
				glacier.Longitude = lon
			}
		}

		if glacier.Latitude != 0 && glacier.Longitude != 0 {
			glaciers = append(glaciers, glacier)
		}
	}

	return glaciers
}
