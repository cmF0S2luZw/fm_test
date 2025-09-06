package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"

	"pm/internal/errors"
	"pm/internal/logger"
)

type Target struct {
	Path    string `json:"path"`
	Exclude string `json:"exclude,omitempty"`
}

func CollectFiles(log logger.LoggerInterface, targets []Target) ([]string, error) {
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
