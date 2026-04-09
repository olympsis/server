package types

// StorageUploader is the interface for uploading files to GCP Storage.
// Defined in a standalone package to avoid import cycles between server, utils, and storage.
type StorageUploader interface {
	UploadToStorage(file []byte, bucket string, name string) error
}
