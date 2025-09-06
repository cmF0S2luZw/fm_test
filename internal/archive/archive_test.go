package archive

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"pm/config"
	"runtime"
	"strings"
	"testing"
)

type mockLogger struct {
	logs []string
}

func (m *mockLogger) Debug(msg string, args ...interface{}) {
	m.logs = append(m.logs, "[DEBUG] "+format(msg, args...))
}

func (m *mockLogger) Info(msg string, args ...interface{}) {
	m.logs = append(m.logs, "[INFO] "+format(msg, args...))
}

func (m *mockLogger) Warn(msg string, args ...interface{}) {
	m.logs = append(m.logs, "[WARN] "+format(msg, args...))
}

func (m *mockLogger) Error(msg string, args ...interface{}) {
	m.logs = append(m.logs, "[ERROR] "+format(msg, args...))
}

func format(msg string, args ...interface{}) string {
	if len(args) == 0 {
		return msg
	}
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(fmt.Sprintf(msg, args...), "\n", " "), "\t", " "))
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[string]int)
	for _, item := range a {
		counts[item]++
	}
	for _, item := range b {
		counts[item]--
		if counts[item] < 0 {
			return false
		}
	}
	for _, c := range counts {
		if c != 0 {
			return false
		}
	}
	return true
}

func logContains(logs []string, substr string) bool {
	for _, log := range logs {
		if strings.Contains(log, substr) {
			return true
		}
	}
	return false
}

func createFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func createZip(t *testing.T, path string) {
	t.Helper()
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	zipWriter := zip.NewWriter(out)
	defer zipWriter.Close()

	addFile := func(name, content string) {
		header := &zip.FileHeader{
			Name:   name,
			Method: zip.Deflate,
		}
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte(content))
	}

	addFile("readme.txt", "Hello")
	addFile("src/main.go", "package main")
	addFile("docs/", "")
}

func TestCollectFiles(t *testing.T) {
	tempDir := t.TempDir()

	createFile(t, tempDir, "file1.txt", "test")
	createFile(t, tempDir, "file2.go", "test")
	createFile(t, tempDir, "subdir/file3.txt", "test")
	createFile(t, tempDir, "exclude_me.tmp", "test")

	tests := []struct {
		name        string
		targets     []config.Target
		wantFiles   []string
		expectError bool
		logContains []string
	}{
		{
			name: "один шаблон, найдены файлы",
			targets: []config.Target{
				{Path: filepath.Join(tempDir, "*.txt")},
			},
			wantFiles: []string{
				filepath.Join(tempDir, "file1.txt"),
			},
			expectError: false,
			logContains: []string{"Найдено совпадений", "Файл добавлен в архив"},
		},
		{
			name: "несколько шаблонов",
			targets: []config.Target{
				{Path: filepath.Join(tempDir, "*.txt")},
				{Path: filepath.Join(tempDir, "*.go")},
				{Path: filepath.Join(tempDir, "subdir", "*.txt")},
			},
			wantFiles: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "file2.go"),
				filepath.Join(tempDir, "subdir", "file3.txt"),
			},
			expectError: false,
		},
		{
			name: "шаблон не дал результатов",
			targets: []config.Target{
				{Path: filepath.Join(tempDir, "*.pdf")},
			},
			wantFiles:   nil,
			expectError: true,
			logContains: []string{"Не найдено ни одного файла для архивации"},
		},
		{
			name: "исключение файла",
			targets: []config.Target{
				{
					Path:    filepath.Join(tempDir, "*.*"),
					Exclude: "*.tmp",
				},
			},
			wantFiles: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "file2.go"),
			},
			expectError: false,
			logContains: []string{"Файл исключен"},
		},
		{
			name: "ошибка в шаблоне исключения",
			targets: []config.Target{
				{
					Path:    filepath.Join(tempDir, "*.txt"),
					Exclude: "foo[bar",
				},
			},
			wantFiles:   nil,
			expectError: true,
			logContains: []string{"Ошибка обработки исключения"},
		},
		{
			name: "пропуск директорий",
			targets: []config.Target{
				{Path: filepath.Join(tempDir, "*")},
			},
			wantFiles: []string{
				filepath.Join(tempDir, "file1.txt"),
				filepath.Join(tempDir, "file2.go"),
				filepath.Join(tempDir, "exclude_me.tmp"),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLog := &mockLogger{}
			got, err := CollectFiles(mockLog, tt.targets)

			if tt.expectError {
				if err == nil {
					t.Fatal("Ожидалась ошибка, но её не было")
				}
				if len(got) != 0 {
					t.Errorf("Ожидался пустой результат, получено: %v", got)
				}
			} else {
				if err != nil {
					t.Fatalf("Не ожидалась ошибка: %v", err)
				}
				if !equalStringSlices(got, tt.wantFiles) {
					t.Errorf("Файлы не совпадают.\nОжидалось: %v\nПолучено: %v", tt.wantFiles, got)
				}
			}

			for _, substr := range tt.logContains {
				if !logContains(mockLog.logs, substr) {
					t.Errorf("В логах не найдено: %q\nЛоги: %v", substr, mockLog.logs)
				}
			}
		})
	}
}

