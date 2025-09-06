package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"pm/config"
	"pm/internal/archive"
	"pm/internal/cli"
	"pm/internal/errors"
	"pm/internal/logger"
	"pm/internal/ssh"
	"pm/internal/utils"
	"pm/pkg/version"
)

const maxConcurrentOps = 5

func main() {
	cmd, err := cli.Parse()
	if err != nil {
		log.Fatalf("Ошибка парсинга команды: %v", err)
	}

	logg := logger.NewLogger(cmd.LogLevel)

	switch cmd.Type {
	case cli.Create:
		if err := handleCreate(cmd.ConfigPath, logg); err != nil {
			logg.Error("Ошибка выполнения команды create: %v", err)
			os.Exit(1)
		}
	case cli.Update:
		if err := handleUpdate(cmd.ConfigPath, logg); err != nil {
			logg.Error("Ошибка выполнения команды update: %v", err)
			os.Exit(1)
		}
	default:
		logg.Error("Неизвестная команда: %s", cmd.Type)
		os.Exit(1)
	}
}

func getArchiveExtension(format string) string {
	switch format {
	case "zip":
		return ".zip"
	case "tar.gz", "tgz":
		return ".tar.gz"
	default:
		return "." + format
	}
}

func handleCreate(configPath string, log logger.LoggerInterface) error {
	packet, err := config.LoadPacketConfig(configPath)
	if err != nil {
		log.Error("Ошибка загрузки конфигурации", "путь", configPath, "ошибка", err.Error())
		return err
	}

	files, err := archive.CollectFiles(log, packet.Targets)
	if err != nil {
		log.Error("Ошибка сбора файлов", "ошибка", err.Error())
		return err
	}

	archiveFormat := "zip"
	if packet.Format != "" {
		archiveFormat = packet.Format
		log.Debug("Используется указанный формат архива", "формат", archiveFormat)
	} else {
		log.Debug("Используется формат архива по умолчанию", "формат", archiveFormat)
	}

	extension := getArchiveExtension(archiveFormat)
	archiveName := packet.Name + "-" + packet.Ver + extension

	log.Info("Создание архива", "имя", archiveName, "формат", archiveFormat, "файлов", len(files))

	switch archiveFormat {
	case "zip":
		if err := archive.CreateZip(log, files, archiveName); err != nil {
			log.Error("Ошибка создания ZIP архива", "имя", archiveName, "ошибка", err.Error())
			return err
		}
	case "tar.gz", "tgz":
		if err := archive.CreateTarGz(log, files, archiveName); err != nil {
			log.Error("Ошибка создания tar.gz архива", "имя", archiveName, "ошибка", err.Error())
			return err
		}
	default:
		log.Error("Неподдерживаемый формат архива", "формат", archiveFormat)
		return errors.NewArchiveCreationError(archiveName, files, fmt.Errorf("неподдерживаемый формат архива: %s", archiveFormat))
	}

	log.Info("Архив успешно создан", "имя", archiveName, "формат", archiveFormat)

	user := os.Getenv("PM_SSH_USER")
	host := os.Getenv("PM_SSH_HOST")
	key := os.Getenv("PM_SSH_KEY")
	port := 22
	if p := os.Getenv("PM_SSH_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}
	remotePath := os.Getenv("PM_REMOTE_PATH")
	if remotePath == "" {
		remotePath = "/tmp/pm/"
		log.Debug("Используется путь на сервере по умолчанию", "путь", remotePath)
	}
	remoteFile := remotePath + archiveName

	if user == "" || host == "" || key == "" {
		log.Info("SSH не настроен. Архив сохранён локально", "путь", archiveName)
		return nil
	}

	log.Debug("Подключение к SSH серверу", "хост", host, "пользователь", user, "порт", port)
	client, err := ssh.NewClient(user, host, key, port)
	if err != nil {
		log.Error("Ошибка подключения к SSH серверу", "хост", host, "ошибка", err.Error())
		return err
	}
	defer client.Close()

	log.Debug("Загрузка архива на сервер", "локальный_файл", archiveName, "удаленный_файл", remoteFile)
	if err := client.Upload(archiveName, remoteFile); err != nil {
		log.Error("Ошибка загрузки архива на сервер", "файл", archiveName, "ошибка", err.Error())
		return err
	}
	log.Info("Архив успешно загружен", "файл", archiveName, "хост", host, "путь", remoteFile)

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentOps)
	errs := make(chan error, len(packet.Packets))

	for _, dep := range packet.Packets {
		wg.Add(1)
		go func(dep config.Packet) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			depArchiveFormat := "zip"
			if dep.Format != "" {
				depArchiveFormat = dep.Format
			}

			extension := getArchiveExtension(depArchiveFormat)
			depName := dep.Name + "-" + dep.Ver + extension
			remoteDepPath := remotePath + depName

			log.Debug("Загрузка зависимости", "имя", dep.Name, "версия", dep.Ver, "формат", depArchiveFormat)
			if err := client.Upload(depName, remoteDepPath); err != nil {
				log.Error("Ошибка загрузки зависимости", "имя", dep.Name, "версия", dep.Ver, "ошибка", err.Error())
				errs <- fmt.Errorf("ошибка загрузки зависимости %s: %w", depName, err)
				return
			}
			log.Info("Зависимость успешно загружена", "имя", dep.Name, "версия", dep.Ver, "путь", remoteDepPath)
		}(dep)
	}

	wg.Wait()
	close(errs)

	var uploadErrors []error
	for err := range errs {
		uploadErrors = append(uploadErrors, err)
	}

	if len(uploadErrors) > 0 {
		log.Warn("Ошибки при загрузке зависимостей", "количество", len(uploadErrors))
		for _, e := range uploadErrors {
			log.Error("Ошибка загрузки", "ошибка", e.Error())
		}
	}

	return nil
}

