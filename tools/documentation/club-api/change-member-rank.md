# Change Member Rank

Updates a member's role within the club.

## Endpoint
`PUT /v1/clubs/{id}/members/{memberID}/rank`

## Headers
- `Authorization: Bearer {firebase_token}`
- `UUID: {user_uuid}`

## Request Body
```json
{
  "role": "owner|admin|member"
}
```

## Response
**200 OK**
```json
{
  // Updated club object with member changes
}
```

## Implementation Details
- Validates member exists
- Updates role in members collection
- Manages notification topic subscriptions (adds to admin topic if promoted)
- Sends push notification to affected member
- Returns updated club data
- If user was member and promoted to admin/owner, adds them to admin topic

## Error Responses
- `400 Bad Request` - Invalid club ID or member ID
- `403 Forbidden` - Insufficient permissions to change roles
- `404 Not Found` - Member not found
- `500 Internal Server Error` - Update failed

## Related Files
- Implementation: `club/service/service.go:399` (`ChangeMemberRank()`)
- Database operations: `club/service/member.go:46` (`UpdateMember()`)