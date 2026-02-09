# Update Club

Updates club information. Requires club admin/owner permissions.

## Endpoint
`PUT /v1/clubs/{id}`

## Headers
- `Authorization: Bearer {firebase_token}`
- `UserID: {user_uuid}`

## Request Body
All fields are optional:
```json
{
  "name": "Updated Club Name",
  "description": "Updated description",
  "sports": ["updated_sports"],
  "city": "Updated City",
  "state": "Updated State",
  "country": "Updated Country",
  "logo": "new_logo_url",
  "banner": "new_banner_url",
  "visibility": "private",
  "blacklist": ["user_ids"],
  "rules": ["Updated rules"]
}
```

## Response
**200 OK**
```json
{
  "msg": "OK"
}
```

## Implementation Details
- Only processes fields that are provided in the request
- Uses MongoDB `$set` operator for partial updates
- Validates club ID format before processing
- Requires admin/owner permissions (validated by middleware)

## Error Responses
- `400 Bad Request` - Invalid club ID or request body
- `403 Forbidden` - Insufficient permissions
- `500 Internal Server Error` - Database update failed

## Related Files
- Implementation: `club/service/service.go:254` (`ModifyClub()`)
- Database operations: `club/service/club.go:45` (`UpdateClub()`)