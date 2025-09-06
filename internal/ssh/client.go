package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"pm/internal/errors"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	sshClient *ssh.Client
	sftp      *sftp.Client
}

func NewClient(user, host, keyPath string, port int) (*SSHClient, error) {
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
	}, nil
}

func (c *SSHClient) Upload(src, dst string) error {
	if c == nil || c.sftp == nil {
		return errors.NewSSHConnectionError("nil", fmt.Errorf("SSH клиент не инициализирован"))
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return c.wrapSSHError("", src, dst, err)
	}
	defer srcFile.Close()

	if err := c.sftp.MkdirAll(filepath.Dir(dst)); err != nil {
		return c.wrapSSHError("", src, dst, fmt.Errorf("failed to create remote directory: %w", err))
	}

	dstFile, err := c.sftp.Create(dst)
	if err != nil {
		return c.wrapSSHError("", src, dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return c.wrapSSHError("", src, dst, err)
	}

	return nil
}

func (c *SSHClient) Download(src, dst string) error {
	if c == nil || c.sftp == nil {
		return errors.NewSSHConnectionError("nil", fmt.Errorf("SSH клиент не инициализирован"))
	}

	srcFile, err := c.sftp.Open(src)
	if err != nil {
		return c.wrapSSHError(src, "", dst, err)
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return errors.NewSSHFileTransferError("", src, dst, err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return c.wrapSSHError(src, "", dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return c.wrapSSHError(src, "", dst, err)
	}

	return nil
}

func (c *SSHClient) UploadReader(r io.Reader, dst string) error {
	if c == nil || c.sftp == nil {
		return errors.NewSSHConnectionError("nil", fmt.Errorf("SSH клиент не инициализирован"))
	}

	if err := c.sftp.MkdirAll(filepath.Dir(dst)); err != nil {
		return c.wrapSSHError("", "(in-memory)", dst, fmt.Errorf("failed to create remote directory: %w", err))
	}

	dstFile, err := c.sftp.Create(dst)
	if err != nil {
		return c.wrapSSHError("", "(in-memory)", dst, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, r); err != nil {
		return c.wrapSSHError("", "(in-memory)", dst, err)
	}

	return nil
}

func (c *SSHClient) ReadDir(path string) ([]os.FileInfo, error) {
	if c == nil || c.sftp == nil {
		return nil, errors.NewSSHConnectionError("nil", fmt.Errorf("SSH клиент не инициализирован"))
	}

	files, err := c.sftp.ReadDir(path)
	if err != nil {
		return nil, c.wrapSSHError("", "", path, err)
	}

	return files, nil
}

func (c *SSHClient) Close() error {
	var errs []error

	if c == nil {
		return errors.NewSSHConnectionError("nil", fmt.Errorf("SSH клиент не инициализирован"))
	}

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

func (c *SSHClient) wrapSSHError(server, source, target string, err error) error {
	if c != nil && c.sshClient != nil {
		server = string(c.sshClient.ServerVersion())
	}
	return errors.NewSSHFileTransferError(server, source, target, err)
}
