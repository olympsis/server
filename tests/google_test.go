package tests

import (
	"context"
	"olympsis-server/auth/google"
	"testing"
)

/*
 * Tries to get keys from google's endpoint
 */
func TestGetPublicKeys(t *testing.T) {
	client := google.NewClient()
	val, err := client.GetPublicKeys(context.TODO(), "")
	if err != nil && val == "" {
		t.Fatalf(err.Error())
	}
}
