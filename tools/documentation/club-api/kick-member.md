# Kick Member

Removes a member from the club.

## Endpoint
`PUT /v1/clubs/{id}/members/{memberID}/kick`

## Headers
- `Authorization: Bearer {firebase_token}`
- `UserID: {user_uuid}`

## Response
**200 OK**
```json
{
  "msg": "OK"
}
```

## Implementation Details
- Removes member from club members collection
- Unsubscribes from all club notification topics (general and admin)
- Sends notification to kicked member
- Validates admin/owner permissions

## Error Responses
- `400 Bad Request` - Invalid club ID or member ID
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Club or member not found
- `500 Internal Server Error` - Removal failed

## Related Files
- Implementation: `club/service/service.go:533` (`KickMember()`)
- Database operations: `club/service/member.go:64` (`DeleteMember()`)