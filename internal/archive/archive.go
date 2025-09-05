package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Target struct {
	Path    string `json:"path"`
	Exclude string `json:"exclude,omitempty"`
}

func CollectFiles(targets []Target) ([]string, error) {
	var files []string

	for _, target := range targets {
		matches, err := filepath.Glob(target.Path)
		if err != nil {
			return nil, fmt.Errorf("ошибка при обработке шаблона %s: %w", target.Path, err)
		}

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			if info.IsDir() {
				continue
			}

			if target.Exclude != "" {
				baseName := filepath.Base(match)
				matched, err := filepath.Match(target.Exclude, baseName)
				if err != nil {
					return nil, fmt.Errorf("ошибка при обработке исключения %s: %w", target.Exclude, err)
				}
				if matched {
					continue
				}
			}

			files = append(files, match)
		}
	}

	return files, nil
}

func CreateZip(files []string, outputPath string) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	for _, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}

		info, err := file.Stat()
		if err != nil {
			file.Close()
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			file.Close()
			return err
		}

		dirName := filepath.Base(filepath.Dir(filePath))
		fileName := filepath.Base(filePath)
		header.Name = filepath.ToSlash(filepath.Join(dirName, fileName))
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			file.Close()
			return err
		}

		_, err = io.Copy(writer, file)
		file.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func ExtractZip(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		filePath := filepath.Join(destDir, file.Name)

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()

		rc, err := file.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		_, err = io.Copy(outFile, rc)
		if err != nil {
			return err
		}
	}

	return nil
}
