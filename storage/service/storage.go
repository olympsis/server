package service

import (
	"context"
	"errors"
	"net/http"

	visionpb "cloud.google.com/go/vision/v2/apiv1/visionpb"
)

// UploadToStorage writes a file to the specified GCP Storage bucket.
// This is the exported version for use by other modules (e.g. map-snapshots).
func (s *Service) UploadToStorage(file []byte, bucket string, name string) error {
	return s._uploadObject(file, bucket, name)
}

// _uploadObject writes a file to the specified GCP Storage bucket.
// Detects the content type from the file bytes so GCS stores it correctly.
func (s *Service) _uploadObject(file []byte, bucket string, name string) error {
	object := s.Client.Bucket(bucket).Object(name)
	wc := object.NewWriter(context.Background())

	// Detect content type from the first 512 bytes of the file
	wc.ContentType = http.DetectContentType(file)

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

// _deleteObject removes a file from the specified GCP Storage bucket
func (s *Service) _deleteObject(bucket string, name string) error {
	object := s.Client.Bucket(bucket).Object(name)
	err := object.Delete(context.Background())
	if err != nil {
		return err
	}
	return nil
}

// bucketAliases maps legacy/incorrect bucket names sent by clients to the
// actual GCS bucket. TODO: remove once all clients are updated.
var bucketAliases = map[string]string{
	"olympsis-event-images": "olympsis-event-media",
}

// resolveBucket returns the actual GCS bucket name, applying any temporary
// alias mapping if one exists.
func resolveBucket(name string) string {
	if actual, ok := bucketAliases[name]; ok {
		return actual
	}
	return name
}

// GrabFileName extracts the filename from the X-Filename request header
func GrabFileName(h *http.Header) (string, error) {
	fileName := h.Get("X-Filename")
	if fileName == "" {
		return "", errors.New("missing X-Filename header")
	} else {
		return fileName, nil
	}
}

// AnnotateImage sends an image to the Google Vision API for safe search detection
func (s *Service) AnnotateImage(image []byte) (*visionpb.BatchAnnotateImagesResponse, error) {
	ann := visionpb.AnnotateImageRequest{
		Image: &visionpb.Image{
			Content: image,
		},
		Features: []*visionpb.Feature{
			{
				Type: visionpb.Feature_SAFE_SEARCH_DETECTION,
			},
		},
	}

	req := &visionpb.BatchAnnotateImagesRequest{
		Requests: []*visionpb.AnnotateImageRequest{&ann},
	}

	resp, err := s.VClient.BatchAnnotateImages(context.TODO(), req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// AggregateSafetyScore calculates a combined safety score (0-5) from Vision API annotations.
// A score of 5 means the image is unsafe and should be rejected.
func (s *Service) AggregateSafetyScore(response *visionpb.AnnotateImageResponse) *int {
	score := 0

	if response == nil || response.SafeSearchAnnotation == nil {
		return &score
	}
	annotation := response.SafeSearchAnnotation

	// Adult/Racy content check
	if annotation.Adult == visionpb.Likelihood_VERY_LIKELY {
		score = 5
		return &score
	} else if annotation.Racy == visionpb.Likelihood_VERY_LIKELY {
		score = 5
		return &score
	} else if annotation.Adult > visionpb.Likelihood_POSSIBLE && annotation.Racy > visionpb.Likelihood_POSSIBLE {
		score = 4
	} else if annotation.Adult <= visionpb.Likelihood_POSSIBLE && annotation.Racy <= visionpb.Likelihood_POSSIBLE {
		score = int(annotation.Racy.Number())
	}

	// Violent content check
	if annotation.Violence == visionpb.Likelihood_VERY_LIKELY {
		score = 5
		return &score
	} else if annotation.Violence <= visionpb.Likelihood_LIKELY {
		score += int(annotation.Violence.Number())
	}

	// Medical content check
	if annotation.Medical == visionpb.Likelihood_VERY_LIKELY {
		score = 5
		return &score
	} else if annotation.Medical <= visionpb.Likelihood_LIKELY {
		score += int(annotation.Medical.Number())
	}

	score /= 3

	return &score
}
