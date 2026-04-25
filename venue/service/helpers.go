package service

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// validateVenuesQuery checks that the request has a valid combination of query parameters.
// Valid query types:
//   - Location: requires longitude, latitude, and radius
//   - Transit: requires transit_system and transit_names (comma-separated)
//   - Bounding box: requires bbox (4 comma-separated floats: west,south,east,north)
//
// Sports is optional in all cases — if empty or "all", all sports are returned.
// At least one query type (location, transit, or bbox) must be provided.
func validateVenuesQuery(r *http.Request) error {
	query := r.URL.Query()

	longitudeStr := query.Get("longitude")
	latitudeStr := query.Get("latitude")
	radiusStr := query.Get("radius")
	transitSystem := query.Get("transit_system")
	transitNames := query.Get("transit_names")
	bbox := query.Get("bbox")

	hasLocation := longitudeStr != "" || latitudeStr != "" || radiusStr != ""
	hasTransit := transitSystem != "" || transitNames != ""
	hasBbox := bbox != ""

	// At least one query type must be provided
	if !hasLocation && !hasTransit && !hasBbox {
		return fmt.Errorf("a query type is required: provide location (longitude, latitude, radius), transit (transit_system, transit_names), or bbox")
	}

	// Validate location query — all three params are required together
	if hasLocation {
		if longitudeStr == "" || latitudeStr == "" || radiusStr == "" {
			return fmt.Errorf("location query requires longitude, latitude, and radius")
		}

		longitude, err := strconv.ParseFloat(longitudeStr, 64)
		if err != nil {
			return fmt.Errorf("invalid longitude value: %s", longitudeStr)
		}
		if longitude < -180 || longitude > 180 {
			return fmt.Errorf("longitude must be between -180 and 180")
		}

		latitude, err := strconv.ParseFloat(latitudeStr, 64)
		if err != nil {
			return fmt.Errorf("invalid latitude value: %s", latitudeStr)
		}
		if latitude < -90 || latitude > 90 {
			return fmt.Errorf("latitude must be between -90 and 90")
		}

		radius, err := strconv.ParseFloat(radiusStr, 64)
		if err != nil {
			return fmt.Errorf("invalid radius value: %s", radiusStr)
		}
		if radius <= 0 {
			return fmt.Errorf("radius must be greater than 0")
		}
	}

	// Validate transit query — both params are required together
	if hasTransit {
		if transitSystem == "" || transitNames == "" {
			return fmt.Errorf("transit query requires both transit_system and transit_names")
		}
	}

	// Validate bbox — must be 4 float values: west,south,east,north
	if hasBbox {
		coords := strings.Split(bbox, ",")
		if len(coords) != 4 {
			return fmt.Errorf("bbox requires 4 comma-separated values: west,south,east,north")
		}
		for i, coord := range coords {
			val, err := strconv.ParseFloat(strings.TrimSpace(coord), 64)
			if err != nil {
				return fmt.Errorf("invalid bbox coordinate at position %d: %s", i, coord)
			}
			// Positions 0,2 are longitudes; 1,3 are latitudes
			if i == 0 || i == 2 {
				if val < -180 || val > 180 {
					return fmt.Errorf("bbox longitude at position %d must be between -180 and 180", i)
				}
			} else {
				if val < -90 || val > 90 {
					return fmt.Errorf("bbox latitude at position %d must be between -90 and 90", i)
				}
			}
		}
	}

	return nil
}

func generateVenuesQuery(r *http.Request) bson.M {
	return bson.M{}
}
