# Create Application

Submit an application to join the club.

## Endpoint
`POST /v1/clubs/{id}/applications`

## Headers
- `Authorization: Bearer {firebase_token}`
- `UUID: {user_uuid}`

## Response
**201 Created**
```json
{
  "id": "application_id"
}
```

## Flow Diagram
```mermaid
flowchart TD
    A[POST /v1/clubs/{id}/applications] --> B[Validate Club ID]
    B --> C{Club ID Valid?}
    C -->|No| D[Return 400 Bad Request]
    C -->|Yes| E[Convert Club ID to ObjectID]
    E --> F[Fetch Club Data]
    F --> G{Club Exists?}
    G -->|No| H[Return 404 Club Not Found]
    G -->|Yes| I[Check User in Blacklist]
    I --> J{User in Club's<br/>Blacklist?}
    J -->|Yes| K[Return 400 Forbidden<br/>User is blocked]
    J -->|No| L[Check Existing Application]
    L --> M{Pending Application<br/>Exists?}
    M -->|Yes| N[Return Existing Application<br/>201 Created]
    M -->|No| O[Create New Application]
    O --> P[Set Status: pending]
    P --> Q[Insert to Database]
    Q --> R[Send Notification to<br/>Club Admins]
    R --> S[Return Application ID<br/>201 Created]
```

## Implementation Details
- **Step 1:** Validates club ID format and existence
- **Step 2:** Checks if user is in club's blacklist array - if blocked, returns 400 error
- **Step 3:** Checks for existing pending applications to prevent duplicates
- **Step 4:** Creates application with "pending" status
- **Step 5:** Sends notification to club admins via admin topic
- **Step 6:** Returns existing application if one already exists

## Error Responses
- `400 Bad Request` - Invalid club ID or user is blocked
- `404 Not Found` - Club doesn't exist
- `500 Internal Server Error` - Application creation failed

## Related Files
- Implementation: `club/service/application.go:99` (`CreateApplication()`)
- Helpers: `club/service/helpers.go:131` (`generateNewApplicationNotification()`)