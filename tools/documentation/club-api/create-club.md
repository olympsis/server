# Create Club

Creates a new club and assigns the creator as the owner.

## Endpoint
`POST /v1/clubs`

## Headers
- `Authorization: Bearer {firebase_token}`
- `UUID: {user_uuid}`

## Request Body
```json
{
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
  "visibility": "public|private",
  "blacklist": ["blocked_user_id"],
  "rules": ["Rule 1", "Rule 2"]
}
```

## Response
**201 Created**
```json
{
  "id": "club_object_id"
}
```

## Implementation Details
- Creates notification topics for the club (general and admin)
- Automatically adds creator as owner in club members collection
- Sets `isVerified: false` by default
- Creates timestamps automatically

## Error Responses
- `400 Bad Request` - Invalid request body or missing required fields
- `500 Internal Server Error` - Club creation failed

## Related Files
- Implementation: `club/service/service.go:159` (`CreateClub()`)
- Database operations: `club/service/club.go:10` (`InsertClub()`)
- Member operations: `club/service/member.go:11` (`InsertMember()`)