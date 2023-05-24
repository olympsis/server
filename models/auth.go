package models

/*
Auth User
  - User data for auth database
  - Contains user identifiable data
*/
type AuthUser struct {
	UUID        string `json:"uuid" bson:"uuid"`
	FirstName   string `json:"first_name" bson:"first_name"`
	LastName    string `json:"last_name" bson:"last_name"`
	Email       string `json:"email" bson:"email"`
	Token       string `json:"token" bson:"token"`
	AccessToken string `json:"access_token" bson:"access_token"`
	Provider    string `json:"provider" bson:"provider"`
	CreatedAt   int64  `json:"created_at" bson:"created_at"`
}

/*
Sign In Request
  - Request comming from client
  - Contains user identifiable data
*/
type AuthRequest struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email,omitempty"`
	Code      string `json:"code"`
	Provider  string `json:"provider"`
}

/*
Log in Response
  - User identifiable data for client
*/
type AuthResponse struct {
	UUID      string `json:"uuid"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Token     string `json:"token"`
}