func handleUpdate(configPath string, log logger.LoggerInterface) error {
	log.Debug("Загрузка конфигурации", "путь", configPath)
	pkgs, err := config.LoadPackagesConfig(configPath)
	if err != nil {
		log.Error("Ошибка загрузки конфигурации", "путь", configPath, "ошибка", err.Error())
		return err
	}

	user := os.Getenv("PM_SSH_USER")
	host := os.Getenv("PM_SSH_HOST")
	key := os.Getenv("PM_SSH_KEY")
	port := 22
	if p := os.Getenv("PM_SSH_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}
	remotePath := os.Getenv("PM_REMOTE_PATH")
	if remotePath == "" {
		remotePath = "/tmp/pm/"
		log.Debug("Используется путь на сервере по умолчанию", "путь", remotePath)
	}

	if user == "" || host == "" || key == "" {
		log.Error("SSH конфигурация не задана")
		return errors.ErrInvalidSSHConfig
	}

	log.Debug("Подключение к SSH серверу", "хост", host, "пользователь", user, "порт", port)
	client, err := ssh.NewClient(user, host, key, port)
	if err != nil {
		log.Error("Ошибка подключения к SSH серверу", "хост", host, "ошибка", err.Error())
		return err
	}
	defer client.Close()

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentOps)
	errs := make(chan error, len(pkgs.Packages))

	for _, pkg := range pkgs.Packages {
		wg.Add(1)
		go func(pkg config.Packet) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			remoteDir := remotePath
			if !strings.HasSuffix(remoteDir, "/") {
				remoteDir += "/"
			}

			log.Debug("Чтение удаленной директории", "путь", remoteDir)
			files, err := client.ReadDir(remoteDir)
			if err != nil {
				server := host
				if server == "" {
					server = "unknown"
				}
				log.Error("Ошибка чтения удаленной директории", "путь", remoteDir, "ошибка", err.Error())
				errs <- errors.NewSSHConnectionError(server, err)
				return
			}

			log.Debug("Найдено файлов в удаленной директории", "количество", len(files), "путь", remoteDir)

			found := false
			for _, f := range files {
				if f.IsDir() {
					continue
				}

				filename := f.Name()
				extension := ""
				format := ""

				// Определяем формат по расширению
				if strings.HasSuffix(filename, ".zip") {
					extension = ".zip"
					format = "zip"
				} else if strings.HasSuffix(filename, ".tar.gz") {
					extension = ".tar.gz"
					format = "tar.gz"
				} else if strings.HasSuffix(filename, ".tgz") {
					extension = ".tgz"
					format = "tar.gz"
				} else {
					continue
				}

				// Удаляем расширение для извлечения версии
				baseName := strings.TrimSuffix(filename, extension)
				versionStr := utils.ExtractVersion(baseName)

				if !strings.HasPrefix(baseName, pkg.Name+"-") {
					continue
				}

				if versionStr == "" {
					log.Debug("Не удалось извлечь версию из файла", "файл", filename)
					continue
				}

				if pkg.Ver != "" {
					log.Debug("Проверка версии", "имя", pkg.Name, "требуемая", pkg.Ver, "найденная", versionStr)
					matches, err := version.Matches(versionStr, pkg.Ver)
					if err != nil {
						log.Error("Ошибка проверки версии", "имя", pkg.Name, "версия", versionStr, "условие", pkg.Ver, "ошибка", err.Error())
						errs <- fmt.Errorf("ошибка проверки версии для %s: %w", pkg.Name, err)
						return
					}
					if !matches {
						log.Debug("Версия не соответствует условию", "файл", filename, "требуется", pkg.Ver, "имеется", versionStr)
						continue
					}
				}

				remoteFile := remotePath + filename
				localFile := "./" + filename

				log.Info("Найден подходящий пакет", "имя", pkg.Name, "версия", versionStr, "формат", format, "файл", filename)

				log.Debug("Скачивание пакета", "удаленный_файл", remoteFile, "локальный_файл", localFile)
				if err := client.Download(remoteFile, localFile); err != nil {
					log.Error("Ошибка скачивания пакета", "имя", pkg.Name, "файл", filename, "ошибка", err.Error())
					errs <- errors.NewSSHFileTransferError(host, remoteFile, localFile, err)
					return
				}

				log.Debug("Распаковка пакета", "файл", localFile, "формат", format)
				switch format {
				case "zip":
					if err := archive.ExtractZip(log, localFile, "./"); err != nil {
						log.Error("Ошибка распаковки ZIP архива", "файл", localFile, "ошибка", err.Error())
						errs <- errors.NewArchiveExtractionError(localFile, "./", err)
						return
					}
				case "tar.gz":
					if err := archive.ExtractTarGz(log, localFile, "./"); err != nil {
						log.Error("Ошибка распаковки tar.gz архива", "файл", localFile, "ошибка", err.Error())
						errs <- errors.NewArchiveExtractionError(localFile, "./", err)
						return
					}
				default:
					log.Error("Неподдерживаемый формат архива", "формат", format, "файл", filename)
					errs <- fmt.Errorf("неподдерживаемый формат архива: %s", format)
					return
				}

				log.Info("Пакет успешно установлен", "имя", pkg.Name, "версия", versionStr, "формат", format)
				found = true
				break
			}

			if !found {
				errMsg := fmt.Sprintf("не найден подходящий пакет: %s (условие: %s)", pkg.Name, pkg.Ver)
				log.Error("Не найден подходящий пакет", "имя", pkg.Name, "условие", pkg.Ver)
				errs <- fmt.Errorf(errMsg)
			}
		}(pkg)
	}

	wg.Wait()
	close(errs)

	var allErrors []error
	for err := range errs {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		log.Error("Ошибки при установке пакетов", "количество", len(allErrors))
		for i, err := range allErrors {
			log.Error("Ошибка установки пакета", "номер", i+1, "ошибка", err.Error())
		}
		return fmt.Errorf("ошибок при установке: %d, первая: %w", len(allErrors), allErrors[0])
	}

	return nil
}
