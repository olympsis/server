# Delete Club

Permanently deletes a club and all associated data. Requires owner permissions.

## Endpoint
`DELETE /v1/clubs/{id}`

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
- Deletes club document
- Removes all club members
- Deletes notification topics (both general and admin)
- Cannot be undone
- Requires owner permissions only

## Error Responses
- `400 Bad Request` - Invalid club ID
- `403 Forbidden` - Only club owner can delete
- `500 Internal Server Error` - Deletion failed

## Related Files
- Implementation: `club/service/service.go:330` (`DeleteClub()`)
- Database operations: `club/service/club.go:59` (`RemoveClub()`)
- Member cleanup: `club/service/member.go:72` (`DeleteMembers()`)