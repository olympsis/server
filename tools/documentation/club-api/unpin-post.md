# Unpin Post

Removes the currently pinned post from the club.

## Endpoint
`PUT /v1/clubs/{id}/post`

## Headers
- `Authorization: Bearer {firebase_token}`
- `UUID: {user_uuid}`

## Response
**200 OK**
```json
{
  "msg": "OK"
}
```

## Implementation Details
- Removes `pinned_post_id` field from club document
- Does not require post ID since it removes any currently pinned post
- Requires admin/owner permissions

## Error Responses
- `400 Bad Request` - Invalid club ID
- `403 Forbidden` - Insufficient permissions  
- `500 Internal Server Error` - Unpin operation failed

## Related Files
- Implementation: `club/service/service.go:743` (`UnpinClubPost()`)
- Management functions: `club/service/management.go:58` (`UnpinPost()`)