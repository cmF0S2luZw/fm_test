// internal/errors/error.go
package errors

import "fmt"

var (
	ErrNoFilesFound     = fmt.Errorf("ни одного файла не найдено по указанным путям")
	ErrEmptyFileList    = fmt.Errorf("список файлов пуст")
	ErrInvalidSSHConfig = fmt.Errorf("некорректная конфигурация SSH")
	ErrUnknownCommand   = fmt.Errorf("отсутствует команда")
)

type UnknownCommandError struct {
	Command string
}

func (e *UnknownCommandError) Error() string {
	return fmt.Sprintf("неизвестная команда: %s", e.Command)
}

type ArchiveCollectionError struct {
	Pattern string
	Err     error
}

func (e *ArchiveCollectionError) Error() string {
	return fmt.Sprintf("ошибка сбора файлов для шаблона %q", e.Pattern)
}

func (e *ArchiveCollectionError) Unwrap() error {
	return e.Err
}

func NewArchiveCollectionError(pattern string, err error) error {
	return &ArchiveCollectionError{Pattern: pattern, Err: err}
}

type PartialCollectionError struct {
	FailedPatterns []string
	FoundFiles     []string
}

func (e *PartialCollectionError) Error() string {
	return fmt.Sprintf("предупреждение: не найдены некоторые пути (%d шаблонов)", len(e.FailedPatterns))
}

func NewPartialCollectionError(failedPatterns []string, foundFiles []string) error {
	return &PartialCollectionError{
		FailedPatterns: failedPatterns,
		FoundFiles:     foundFiles,
	}
}

type ArchiveCreationError struct {
	File  string
	Files []string
	Err   error
}

func (e *ArchiveCreationError) Error() string {
	return fmt.Sprintf("ошибка создания архива %q", e.File)
}

func (e *ArchiveCreationError) Unwrap() error {
	return e.Err
}

func NewArchiveCreationError(file string, files []string, err error) error {
	return &ArchiveCreationError{File: file, Files: files, Err: err}
}

type ArchiveExtractionError struct {
	ZipPath string
	DestDir string
	Err     error
}

func (e *ArchiveExtractionError) Error() string {
	return fmt.Sprintf("ошибка распаковки архива %q в %q", e.ZipPath, e.DestDir)
}

func (e *ArchiveExtractionError) Unwrap() error {
	return e.Err
}

func NewArchiveExtractionError(zipPath, destDir string, err error) error {
	return &ArchiveExtractionError{ZipPath: zipPath, DestDir: destDir, Err: err}
}

type SSHConnectionError struct {
	Server string
	Err    error
}

func (e *SSHConnectionError) Error() string {
	return fmt.Sprintf("ошибка подключения к серверу %q", e.Server)
}

func (e *SSHConnectionError) Unwrap() error {
	return e.Err
}

func NewSSHConnectionError(server string, err error) error {
	return &SSHConnectionError{Server: server, Err: err}
}

type SSHFileTransferError struct {
	Server string
	Source string
	Target string
	Err    error
}

func (e *SSHFileTransferError) Error() string {
	return fmt.Sprintf("ошибка передачи файла с %q на %q", e.Source, e.Target)
}

func (e *SSHFileTransferError) Unwrap() error {
	return e.Err
}

func NewSSHFileTransferError(server, source, target string, err error) error {
	return &SSHFileTransferError{
		Server: server,
		Source: source,
		Target: target,
		Err:    err,
	}
}

type VersionError struct {
	Version    string
	Constraint string
	Err        error
}

func (e *VersionError) Error() string {
	if e.Constraint != "" {
		return fmt.Sprintf("ошибка проверки версии: %q не удовлетворяет условию %q", e.Version, e.Constraint)
	}
	if e.Err != nil {
		return fmt.Sprintf("ошибка парсинга версии: %s", e.Err.Error())
	}
	return "ошибка версии"
}

func (e *VersionError) Unwrap() error {
	return e.Err
}

func NewVersionError(version, constraint string, err error) error {
	return &VersionError{
		Version:    version,
		Constraint: constraint,
		Err:        err,
	}
}
