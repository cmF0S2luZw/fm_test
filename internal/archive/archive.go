package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"pm/config"
	"pm/internal/errors"
	"pm/internal/logger"
)

func CollectFiles(log logger.LoggerInterface, targets []config.Target) ([]string, error) {
	log.Debug("Начало сборки файлов", "колличество шаблонов", len(targets))

	var files []string
	var failedPatterns []string

	for i, target := range targets {
		log.Debug("Обработка шаблона", "номер", i+1, "шаблон", target.Path)

		matches, err := filepath.Glob(target.Path)
		if err != nil {
			log.Error("Ошибка обработки шаблона", "шаблон", target.Path, "ошибка", err.Error())
			return nil, errors.NewArchiveCollectionError(target.Path, err)
		}

		log.Debug("Найдено совпадений", "шаблон", target.Path, "количество", len(matches))

		if len(matches) == 0 {
			log.Warn("Шаблон не дал результатов", "шаблон", target.Path)
			failedPatterns = append(failedPatterns, target.Path)
			continue
		}

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				log.Warn("Не удалось получить информацию о файле", "файл", match, "ошибка", err.Error())
				continue
			}
			if info.IsDir() {
				log.Debug("Пропуск директории", "директория", match)
				continue
			}

			if target.Exclude != "" {
				baseName := filepath.Base(match)
				matched, err := filepath.Match(target.Exclude, baseName)
				if err != nil {
					log.Error("Ошибка обработки исключения", "шаблон", target.Exclude, "файл", match, "ошибка", err.Error())
					return nil, errors.NewArchiveCollectionError(target.Path, err)
				}
				if matched {
					log.Debug("Файл исключен", "файл", match, "шаблон_исключения", target.Exclude)
					continue
				}
			}

			log.Debug("Файл добавлен в архив", "файл", match)
			files = append(files, match)
		}
	}

	if len(failedPatterns) > 0 {
		log.Warn("Не все шаблоны дали результаты",
			"количество_неудачных", len(failedPatterns),
			"шаблоны", failedPatterns,
			"найдено_файлов", len(files),
		)
	}

	if len(files) == 0 {
		log.Error("Не найдено ни одного файла для архивации")
		return nil, errors.NewArchiveCollectionError(
			"сбора файлов",
			errors.ErrNoFilesFound,
		)
	}

	log.Info("Сбор файлов завершен",
		"найдено_файлов", len(files),
		"количество_шаблонов", len(targets),
	)

	return files, nil
}

func CreateZip(log logger.LoggerInterface, files []string, outputPath string) error {
	log.Info("Начало создания архива",
		"выходной_файл", outputPath,
		"количество_файлов", len(files),
	)

	if len(files) == 0 {
		log.Error("Попытка создать архив без файлов")
		return errors.NewArchiveCreationError(
			outputPath,
			files,
			errors.ErrEmptyFileList,
		)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		log.Error("Ошибка создания файла архива",
			"файл", outputPath,
			"ошибка", err.Error(),
		)
		return errors.NewArchiveCreationError(outputPath, files, err)
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	root := filepath.Dir(outputPath)

	for i, filePath := range files {
		log.Debug("Добавление файла в архив",
			"номер", i+1,
			"всего", len(files),
			"файл", filePath,
		)

		file, err := os.Open(filePath)
		if err != nil {
			log.Error("Ошибка открытия файла для архивации",
				"файл", filePath,
				"ошибка", err.Error(),
			)
			return errors.NewArchiveCreationError(outputPath, files, err)
		}

		info, err := file.Stat()
		if err != nil {
			file.Close()
			log.Error("Ошибка получения информации о файле",
				"файл", filePath,
				"ошибка", err.Error(),
			)
			return errors.NewArchiveCreationError(outputPath, files, err)
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			file.Close()
			log.Error("Ошибка создания заголовка для файла",
				"файл", filePath,
				"ошибка", err.Error(),
			)
			return errors.NewArchiveCreationError(outputPath, files, err)
		}

		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			file.Close()
			log.Error("Ошибка определения относительного пути",
				"файл", filePath,
				"ошибка", err.Error(),
			)
			return errors.NewArchiveCreationError(outputPath, files, err)
		}
		header.Name = filepath.ToSlash(rel)
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			file.Close()
			log.Error("Ошибка создания записи в архиве",
				"файл", filePath,
				"ошибка", err.Error(),
			)
			return errors.NewArchiveCreationError(outputPath, files, err)
		}

		_, err = io.Copy(writer, file)
		file.Close()
		if err != nil {
			log.Error("Ошибка копирования файла в архив",
				"файл", filePath,
				"ошибка", err.Error(),
			)
			return errors.NewArchiveCreationError(outputPath, files, err)
		}
	}

	log.Info("Архив успешно создан",
		"файл", outputPath,
		"количество_файлов", len(files),
	)

	return nil
}

