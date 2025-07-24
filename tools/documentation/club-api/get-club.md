# Get Club

Retrieves a specific club's detailed information.

## Endpoint
`GET /v1/clubs/{id}`

## Headers
None required (public endpoint)

## Response
**200 OK**
```json
{
  "id": "club_object_id",
  "name": "Club Name",
  "description": "Club description",
  "tags": ["tag1", "tag2"],
  "sports": ["basketball", "soccer"],
  "city": "City Name",
  "state": "State Name",
  "country": "Country Name",
  "location": {
    "type": "Point", 
    "coordinates": [longitude, latitude]
  },
  "logo": "logo_url",
  "banner": "banner_url",
  "visibility": "public",
  "members": [...],
  "isVerified": false,
  "createdAt": "timestamp"
}
```

## Implementation Details
- Uses aggregation pipeline to return enriched club data
- Public endpoint - no authentication required
- Returns complete club information including member details

## Error Responses
- `400 Bad Request` - Invalid club ID format
- `404 Not Found` - Club doesn't exist

## Related Files
- Implementation: `club/service/service.go:115` (`GetClub()`)
- Aggregation: `aggregations/clubs.go` (`AggregateClub()`)