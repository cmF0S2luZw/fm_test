package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"pm/internal/errors"
	"pm/internal/logger"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	sshClient *ssh.Client
	sftp      *sftp.Client
	logger    logger.LoggerInterface
}

func NewClient(user, host, keyPath string, port int, log logger.LoggerInterface) (*SSHClient, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, errors.NewSSHConnectionError(host, err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, errors.NewSSHConnectionError(host, err)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	sshConn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, errors.NewSSHConnectionError(host, err)
	}

	sftpClient, err := sftp.NewClient(sshConn)
	if err != nil {
		sshConn.Close()
		return nil, errors.NewSSHConnectionError(host, err)
	}

	return &SSHClient{
		sshClient: sshConn,
		sftp:      sftpClient,
		logger:    log,
	}, nil
}

func (c *SSHClient) Upload(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return errors.NewSSHFileTransferError(string(c.sshClient.ServerVersion()), src, dst, err)
	}
	defer srcFile.Close()

	dir := filepath.Dir(dst)
	_ = c.sftp.MkdirAll(dir)

	dstFile, err := c.sftp.Create(dst)
	if err != nil {
		return errors.NewSSHFileTransferError(string(c.sshClient.ServerVersion()), src, dst, err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return errors.NewSSHFileTransferError(string(c.sshClient.ServerVersion()), src, dst, err)
	}

	return nil
}

func (c *SSHClient) Download(src, dst string) error {
	srcFile, err := c.sftp.Open(src)
	if err != nil {
		return errors.NewSSHFileTransferError(string(c.sshClient.ServerVersion()), src, dst, err)
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return errors.NewSSHFileTransferError(string(c.sshClient.ServerVersion()), src, dst, err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return errors.NewSSHFileTransferError(string(c.sshClient.ServerVersion()), src, dst, err)
	}

	return nil
}

func (c *SSHClient) UploadReader(r io.Reader, dst string) error {
	dir := filepath.Dir(dst)
	_ = c.sftp.MkdirAll(dir)

	dstFile, err := c.sftp.Create(dst)
	if err != nil {
		return errors.NewSSHFileTransferError(string(c.sshClient.ServerVersion()), "(in-memory)", dst, err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, r)
	if err != nil {
		return errors.NewSSHFileTransferError(string(c.sshClient.ServerVersion()), "(in-memory)", dst, err)
	}

	return nil
}

func (c *SSHClient) Close() error {
	var errs []error

	if c.sftp != nil {
		if err := c.sftp.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (c *SSHClient) ReadDir(path string) ([]os.FileInfo, error) {
	files, err := c.sftp.ReadDir(path)
	if err != nil {
		return nil, errors.NewSSHFileTransferError(string(c.sshClient.ServerVersion()), "", path, err)
	}
	return files, nil
}
