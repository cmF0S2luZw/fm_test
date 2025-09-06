package errors

import "fmt"

// Общие ошибки, используемые во всем проекте
var (
	ErrNoFilesFound     = fmt.Errorf("ни одного файла не найдено по указанным путям")
	ErrEmptyFileList    = fmt.Errorf("список файлов пуст")
	ErrInvalidSSHConfig = fmt.Errorf("некорректная конфигурация SSH")
)

// UnknownCommandError возникает при вводе неизвестной команды
type UnknownCommandError struct {
	Command string
}

func (e *UnknownCommandError) Error() string {
	return fmt.Sprintf("неизвестная команда: %s", e.Command)
}

// ArchiveCollectionError возникает при ошибках сбора файлов
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

// NewArchiveCollectionError создает новую ошибку сбора файлов
func NewArchiveCollectionError(pattern string, err error) error {
	return &ArchiveCollectionError{Pattern: pattern, Err: err}
}

// PartialCollectionError возникает когда часть файлов найдена, часть нет
type PartialCollectionError struct {
	FailedPatterns []string
	FoundFiles     []string
}

func (e *PartialCollectionError) Error() string {
	return fmt.Sprintf("предупреждение: не найдены некоторые пути (%d шаблонов)", len(e.FailedPatterns))
}

// NewPartialCollectionError создает новую частичную ошибку сбора
func NewPartialCollectionError(failedPatterns []string, foundFiles []string) error {
	return &PartialCollectionError{
		FailedPatterns: failedPatterns,
		FoundFiles:     foundFiles,
	}
}

// ArchiveCreationError возникает при ошибках создания архива
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

// NewArchiveCreationError создает новую ошибку создания архива
func NewArchiveCreationError(file string, files []string, err error) error {
	return &ArchiveCreationError{File: file, Files: files, Err: err}
}

// ArchiveExtractionError возникает при ошибках распаковки архива
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

// NewArchiveExtractionError создает новую ошибку распаковки архива
func NewArchiveExtractionError(zipPath, destDir string, err error) error {
	return &ArchiveExtractionError{ZipPath: zipPath, DestDir: destDir, Err: err}
}

// SSHConnectionError возникает при ошибках подключения по SSH
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

// NewSSHConnectionError создает новую ошибку подключения по SSH
func NewSSHConnectionError(server string, err error) error {
	return &SSHConnectionError{Server: server, Err: err}
}

// SSHFileTransferError возникает при ошибках передачи файлов по SSH
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

// NewSSHFileTransferError создает новую ошибку передачи файлов по SSH
func NewSSHFileTransferError(server, source, target string, err error) error {
	return &SSHFileTransferError{
		Server: server,
		Source: source,
		Target: target,
		Err:    err,
	}
}