func TestCreateZip(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "test.zip")

	file1 := createFile(t, tempDir, "data.txt", "hello")
	file2 := createFile(t, tempDir, "src/main.go", "package main")

	tests := []struct {
		name        string
		files       []string
		outputPath  string
		expectError bool
		logContains []string
	}{
		{
			name:        "успешное создание архива",
			files:       []string{file1, file2},
			outputPath:  outputPath,
			expectError: false,
			logContains: []string{"Архив успешно создан"},
		},
		{
			name:        "пустой список файлов",
			files:       []string{},
			outputPath:  outputPath,
			expectError: true,
			logContains: []string{"Попытка создать архив без файлов"},
		},
		{
			name:        "файл не существует",
			files:       []string{filepath.Join(tempDir, "missing.txt")},
			outputPath:  outputPath,
			expectError: true,
			logContains: []string{"Ошибка открытия файла для архивации"},
		},
		{
			name:        "недоступный путь для записи",
			files:       []string{file1},
			outputPath:  "/restricted/path/forbidden.zip",
			expectError: true,
			logContains: []string{"Ошибка создания файла архива"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLog := &mockLogger{}
			err := CreateZip(mockLog, tt.files, tt.outputPath)

			if tt.expectError {
				if err == nil {
					t.Fatal("Ожидалась ошибка, но её не было")
				}
			} else {
				if err != nil {
					t.Fatalf("Не ожидалась ошибка: %v", err)
				}
				if _, err := os.Stat(tt.outputPath); os.IsNotExist(err) {
					t.Fatalf("Архив не был создан: %s", tt.outputPath)
				}

				zipReader, err := zip.OpenReader(tt.outputPath)
				if err != nil {
					t.Fatalf("Не удалось открыть zip: %v", err)
				}
				defer zipReader.Close()

				var archivedNames []string
				for _, f := range zipReader.File {
					archivedNames = append(archivedNames, f.Name)
				}

				expectedNames := []string{"data.txt", filepath.ToSlash(filepath.Join("src", "main.go"))}
				if runtime.GOOS == "windows" {
					for i := range expectedNames {
						expectedNames[i] = filepath.ToSlash(expectedNames[i])
					}
				}

				if !equalStringSlices(archivedNames, expectedNames) {
					t.Errorf("Файлы в архиве не совпадают.\nОжидалось: %v\nПолучено: %v", expectedNames, archivedNames)
				}
			}

			for _, substr := range tt.logContains {
				if !logContains(mockLog.logs, substr) {
					t.Errorf("В логах не найдено: %q\nЛоги: %v", substr, mockLog.logs)
				}
			}
		})
	}
}

func TestExtractZip(t *testing.T) {
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	createZip(t, zipPath)

	corruptPath := filepath.Join(tempDir, "corrupt.zip")
	if err := os.WriteFile(corruptPath, []byte("not a zip"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		zipPath     string
		destDir     string
		expectError bool
		logContains []string
	}{
		{
			name:        "успешная распаковка",
			zipPath:     zipPath,
			destDir:     destDir,
			expectError: false,
			logContains: []string{"Распаковка архива завершена успешно"},
		},
		{
			name:        "архив не существует",
			zipPath:     filepath.Join(tempDir, "missing.zip"),
			destDir:     destDir,
			expectError: true,
			logContains: []string{"Ошибка открытия архива"},
		},
		{
			name:    "недоступная директория назначения",
			zipPath: zipPath,
			destDir: func() string {
				if runtime.GOOS == "windows" {
					return `nul`
				}
				return "/dev/.invalid"
			}(),
			expectError: true,
			logContains: []string{"Ошибка создания родительской директории"},
		},
		{
			name:        "поврежденный архив",
			zipPath:     corruptPath,
			destDir:     destDir,
			expectError: true,
			logContains: []string{"Ошибка открытия архива"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLog := &mockLogger{}
			err := ExtractZip(mockLog, tt.zipPath, tt.destDir)

			if tt.expectError {
				if err == nil {
					t.Fatal("Ожидалась ошибка, но её не было")
				}
			} else {
				if err != nil {
					t.Fatalf("Не ожидалась ошибка: %v", err)
				}
				if _, err := os.Stat(filepath.Join(tt.destDir, "readme.txt")); os.IsNotExist(err) {
					t.Error("Файл readme.txt не был распакован")
				}
				if _, err := os.Stat(filepath.Join(tt.destDir, "src", "main.go")); os.IsNotExist(err) {
					t.Error("Файл src/main.go не был распакован")
				}
			}

			for _, substr := range tt.logContains {
				if !logContains(mockLog.logs, substr) {
					t.Errorf("В логах не найдено: %q\nЛоги: %v", substr, mockLog.logs)
				}
			}
		})
	}
}
