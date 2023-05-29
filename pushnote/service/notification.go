package service

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/certificate"
	"github.com/sideshow/apns2/payload"
	"github.com/sirupsen/logrus"
)

/*
Create new field service struct
*/
func NewNotificationService(l *logrus.Logger, r *mux.Router) *Service {
	return &Service{Logger: l, Router: r}
}

/*
Create apns client from p12 file
*/
func (p *Service) CreateNewClient() {
	cert, err := certificate.FromP12File("./pushnote/files/cert.p12", "")
	if err != nil {
		p.Logger.Fatal("token error:", err)
	}

	p.Client = apns2.NewClient(cert).Development()
}

func (s *Service) ConnectToDatabase() {
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	addr := os.Getenv("DB_ADDR")
	dbName := os.Getenv("TOPIC_DB_NAME")

	connStr := "postgres://" + user + ":" + password + "@" + addr + "/" + dbName + "?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		s.Logger.Fatal("failed to connect to database ", err)
	}

	s.Database = pool
	s.Logger.Info("Notification service established connection to database")
}

func (s *Service) PushNote(t string, b string, tk string) error {
	notification := &apns2.Notification{}
	notification.DeviceToken = tk
	notification.Topic = "com.coronislabs.Olympsis"
	notification.Payload = payload.NewPayload().AlertTitle(t).AlertBody(b).Badge(1)
	notification.Priority = 5

	res, err := s.Client.Push(notification)
	if err != nil {
		return err
	}

	if res.Sent() {
		s.Logger.Debug("Sent:", res.ApnsID)
		return nil
	} else {
		errString := fmt.Sprintf("Not Sent: %v %v %v\n", res.StatusCode, res.ApnsID, res.Reason)
		return errors.New(errString)
	}
}

/*
Send Notification to token
*/
func (s *Service) SendNotificationToToken(note *Notification, tk string) error {
	err := s.PushNote(note.Title, note.Body, tk)
	if err != nil {
		return err
	}
	return nil
}

/*
Send Notification to topic
*/
func (s *Service) SendNotificationToTopic(note *Notification) error {
	topic, err := s.GetTopic(note.Topic)
	if err != nil {
		return err
	}
	for i := range topic.Tokens {
		s.PushNote(note.Title, note.Body, topic.Tokens[i].Token)
	}
	return nil
}

/*
Create a topic
*/
func (s *Service) CreateTopic(name string) error {
	// topic := Topic{
	// 	Name:   name,
	// 	Tokens: tokens,
	// }
	// topicJSON, err := json.Marshal(topic)
	// if err != nil {
	// 	return err
	// }

	createStmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		ID SERIAL PRIMARY KEY, 
		uuid VARCHAR(50),
		token VARCHAR(100)
	)`, name)

	_, err := s.Database.Query(context.TODO(), createStmt)

	if err != nil {
		return err
	}
	return nil
}

/*
Get a topic
*/
func (s *Service) GetTopic(name string) (Topic, error) {
	queryStmt := fmt.Sprintf(`SELECT uuid, token FROM %s`, name)
	rows, err := s.Database.Query(context.TODO(), queryStmt)
	if err != nil {
		return Topic{}, err
	}
	defer rows.Close()

	var tokens []Token
	for rows.Next() {
		var ud string
		var tk string
		err = rows.Scan(&ud, &tk)
		if err != nil {
			return Topic{}, err
		}
		token := Token{
			UUID:  ud,
			Token: tk,
		}
		tokens = append(tokens, token)
	}

	topic := Topic{
		Name:   name,
		Tokens: tokens,
	}
	return topic, nil
}

/*
Add tokens to a topic
*/
func (s *Service) AddTokenToTopic(name string, uuid string, token string) error {

	insertStmt := fmt.Sprintf(`INSERT INTO %s (uuid, token) VALUES ($1, $2)`, name)

	// add token to topic table
	_, err := s.Database.Exec(context.TODO(), insertStmt, uuid, token)
	if err != nil {
		return err
	}

	return nil
}

/*
Remove token from topic
*/
func (s *Service) RemoveTokenFromTopic(name string, token string) error {
	updateStmt := fmt.Sprintf("DELETE FROM %s WHERE token=$1", name)

	// remove entry from table
	_, err := s.Database.Exec(context.TODO(), updateStmt, token)
	if err != nil {
		return err
	}

	return nil
}

/*
Delete a topic
*/
func (s *Service) DeleteTopic(name string) error {
	deleteSmt := fmt.Sprintf("DROP TABLE IF EXISTS %s", name)
	_, err := s.Database.Exec(context.TODO(), deleteSmt)
	if err != nil {
		return err
	}

	return nil
}
