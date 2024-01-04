package service

import (
	"context"
	"errors"
	"net/http"
)

func (s *Service) _uploadObject(file []byte, bucket string, name string) error {

	// upload file to bucket
	object := s.Client.Bucket(bucket).Object(name)
	wc := object.NewWriter(context.Background())
	_, err := wc.Write(file)
	if err != nil {
		return err
	}
	err = wc.Close()
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) _deleteObject(bucket string, name string) error {

	object := s.Client.Bucket(bucket).Object(name)
	err := object.Delete(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func GrabFileName(h *http.Header) (string, error) {
	// Get the filename from the request header.
	fileName := h.Get("X-Filename")
	if fileName == "" {
		return "", errors.New("missing X-Filename header")
	} else {
		return fileName, nil
	}
}