func ExtractZip(log logger.LoggerInterface, zipPath, destDir string) error {
	log.Info("Начало распаковки архива",
		"архив", zipPath,
		"цель", destDir,
	)

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		log.Error("Ошибка открытия архива",
			"архив", zipPath,
			"ошибка", err.Error(),
		)
		return errors.NewArchiveExtractionError(zipPath, destDir, err)
	}
	defer reader.Close()

	log.Debug("Архив содержит файлов", "количество", len(reader.File))

	for i, file := range reader.File {
		filePath := filepath.Join(destDir, file.Name)
		log.Debug("Обработка файла из архива",
			"номер", i+1,
			"всего", len(reader.File),
			"файл", file.Name,
		)

		if file.FileInfo().IsDir() {
			log.Debug("Создание директории", "директория", filePath)
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				log.Error("Ошибка создания директории",
					"директория", filePath,
					"ошибка", err.Error(),
				)
				return errors.NewArchiveExtractionError(zipPath, destDir, err)
			}
			continue
		}

		log.Debug("Создание родительской директории", "директория", filepath.Dir(filePath))
		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			log.Error("Ошибка создания родительской директории",
				"директория", filepath.Dir(filePath),
				"ошибка", err.Error(),
			)
			return errors.NewArchiveExtractionError(zipPath, destDir, err)
		}

		log.Debug("Создание файла", "файл", filePath)
		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			log.Error("Ошибка создания файла",
				"файл", filePath,
				"ошибка", err.Error(),
			)
			return errors.NewArchiveExtractionError(zipPath, destDir, err)
		}
		defer outFile.Close()

		log.Debug("Открытие файла из архива", "файл", file.Name)
		rc, err := file.Open()
		if err != nil {
			log.Error("Ошибка открытия файла из архива",
				"файл", file.Name,
				"ошибка", err.Error(),
			)
			return errors.NewArchiveExtractionError(zipPath, destDir, err)
		}
		defer rc.Close()

		log.Debug("Копирование содержимого", "из", file.Name, "в", filePath)
		_, err = io.Copy(outFile, rc)
		if err != nil {
			log.Error("Ошибка копирования содержимого",
				"из", file.Name,
				"в", filePath,
				"ошибка", err.Error(),
			)
			return errors.NewArchiveExtractionError(zipPath, destDir, err)
		}
	}

	log.Info("Распаковка архива завершена успешно",
		"архив", zipPath,
		"цель", destDir,
		"файлов_распаковано", len(reader.File),
	)

	return nil
}

func CreateTarGz(log logger.LoggerInterface, files []string, outputPath string) error {
	log.Info("Начало создания tar.gz архива",
		"выходной_файл", outputPath,
		"количество_файлов", len(files),
	)

	if len(files) == 0 {
		log.Error("Попытка создать архив без файлов")
		return errors.NewArchiveCreationError(
			outputPath,
			files,
			errors.ErrEmptyFileList,
		)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		log.Error("Ошибка создания файла архива",
			"файл", outputPath,
			"ошибка", err.Error(),
		)
		return errors.NewArchiveCreationError(outputPath, files, err)
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, filePath := range files {
		err := addToTar(tw, filePath)
		if err != nil {
			log.Error("Ошибка добавления файла в tar",
				"файл", filePath,
				"ошибка", err.Error(),
			)
			return errors.NewArchiveCreationError(outputPath, files, err)
		}
	}

	log.Info("Tar.gz архив успешно создан",
		"файл", outputPath,
		"количество_файлов", len(files),
	)

	return nil
}

func ExtractTarGz(log logger.LoggerInterface, tarGzPath, destDir string) error {
	log.Info("Начало распаковки tar.gz архива",
		"архив", tarGzPath,
		"цель", destDir,
	)

	file, err := os.Open(tarGzPath)
	if err != nil {
		log.Error("Ошибка открытия tar.gz архива",
			"архив", tarGzPath,
			"ошибка", err.Error(),
		)
		return errors.NewArchiveExtractionError(tarGzPath, destDir, err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		log.Error("Ошибка создания gzip ридера",
			"архив", tarGzPath,
			"ошибка", err.Error(),
		)
		return errors.NewArchiveExtractionError(tarGzPath, destDir, err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	fileCount := 0

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error("Ошибка чтения tar записи",
				"архив", tarGzPath,
				"ошибка", err.Error(),
			)
			return errors.NewArchiveExtractionError(tarGzPath, destDir, err)
		}

		fileCount++
		target := filepath.Join(destDir, header.Name)
		log.Debug("Обработка файла из архива",
			"файл", header.Name,
			"тип", header.Typeflag,
		)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
					log.Error("Ошибка создания директории",
						"директория", target,
						"ошибка", err.Error(),
					)
					return errors.NewArchiveExtractionError(tarGzPath, destDir, err)
				}
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
				log.Error("Ошибка создания родительской директории",
					"директория", filepath.Dir(target),
					"ошибка", err.Error(),
				)
				return errors.NewArchiveExtractionError(tarGzPath, destDir, err)
			}

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				log.Error("Ошибка создания файла",
					"файл", target,
					"ошибка", err.Error(),
				)
				return errors.NewArchiveExtractionError(tarGzPath, destDir, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				log.Error("Ошибка копирования содержимого",
					"из", header.Name,
					"в", target,
					"ошибка", err.Error(),
				)
				return errors.NewArchiveExtractionError(tarGzPath, destDir, err)
			}
			outFile.Close()
		}
	}

	log.Info("Распаковка tar.gz архива завершена успешно",
		"архив", tarGzPath,
		"цель", destDir,
		"файлов_распаковано", fileCount,
	)

	return nil
}

func CreateTgz(log logger.LoggerInterface, files []string, outputPath string) error {
	return CreateTarGz(log, files, outputPath)
}

func ExtractTgz(log logger.LoggerInterface, tgzPath, destDir string) error {
	return ExtractTarGz(log, tgzPath, destDir)
}

func addToTar(tw *tar.Writer, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(filepath.Dir(filePath), filePath)
	if err != nil {
		return err
	}
	header.Name = relPath

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if info.IsDir() {
		return nil
	}

	_, err = io.Copy(tw, file)
	return err
}
