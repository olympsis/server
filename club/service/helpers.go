package service

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/olympsis/models"
)

// LocationQueryParams holds validated query parameters for the Location endpoint
type ClubsQueryParams struct {
	Location *models.GeoJSON
	Radius   float64
	City     string
	State    string
	Country  string
	Sports   []string
	Skip     int
	Limit    int
}

// Parse query parameters for the get clubs function
//
// - Parses location queries
// - Parses radius queries or set default to 16000 meters
// - Parses sports queries
// - Parses skip query for pagination
// - Parses limit query
//
// Returns: an array of Club objects
func parseQueryParams(r *http.Request) (*ClubsQueryParams, error) {
	query := r.URL.Query()
	params := &ClubsQueryParams{}

	// Parse and validate longitude
	locationStr := query.Get("location")

	cityStr := query.Get("city")
	stateStr := query.Get("state")
	countryStr := query.Get("country")

	params.City = cityStr
	params.State = stateStr
	params.Country = countryStr

	if locationStr == "" && countryStr == "" {
		return nil, fmt.Errorf("location (long,lat) or country name required")
	}

	// Parse location if provided
	if locationStr != "" {
		coords := strings.Split(locationStr, ",")
		if len(coords) != 2 {
			return nil, fmt.Errorf("invalid location format, expected 'long,lat'")
		}

		long, err := strconv.ParseFloat(coords[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid longitude value: %s", coords[0])
		}

		lat, err := strconv.ParseFloat(coords[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid latitude value: %s", coords[1])
		}

		params.Location = &models.GeoJSON{
			Type:        "Point",
			Coordinates: []float64{long, lat},
		}
	}

	// Parse and validate radius
	radiusStr := query.Get("radius")
	if radiusStr != "" {
		radius, err := strconv.ParseFloat(radiusStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid radius value: %s", radiusStr)
		}
		if radius <= 0 {
			return nil, fmt.Errorf("radius must be greater than 0")
		}

		tempRadius := float64(radius)
		params.Radius = tempRadius
	} else {
		tempRadius := float64(16000)
		params.Radius = tempRadius
	}

	// Parse sports
	sportsStr := query.Get("sports")
	if sportsStr == "" {
		params.Sports = []string{}
	} else {
		params.Sports = strings.Split(sportsStr, ",")
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

func generateNewReportNotification(id string, name string, repID string) models.PushNotification {
	return models.PushNotification{
		Title:    "New Report!",
		Body:     fmt.Sprintf("A member of %s created a report.", name),
		Type:     "push",
		Category: "groups",
		Data: map[string]interface{}{
			"type":      "new_report",
			"id":        id,
			"report_id": repID,
		},
	}
}

func generateNewApplicationNotification(id string, clubName string) models.PushNotification {
	return models.PushNotification{
		Title:    fmt.Sprintf("[%s] New Club Application", clubName),
		Body:     "A new club application was filed", //fmt.Sprintf("%s applied to your club", applicantName),
		Type:     "push",
		Category: "groups",
		Data: map[string]interface{}{
			"type":    models.NewClubApplication,
			"club_id": id,
		},
	}
}

func generateUpdateApplicationNotification(id string, name string) models.PushNotification {
	return models.PushNotification{
		Title:    "Club Application Update",
		Body:     fmt.Sprintf("Your application to %s was approved!", name),
		Type:     "push",
		Category: "groups",
		Data: map[string]interface{}{
			"type":    models.NewClubApplication,
			"club_id": id,
		},
	}
}
