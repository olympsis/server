package tests

import (
	"fmt"
	"olympsis-server/database"
	search "olympsis-server/search"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSearchUserByUUID(t *testing.T) {
	// set env
	os.Setenv("ENV", "PORT=8080")
	os.Setenv("DB_ADDR", "192.168.1.205")
	os.Setenv("DB_USR", "service")
	os.Setenv("DB_PASS", "qN1PHHgo6L942AvpTgGQ")
	os.Setenv("DB_NAME", "olympsis")
	os.Setenv("AUTH_COL", "auth")
	os.Setenv("USER_COL", "users")
	os.Setenv("CLUB_COL", "clubs")
	os.Setenv("EVENT_COL", "events")
	os.Setenv("FIELD_COL", "fields")
	os.Setenv("POST_COL", "posts")
	os.Setenv("CINVITE_COL", "clubInvites")
	os.Setenv("COMMENTS_COL", "comments")
	os.Setenv("FREQUEST_COL", "friendRequests")
	os.Setenv("CAPPICATIONS_COL", "clubApplications")
	os.Setenv("KEY", "SZkp78avQkxGyjRakxb5Ob08zqjguNRA")
	os.Setenv("POSTGRES_USER", "postgres")
	os.Setenv("POSTGRES_PASSWORD", "20031998")
	os.Setenv("TOPIC_DB_NAME", "olympsis_notif")

	// logger
	l := logrus.New()

	// database
	d := database.NewDatabase(l)
	d.EstablishConnection()

	s := search.NewSearchService(l, d)

	user, err := s.SearchUserByUUID("1edbdae3-55ea-4934-a1fa-f6d73f0fe951")
	if err != nil {
		t.Error(err.Error())
		return
	}
	fmt.Println("Search successful: ")
	fmt.Println(user)
}

func TestSearchUserByUsername(t *testing.T) {
	// set env
	os.Setenv("ENV", "PORT=8080")
	os.Setenv("DB_ADDR", "192.168.1.205")
	os.Setenv("DB_USR", "service")
	os.Setenv("DB_PASS", "qN1PHHgo6L942AvpTgGQ")
	os.Setenv("DB_NAME", "olympsis")
	os.Setenv("AUTH_COL", "auth")
	os.Setenv("USER_COL", "users")
	os.Setenv("CLUB_COL", "clubs")
	os.Setenv("EVENT_COL", "events")
	os.Setenv("FIELD_COL", "fields")
	os.Setenv("POST_COL", "posts")
	os.Setenv("CINVITE_COL", "clubInvites")
	os.Setenv("COMMENTS_COL", "comments")
	os.Setenv("FREQUEST_COL", "friendRequests")
	os.Setenv("CAPPICATIONS_COL", "clubApplications")
	os.Setenv("KEY", "SZkp78avQkxGyjRakxb5Ob08zqjguNRA")
	os.Setenv("POSTGRES_USER", "postgres")
	os.Setenv("POSTGRES_PASSWORD", "20031998")
	os.Setenv("TOPIC_DB_NAME", "olympsis_notif")

	// logger
	l := logrus.New()

	// database
	d := database.NewDatabase(l)
	d.EstablishConnection()

	s := search.NewSearchService(l, d)

	user, err := s.SearchUserByUsername("joeljojo")
	if err != nil {
		t.Error(err.Error())
		return
	}
	fmt.Println("Search successful: ")
	fmt.Println(user)
}
