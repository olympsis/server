package service

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Parse and validate location query parameters
func parseLocationQueryParams(r *http.Request) (*LocationQueryParams, error) {
	query := r.URL.Query()
	params := &LocationQueryParams{}

	// Parse and validate longitude
	longitudeStr := query.Get("longitude")
	if longitudeStr == "" {
		return nil, fmt.Errorf("missing required parameter: longitude")
	}
	longitude, err := strconv.ParseFloat(longitudeStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid longitude value: %s", longitudeStr)
	}
	// Validate longitude is within valid range (-180 to 180)
	if longitude < -180 || longitude > 180 {
		return nil, fmt.Errorf("longitude must be between -180 and 180")
	}
	params.Longitude = longitude

	// Parse and validate latitude
	latitudeStr := query.Get("latitude")
	if latitudeStr == "" {
		return nil, fmt.Errorf("missing required parameter: latitude")
	}
	latitude, err := strconv.ParseFloat(latitudeStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid latitude value: %s", latitudeStr)
	}
	// Validate latitude is within valid range (-90 to 90)
	if latitude < -90 || latitude > 90 {
		return nil, fmt.Errorf("latitude must be between -90 and 90")
	}
	params.Latitude = latitude

	// Parse and validate radius
	radiusStr := query.Get("radius")
	if radiusStr == "" {
		return nil, fmt.Errorf("missing required parameter: radius")
	}
	radius, err := strconv.ParseInt(radiusStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid radius value: %s", radiusStr)
	}
	if radius <= 0 {
		return nil, fmt.Errorf("radius must be greater than 0")
	}
	params.Radius = int(radius)

	// Parse sports
	sportsStr := query.Get("sports")
	if sportsStr == "" {
		return nil, fmt.Errorf("missing required parameter: sports")
	}
	params.Sports = strings.Split(sportsStr, ",")

	// Parse status
	statusStr := query.Get("status")
	if statusStr == "" {
		return nil, fmt.Errorf("missing required parameter: status")
	}
	// Validate status is one of the allowed values
	if statusStr != "completed" && statusStr != "upcoming" && statusStr != "live" {
		params.Status = "upcoming" // Default to upcoming if invalid
	} else {
		params.Status = statusStr
	}

	// Parse pagination parameters with defaults
	skipStr := query.Get("skip")
	skip := 0
	if skipStr != "" {
		skipVal, err := strconv.ParseInt(skipStr, 10, 32)
		if err == nil && skipVal >= 0 {
			skip = int(skipVal)
		}
	}
	params.Skip = skip

	limitStr := query.Get("limit")
	limit := 20 // Default limit
	if limitStr != "" {
		limitVal, err := strconv.ParseInt(limitStr, 10, 32)
		if err == nil && limitVal > 0 {
			limit = int(limitVal)
		}
	}
	params.Limit = limit

	return params, nil
}
