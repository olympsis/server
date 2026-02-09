# Pin Post

Pins a specific post to the top of the club's feed.

## Endpoint
`PUT /v1/clubs/{id}/post/{postID}`

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
- Updates club document with `pinned_post_id` field
- Validates both club ID and post ID formats
- Only one post can be pinned at a time (overwrites previous)
- Requires admin/owner permissions

## Error Responses
- `400 Bad Request` - Invalid club ID or post ID
- `403 Forbidden` - Insufficient permissions
- `500 Internal Server Error` - Pin operation failed

## Related Files
- Implementation: `club/service/service.go:712` (`PinClubPost()`)
- Management functions: `club/service/management.go:30` (`PinPost()`)