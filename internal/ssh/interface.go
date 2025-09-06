package ssh

import (
	"io"
	"os"
)

type ClientInterface interface {
	Upload(src, dst string) error
	Download(src, dst string) error
	UploadReader(r io.Reader, dst string) error
	ReadDir(path string) ([]os.FileInfo, error)
	Close() error
}
