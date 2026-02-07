package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/olympsis/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func GetTokenFromHeader(r *http.Request) (string, error) {
	// Get the authorization header from the request
	authHeader := r.Header.Get("Authorization")

	// Check if the authorization header is present
	if authHeader == "" {
		return "", errors.New("authorization header not present")
	}

	// Return the token string
	return authHeader, nil
}

func GetClubTokenFromHeader(r *http.Request) (string, error) {
	token := r.Header.Get("X-Admin-Token")
	if token == "" {
		return "", errors.New("no club token found")
	}
	return token, nil
}

func ValidateClubID(s string) bool {
	_, err := bson.ObjectIDFromHex(s)
	return err == nil
}

func GenerateAuthToken(u string, p string) (string, error) {
	var key = []byte(os.Getenv("KEY"))
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)

	claims["iss"] = "https://api.olympsis.com"
	claims["sub"] = u
	claims["pod"] = p
	claims["iat"] = time.Now().Unix()
	claims["exp"] = time.Now().Add(30 * 24 * time.Hour).Unix() // 30 days

	ts, err := token.SignedString(key)

	if err != nil {
		return "", err
	}

	return ts, nil
}

func ValidateAuthToken(s string) (string, float64, float64, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(s, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("SECRET")), nil
	})

	if err != nil {
		return "", 0, 0, err
	} else {
		uuid, ok := claims["sub"].(string)
		if !ok {
			return "", 0, 0, errors.New("sub claim not found")
		}
		createdAt, ok := claims["iat"].(float64)
		if !ok {
			return "", 0, 0, errors.New("iat claim not found")
		}
		expiresAt, ok := claims["exp"].(float64)
		if !ok {
			return "", 0, 0, errors.New("exp claim not found")
		} else {
			now := time.Now().Unix()
			if expiresAt < float64(now) {
				return "", 0, 0, errors.New("token is expired")
			}
		}

		return uuid, createdAt, expiresAt, nil
	}
}

func GenerateClubToken(i string, r string, u string) (string, error) {
	var key = []byte(os.Getenv("KEY"))
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)

	claims["iss"] = i
	claims["sub"] = u
	claims["role"] = r

	ts, err := token.SignedString(key)

	if err != nil {
		return "", err
	}

	return ts, nil
}

func ValidateClubToken(s string, u string) (string, string, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(s, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("KEY")), nil
	})

	if err != nil {
		return "", "", err
	} else {
		id := claims["iss"].(string)
		uuid := claims["sub"].(string)
		role := claims["role"].(string)

		if uuid != u {
			return "", "", errors.New("uuid does not match")
		}

		return id, role, nil
	}
}

func CreateImageHash(input string) string {
	h := sha256.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil)) + ".png"
}

func RemovePostsByPosterUUIDs(posts *[]models.Post, uuids []string) *[]models.Post {
	var result []models.Post
	if len(*posts) > 0 {
		for _, post := range *posts {
			found := false
			for _, uuid := range uuids {
				if post.Poster != nil {
					if post.Poster.UUID == uuid {
						found = true
						break
					}
				}
			}
			if !found {
				result = append(result, post)
			}
		}
	}
	return &result
}

// Add to the existing service functions
func GetDefaultTextStyles() (models.TextStyleConfig, models.TextStyleConfig) {
	// Default title style
	titleStyle := models.TextStyleConfig{
		FontSize:      "title",
		FontWeight:    "bold",
		Color:         "FFFFFF",
		TextAlign:     "left",
		LineHeight:    "1.2",
		LetterSpacing: "normal",
	}

	// Default subtitle style
	subtitleStyle := models.TextStyleConfig{
		FontSize:      "subtitle",
		FontWeight:    "normal",
		Color:         "#FFFFFF",
		TextAlign:     "left",
		LineHeight:    "1.4",
		LetterSpacing: "normal",
	}

	return titleStyle, subtitleStyle
}

// Helper to apply default styles where properties are missing
func FillMissingTextStyles(style *models.TextStyleConfig, defaultStyle models.TextStyleConfig) {
	if style.FontSize == "" {
		style.FontSize = defaultStyle.FontSize
	}
	if style.FontWeight == "" {
		style.FontWeight = defaultStyle.FontWeight
	}
	if style.Color == "" {
		style.Color = defaultStyle.Color
	}
	if style.TextAlign == "" {
		style.TextAlign = defaultStyle.TextAlign
	}
	if style.LineHeight == "" {
		style.LineHeight = defaultStyle.LineHeight
	}
	if style.LetterSpacing == "" {
		style.LetterSpacing = defaultStyle.LetterSpacing
	}
}

// Helper function to ensure text emphasis settings are consistent
func ApplyTextEmphasisDefaults(announcement *models.Announcement) {
	// If text emphasis is not set, default to title
	if announcement.TextEmphasis == "" {
		announcement.TextEmphasis = models.EmphasisTitle
	}

	// Get default styles
	defaultTitleStyle, defaultSubtitleStyle := GetDefaultTextStyles()

	// Apply emphasis-specific adjustments
	switch announcement.TextEmphasis {
	case models.EmphasisTitle:
		// Title is more prominent - larger font for title, smaller for subtitle
		if announcement.TitleStyle.FontSize == "" {
			announcement.TitleStyle.FontSize = "title"
		}
		if announcement.SubtitleStyle.FontSize == "" {
			announcement.SubtitleStyle.FontSize = "subtitle"
		}

	case models.EmphasisSubtitle:
		// Subtitle is more prominent - larger font for subtitle, smaller for title
		if announcement.TitleStyle.FontSize == "" {
			announcement.TitleStyle.FontSize = "subtitle"
		}
		if announcement.SubtitleStyle.FontSize == "" {
			announcement.SubtitleStyle.FontSize = "title"
		}

	case models.EmphasisEqual:
		// Equal prominence - similar font sizes
		if announcement.TitleStyle.FontSize == "" {
			announcement.TitleStyle.FontSize = "title"
		}
		if announcement.SubtitleStyle.FontSize == "" {
			announcement.SubtitleStyle.FontSize = "title"
		}
	}

	// Fill in any missing style properties with defaults
	FillMissingTextStyles(&announcement.TitleStyle, defaultTitleStyle)
	FillMissingTextStyles(&announcement.SubtitleStyle, defaultSubtitleStyle)
}
