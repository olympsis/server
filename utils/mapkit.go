package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/sideshow/apns2/token"
)

// MapKitConfig holds the credentials needed to generate MapKit JWTs.
type MapKitConfig struct {
	KeyFilePath string // Path to the .p8 private key file
	KeyID       string // MapKit key ID
	TeamID      string // Apple Team ID
}

// GenerateMapKitJWT creates a signed ES256 JWT for Apple's MapKit APIs.
// The token is valid for 30 minutes.
func GenerateMapKitJWT(config MapKitConfig) (string, error) {
	authKey, err := token.AuthKeyFromFile(config.KeyFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read MapKit key from %s: %w", config.KeyFilePath, err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": config.TeamID,
		"iat": now.Unix(),
		"exp": now.Add(30 * time.Minute).Unix(),
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	jwtToken.Header["kid"] = config.KeyID

	signedToken, err := jwtToken.SignedString(authKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign MapKit JWT: %w", err)
	}

	return signedToken, nil
}
