package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	// Publick Key URL for google
	PublicKeyURL string = "https://www.googleapis.com/oauth2/v1/certs"
	// ContentType is the one expected by Apple
	ContentType string = "application/x-www-form-urlencoded"
	// UserAgent is required by Apple or the request will fail
	UserAgent string = "olympsis-server"
	// AcceptHeader is the content that we are willing to accept
	AcceptHeader string = "application/json"
)

// client struct to handle functions/validation
type Client struct {
	publicKeyURL string
	client       *http.Client
}

// new client object
func NewClient() *Client {
	return &Client{
		publicKeyURL: PublicKeyURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) GetPublicKeys(ctx context.Context, keyID string) (string, error) {
	results := map[string]string{}
	err := doRequest(ctx, c.client, "GET", &results, PublicKeyURL, url.Values{})
	if err != nil {
		return "", err
	}
	key, ok := results[keyID]
	if !ok {
		return "", errors.New("key not found")
	}
	return key, nil
}

func (c *Client) ValidateJWT(tokenString string) (GoogleClaims, error) {
	claims := GoogleClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		func(token *jwt.Token) (interface{}, error) {
			pem, err := c.GetPublicKeys(context.TODO(), fmt.Sprintf("%s", token.Header["kid"]))
			if err != nil {
				return nil, err
			}
			key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(pem))
			if err != nil {
				return nil, err
			}
			return key, nil
		},
	)
	if err != nil {
		return GoogleClaims{}, err
	}

	claims = token.Claims.(GoogleClaims)

	if claims.Issuer != "accounts.google.com" && claims.Issuer != "https://accounts.google.com" {
		return GoogleClaims{}, errors.New("iss is invalid")
	}

	if claims.Audience != "YOUR_CLIENT_ID_HERE" {
		return GoogleClaims{}, errors.New("aud is invalid")
	}

	if claims.ExpiresAt < time.Now().UTC().Unix() {
		return GoogleClaims{}, errors.New("JWT is expired")
	}

	return claims, nil
}

// perform http request
func doRequest(ctx context.Context, client *http.Client, method string, result interface{}, url string, data url.Values) error {
	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Add("content-type", ContentType)
	req.Header.Add("accept", AcceptHeader)
	req.Header.Add("user-agent", UserAgent)

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(result)
}
