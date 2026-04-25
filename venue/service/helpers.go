package service

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// validateCreateVenueRequest checks that all required fields are present
// on the venue and any inline units.
func validateCreateVenueRequest(req *models.VenueCreationRequest) error {
	v := &req.Venue

	if v.OwnerID == nil || v.OwnerID.IsZero() {
		return fmt.Errorf("owner_id is required")
	}
	if v.Name == nil || strings.TrimSpace(*v.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if v.Sports == nil || len(*v.Sports) == 0 {
		return fmt.Errorf("at least one sport is required")
	}
	if v.Location == nil || v.Location.Type == "" || v.Location.Coordinates == nil {
		return fmt.Errorf("a valid location with type and coordinates is required")
	}
	// Validate coordinates based on GeoJSON type
	switch v.Location.Type {
	case "Point":
		coords, ok := v.Location.Coordinates.([]interface{})
		if !ok || len(coords) < 2 {
			return fmt.Errorf("Point location requires [longitude, latitude] coordinates")
		}
	case "Polygon":
		coords, ok := v.Location.Coordinates.([]interface{})
		if !ok || len(coords) == 0 {
			return fmt.Errorf("Polygon location requires at least one ring of coordinates")
		}
	default:
		return fmt.Errorf("unsupported location type: %s", v.Location.Type)
	}
	if v.Address == nil || strings.TrimSpace(*v.Address) == "" {
		return fmt.Errorf("address is required")
	}
	if v.AdministrativeArea == nil || strings.TrimSpace(*v.AdministrativeArea) == "" {
		return fmt.Errorf("administrative_area is required")
	}
	if v.CountryCode == nil || strings.TrimSpace(*v.CountryCode) == "" {
		return fmt.Errorf("country_code is required")
	}
	if v.Timezone == nil || strings.TrimSpace(*v.Timezone) == "" {
		return fmt.Errorf("timezone is required")
	}

	// Validate inline units if provided
	for i, unit := range req.Units {
		if strings.TrimSpace(unit.Name) == "" {
			return fmt.Errorf("unit at index %d: name is required", i)
		}
		if strings.TrimSpace(unit.UnitType) == "" {
			return fmt.Errorf("unit at index %d: unit_type is required", i)
		}
		if len(unit.Sports) == 0 {
			return fmt.Errorf("unit at index %d: at least one sport is required", i)
		}
	}

	return nil
}

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

// generateVenuesQuery builds a MongoDB aggregation pipeline from the validated
// query parameters. The caller runs this pipeline against the venues collection.
//
// Query combination rules:
//   - Location + transit can combine
//   - Bbox + transit can combine
//   - Location always takes priority over bbox (bbox is ignored when location is present)
//
// Pipeline stages (in order):
//  1. $geoNear (location) OR $match with $geoWithin (bbox) — spatial filter
//  2. $match on sports — if a specific list was provided
//  3. $lookup + $match on transit lines — if transit params were provided
func generateVenuesQuery(r *http.Request) bson.A {
	query := r.URL.Query()

	longitudeStr := query.Get("longitude")
	latitudeStr := query.Get("latitude")
	radiusStr := query.Get("radius")
	transitSystem := query.Get("transit_system")
	transitNames := query.Get("transit_names")
	bboxStr := query.Get("bbox")
	sportsStr := query.Get("sports")

	hasLocation := longitudeStr != "" && latitudeStr != "" && radiusStr != ""
	hasTransit := transitSystem != "" && transitNames != ""
	hasBbox := bboxStr != ""

	pipeline := bson.A{}

	// --- Spatial stage: location takes priority over bbox ---
	if hasLocation {
		longitude, _ := strconv.ParseFloat(longitudeStr, 64)
		latitude, _ := strconv.ParseFloat(latitudeStr, 64)
		radius, _ := strconv.ParseFloat(radiusStr, 64)

		// $geoNear must be the first stage in the pipeline
		pipeline = append(pipeline, bson.M{
			"$geoNear": bson.M{
				"near": bson.M{
					"type":        "Point",
					"coordinates": bson.A{longitude, latitude},
				},
				"distanceField": "distance",
				"maxDistance":    radius,
				"spherical":     true,
			},
		})
	} else if hasBbox {
		coords := strings.Split(bboxStr, ",")
		west, _ := strconv.ParseFloat(strings.TrimSpace(coords[0]), 64)
		south, _ := strconv.ParseFloat(strings.TrimSpace(coords[1]), 64)
		east, _ := strconv.ParseFloat(strings.TrimSpace(coords[2]), 64)
		north, _ := strconv.ParseFloat(strings.TrimSpace(coords[3]), 64)

		pipeline = append(pipeline, bson.M{
			"$match": bson.M{
				"location": bson.M{
					"$geoWithin": bson.M{
						"$box": bson.A{
							bson.A{west, south},  // bottom-left
							bson.A{east, north},  // top-right
						},
					},
				},
			},
		})
	}

	// --- Sports filter ---
	if sportsStr != "" && sportsStr != "all" {
		sports := strings.Split(sportsStr, ",")
		pipeline = append(pipeline, bson.M{
			"$match": bson.M{
				"sports": bson.M{
					"$in": sports,
				},
			},
		})
	}

	// --- Transit filter: lookup transit lines and match by system + name ---
	if hasTransit {
		names := strings.Split(transitNames, ",")

		// Join the transit_lines ObjectID array against the transitLines collection
		pipeline = append(pipeline, bson.M{
			"$lookup": bson.M{
				"from":         "transitLines",
				"localField":   "transit_lines",
				"foreignField": "_id",
				"as":           "_transit_lines",
			},
		})

		// Keep only venues whose looked-up transit lines match the requested system + names
		pipeline = append(pipeline, bson.M{
			"$match": bson.M{
				"_transit_lines": bson.M{
					"$elemMatch": bson.M{
						"system": transitSystem,
						"name":   bson.M{"$in": names},
					},
				},
			},
		})

		// Drop the temporary lookup field so it doesn't leak into the response
		pipeline = append(pipeline, bson.M{
			"$project": bson.M{
				"_transit_lines": 0,
			},
		})
	}

	return pipeline
}
