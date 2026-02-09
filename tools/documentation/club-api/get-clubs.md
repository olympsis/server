# Get Clubs

Retrieves a list of clubs based on location and filter criteria.

## Endpoint
`GET /v1/clubs`

## Headers
- `Authorization: Bearer {firebase_token}` (required)
- `UserID: {user_uuid}` (required)

## Query Parameters
- `location` (string): Comma-separated longitude,latitude (e.g., "-122.4194,37.7749")
- `radius` (float): Search radius in meters (default: 16000)
- `country` (string): Filter by country name
- `state` (string): Filter by state name (requires country)
- `city` (string): Filter by city name (requires state and country)
- `sports` (string): Comma-separated list of sports to filter by
- `skip` (int): Number of results to skip for pagination (default: 0)
- `limit` (int): Maximum number of results to return (default: 20)

## Response
**200 OK**
```json
{
  "totalClubs": 15,
  "clubs": [
    {
      "id": "club_object_id",
      "name": "Club Name",
      // ... club details
    }
  ]
}
```

## Implementation Details
- Supports both geospatial and text-based filtering
- Uses aggregation pipeline for complex queries
- Supports pagination with skip/limit
- Combines location radius search with sports/location filtering

## Error Responses
- `400 Bad Request` - Invalid query parameters
- `204 No Content` - No clubs found matching criteria

## Related Files
- Implementation: `club/service/service.go:45` (`GetClubs()`)
- Query parsing: `club/service/helpers.go:25` (`parseQueryParams()`)
- Aggregation: `aggregations/clubs.go` (`AggregateClubs()`)