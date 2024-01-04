package tests

import (
	"context"
	"fmt"
	"olympsis-server/auth/google"
	"testing"
)

/*
 * Tries to get keys from google's endpoint
 */
func TestGetPublicKeys(t *testing.T) {
	client := google.NewClient()
	val, err := client.GetPublicKeys(context.TODO(), "7c0b6913fe13820a333399ace426e70535a9a0bf")
	if err != nil && val == "" {
		t.Fatalf(err.Error())
	}
}

func TestValidateJWT(t *testing.T) {
	client := google.NewClient()
	token := "eyJhbGciOiJSUzI1NiIsImtpZCI6IjdjMGI2OTEzZmUxMzgyMGEzMzMzOTlhY2U0MjZlNzA1MzVhOWEwYmYiLCJ0eXAiOiJKV1QifQ.eyJpc3MiOiJodHRwczovL2FjY291bnRzLmdvb2dsZS5jb20iLCJhenAiOiIxNDYxODI2NDk0NDYtbWpraWY1dm5wMGg5MWJoZmwwZG5xanJzdHVhdmZ0N2UuYXBwcy5nb29nbGV1c2VyY29udGVudC5jb20iLCJhdWQiOiIxNDYxODI2NDk0NDYtbWpraWY1dm5wMGg5MWJoZmwwZG5xanJzdHVhdmZ0N2UuYXBwcy5nb29nbGV1c2VyY29udGVudC5jb20iLCJzdWIiOiIxMTAxODQzODc1NzE4OTMyMjg5MjIiLCJoZCI6Im9seW1wc2lzLmNvbSIsImVtYWlsIjoiYWRtaW5Ab2x5bXBzaXMuY29tIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsIm5iZiI6MTY5NTI2Njc3NCwibmFtZSI6ImFkbWluIGFkbWluIiwicGljdHVyZSI6Imh0dHBzOi8vbGgzLmdvb2dsZXVzZXJjb250ZW50LmNvbS9hL0FDZzhvY0pVeEduODdheHpFLW1mQ1haZVo1R2lJU1RSbVM2TnFmRUxHejZhVlh4VT1zOTYtYyIsImdpdmVuX25hbWUiOiJhZG1pbiIsImZhbWlseV9uYW1lIjoiYWRtaW4iLCJsb2NhbGUiOiJlbiIsImlhdCI6MTY5NTI2NzA3NCwiZXhwIjoxNjk1MjcwNjc0LCJqdGkiOiJiMDViNWQ2NzAwNTdkZWU4NDY0NDFhOGY0ZWZmNjllYWM5ZjM1NjA5In0.X1pYFqgOpHvsl7ZSWsUL1AE1yS6z-56VqxbdahRAlXY5ukQ1VFTcovXzDLZF5jlVbMjMfPreNXnTGcsdp7uKB3pX4xfm3nXn_DtOlFXB_WsC_Z0yLQd_8eXQbJAaaoTJCXQJKbLQxqNi5o-CtWwiewlq9WTIPzqJNDUQgXzs2A3ADZ0OxgCh2BigoIebKi93CiludHoylSsnArnScU05-hD0-NGwUMbjeYX2bWb6bV1MNzoy_NsnDxwkTBB86Ifu5Q2WtLaFdjsCThg93viMXyJKgXOzHtPujjaqLWE6o1gJuv411gK5FVbvy6jNJDIQR_r1phGtNeQ6uE0AR_aySg"
	claims, err := client.ValidateJWT(token)
	if err != nil {
		t.Fatalf(err.Error())
	}
	fmt.Print(claims["email"])
}
