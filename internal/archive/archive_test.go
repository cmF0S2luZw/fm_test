package archive

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := ioutil.TempDir("", "archive-test")
	require.NoError(t, err)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "archive_this1"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "archive_this2"), 0755))

	files := []struct {
		path    string
		content string
	}{
		{"archive_this1/file1.txt", "Content 1"},
		{"archive_this1/file2.txt", "Content 2"},
		{"archive_this2/file3.txt", "Content 3"},
		{"archive_this2/file4.tmp", "Temporary content"},
		{"archive_this2/file5.log", "Log content"},
	}

	for _, f := range files {
		err := ioutil.WriteFile(filepath.Join(tmpDir, f.path), []byte(f.content), 0644)
		require.NoError(t, err)
	}

	return tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}

func TestCollectFiles(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	targets := []Target{
		{Path: filepath.Join(tmpDir, "archive_this1", "*.txt")},
		{Path: filepath.Join(tmpDir, "archive_this2", "*"), Exclude: "*.tmp"},
	}

	files, err := CollectFiles(targets)
	require.NoError(t, err)

	expectedFiles := []string{
		filepath.Join(tmpDir, "archive_this1", "file1.txt"),
		filepath.Join(tmpDir, "archive_this1", "file2.txt"),
		filepath.Join(tmpDir, "archive_this2", "file3.txt"),
		filepath.Join(tmpDir, "archive_this2", "file5.log"),
	}

	assert.ElementsMatch(t, expectedFiles, files)
}

func TestCreateZip(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	targets := []Target{
		{Path: filepath.Join(tmpDir, "archive_this1", "*.txt")},
	}
	files, err := CollectFiles(targets)
	require.NoError(t, err)

	zipPath := filepath.Join(tmpDir, "test.zip")
	err = CreateZip(files, zipPath)
	require.NoError(t, err)

	_, err = os.Stat(zipPath)
	assert.NoError(t, err)

	zipReader, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	defer zipReader.Close()

	assert.Equal(t, 2, len(zipReader.File))

	expectedNames := []string{"archive_this1/file1.txt", "archive_this1/file2.txt"}
	var actualNames []string
	for _, f := range zipReader.File {
		actualNames = append(actualNames, f.Name)
	}

	assert.ElementsMatch(t, expectedNames, actualNames)
}

func TestExtractZip(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	targets := []Target{
		{Path: filepath.Join(tmpDir, "archive_this1", "*.txt")},
	}
	files, err := CollectFiles(targets)
	require.NoError(t, err)

	zipPath := filepath.Join(tmpDir, "test.zip")
	require.NoError(t, CreateZip(files, zipPath))

	extractDir := filepath.Join(tmpDir, "extracted")
	require.NoError(t, os.MkdirAll(extractDir, 0755))

	err = ExtractZip(zipPath, extractDir)
	require.NoError(t, err)

	expectedFiles := []string{
		filepath.Join(extractDir, "archive_this1", "file1.txt"),
		filepath.Join(extractDir, "archive_this1", "file2.txt"),
	}

	for _, file := range expectedFiles {
		_, err := os.Stat(file)
		assert.NoError(t, err, "Файл %s должен существовать", file)
	}

	for i, file := range expectedFiles {
		content, err := ioutil.ReadFile(file)
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Content %d", i+1), string(content))
	}
}

func TestCollectFiles_WithInvalidPattern(t *testing.T) {
	_, err := CollectFiles([]Target{
		{Path: "invalid[pattern"},
	})
	assert.Error(t, err)
}

func TestExtractZip_WithNonExistingFile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	err := ExtractZip(filepath.Join(tmpDir, "non-existing.zip"), tmpDir)
	assert.Error(t, err)
}
