# Leave Club

Allows a member to voluntarily leave the club.

## Endpoint
`PUT /v1/clubs/{id}/leave`

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
- Removes member record for the user
- Unsubscribes from all club notification topics (general and admin)
- Uses user UserID from header to identify member to remove
- Self-service endpoint - members can leave without admin approval

## Error Responses
- `400 Bad Request` - Invalid club ID
- `500 Internal Server Error` - Leave operation failed

## Related Files
- Implementation: `club/service/service.go:657` (`LeaveClub()`)
- Database operations: `club/service/member.go:64` (`DeleteMember()`)