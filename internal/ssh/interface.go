package ssh

import "io"

type ClientInterface interface {
	Upload(src, dst string) error
	Download(src, dst string) error
	UploadReader(r io.Reader, dst string) error
	Close() error
}
