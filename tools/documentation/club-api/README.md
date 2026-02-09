# Club API Documentation

This directory contains comprehensive documentation for the Olympsis Club API endpoints, organized by functionality.

## Base URL Structure
All club endpoints follow the pattern: `/v1/clubs/{id}` where `{id}` is the club's ObjectID.

## Authentication
- Most endpoints require Firebase authentication token in the `Authorization` header
- User UserID is passed in the `UserID` header
- Admin/Owner permissions are validated internally based on club membership roles

---

## 📁 Documentation Files

### CRUD Operations
- [**create-club.md**](./create-club.md) - `POST /v1/clubs` - Create a new club
- [**get-club.md**](./get-club.md) - `GET /v1/clubs/{id}` - Get specific club details
- [**get-clubs.md**](./get-clubs.md) - `GET /v1/clubs` - Search and list clubs
- [**update-club.md**](./update-club.md) - `PUT /v1/clubs/{id}` - Update club information
- [**delete-club.md**](./delete-club.md) - `DELETE /v1/clubs/{id}` - Delete club

### Members Management
- [**change-member-rank.md**](./change-member-rank.md) - `PUT /v1/clubs/{id}/members/{memberID}/rank` - Change member role
- [**kick-member.md**](./kick-member.md) - `PUT /v1/clubs/{id}/members/{memberID}/kick` - Remove member
- [**leave-club.md**](./leave-club.md) - `PUT /v1/clubs/{id}/leave` - Leave club voluntarily

### Applications Management
- [**create-application.md**](./create-application.md) - `POST /v1/clubs/{id}/applications` - Apply to join club
- [**update-application.md**](./update-application.md) - `PUT /v1/clubs/{id}/applications/{applicationID}` - Accept/deny application

### Posts Management
- [**pin-post.md**](./pin-post.md) - `PUT /v1/clubs/{id}/post/{postID}` - Pin post to club feed
- [**unpin-post.md**](./unpin-post.md) - `PUT /v1/clubs/{id}/post` - Remove pinned post

---

## 🚧 Not Yet Implemented

### Finance Operations
- Create Wallet - `POST /v1/clubs/{id}/wallet`
- Update Wallet - `PUT /v1/clubs/{id}/wallet`
- Withdraw - `POST /v1/clubs/{id}/wallet/withdraw`

### Invitations System
- Make Invitation - `POST /v1/clubs/{id}/invitations`
- Update Invitation - `PUT /v1/clubs/{id}/invitations/{invitationID}`
- Delete Invitation - `DELETE /v1/clubs/{id}/invitations/{invitationID}`

### Additional Member Features
- Report Member - `POST /v1/clubs/{id}/members/{memberID}/report`
- Suspend Member - `PUT /v1/clubs/{id}/members/{memberID}/suspend`

### Additional Post Features
- Approve Post - `PUT /v1/clubs/{id}/posts/{postID}/approve`
- Deny Post - `PUT /v1/clubs/{id}/posts/{postID}/deny`

---

## 🏗️ Implementation Architecture

### Database Collections
- `ClubCol` - Club documents
- `ClubMembersCollection` - Club membership records
- `ClubApplicationCol` - Membership applications  
- `ClubInvitationCol` - Club invitations (not yet implemented)

### Service Layer Structure
- **API Layer:** `club/api.go` - HTTP handlers and routing
- **Service Layer:** `club/service/` - Business logic implementation
  - `service.go` - Main service functions and CRUD operations
  - `club.go` - Club-specific database operations
  - `member.go` - Member management database operations
  - `application.go` - Application handling logic
  - `management.go` - Administrative functions
  - `helpers.go` - Utility functions and query parsing

### Key Features
- **Notification Integration** - Topic-based push notifications
- **Role-based Permissions** - Owner > Admin > Member hierarchy
- **Geospatial Search** - Location-based club discovery
- **Blacklist Support** - User blocking functionality
- **Aggregation Pipelines** - Complex data queries with MongoDB

---

## 🔧 Common Error Responses

### HTTP Status Codes
- **400 Bad Request** - Invalid parameters or request format
- **401 Unauthorized** - Missing or invalid authentication
- **403 Forbidden** - Insufficient permissions for operation
- **404 Not Found** - Resource doesn't exist
- **500 Internal Server Error** - Database or server error

### Standard Error Format
```json
{
  "msg": "error description"
}
```

---

## 📚 Additional Resources

- [**CLAUDE.md**](../../CLAUDE.md) - Development setup and build commands
- [**Main API Documentation**](../club-api.md) - Original consolidated documentation
- **Source Code** - `/club/` directory for implementation details