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

func handleCreate(configPath string, log logger.LoggerInterface) error {
	packet, err := config.LoadPacketConfig(configPath)
	if err != nil {
		return err
	}

	files, err := archive.CollectFiles(log, packet.Targets)
	if err != nil {
		return err
	}

	archiveName := packet.Name + "-" + packet.Ver + ".zip"
	if err := archive.CreateZip(log, files, archiveName); err != nil {
		return err
	}
	log.Info("Архив создан: %s", archiveName)

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
	}
	remoteFile := remotePath + archiveName

	if user == "" || host == "" || key == "" {
		log.Info("SSH не настроен. Архив сохранён локально: %s", archiveName)
		return nil
	}

	client, err := ssh.NewClient(user, host, key, port, log)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Upload(archiveName, remoteFile); err != nil {
		return err
	}
	log.Info("Загружено: %s -> %s:%s", archiveName, host, remoteFile)

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentOps)
	errs := make(chan error, len(packet.Packets))

	for _, dep := range packet.Packets {
		wg.Add(1)
		go func(dep config.Packet) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			depName := dep.Name + "-" + dep.Ver + ".zip"
			remoteDepPath := remotePath + depName

			if err := client.Upload(depName, remoteDepPath); err != nil {
				errs <- fmt.Errorf("ошибка загрузки зависимости %s: %w", depName, err)
				return
			}
			log.Info("Загружена зависимость: %s", depName)
		}(dep)
	}

	wg.Wait()
	close(errs)

	var uploadErrors []error
	for err := range errs {
		uploadErrors = append(uploadErrors, err)
	}

	if len(uploadErrors) > 0 {
		log.Warn("Ошибки при загрузке зависимостей: %d", len(uploadErrors))
		for _, e := range uploadErrors {
			log.Error("%v", e)
		}
	}

	return nil
}

func handleUpdate(configPath string, log logger.LoggerInterface) error {
	pkgs, err := config.LoadPackagesConfig(configPath)
	if err != nil {
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
	}

	if user == "" || host == "" || key == "" {
		return errors.ErrInvalidSSHConfig
	}

	client, err := ssh.NewClient(user, host, key, port, log)
	if err != nil {
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

			// ✅ ПРАВИЛЬНЫЙ ВЫЗОВ: используем интерфейсный метод ReadDir
			files, err := client.ReadDir(remoteDir)
			if err != nil {
				log.Error("Ошибка чтения удалённой директории: %v", err)
				errs <- errors.NewSSHConnectionError(host, err)
				return
			}

			found := false
			for _, f := range files {
				if f.IsDir() || !strings.HasSuffix(f.Name(), ".zip") {
					continue
				}

				if !strings.HasPrefix(f.Name(), pkg.Name+"-") {
					continue
				}

				versionStr := utils.ExtractVersion(f.Name())
				if versionStr == "" {
					log.Debug("Не удалось извлечь версию из файла", "файл", f.Name())
					continue
				}

				if pkg.Ver != "" {
					matches, err := version.Matches(versionStr, pkg.Ver)
					if err != nil {
						log.Warn("Ошибка проверки версии %s для %s: %v", versionStr, pkg.Name, err)
						continue
					}
					if !matches {
						log.Debug("Версия не соответствует условию", "файл", f.Name(), "требуется", pkg.Ver, "имеется", versionStr)
						continue
					}
				}

				remoteFile := remotePath + f.Name()
				localFile := "./" + f.Name()

				log.Info("Найден подходящий пакет: %s (версия %s)", f.Name(), versionStr)

				if err := client.Download(remoteFile, localFile); err != nil {
					log.Error("Ошибка скачивания пакета", "файл", f.Name(), "ошибка", err.Error())
					errs <- errors.NewSSHFileTransferError(host, remoteFile, localFile, err)
					return
				}

				if err := archive.ExtractZip(log, localFile, "./"); err != nil {
					log.Error("Ошибка распаковки пакета", "файл", f.Name(), "ошибка", err.Error())
					errs <- errors.NewArchiveExtractionError(localFile, "./", err)
					return
				}

				log.Info("Пакет успешно установлен", "имя", pkg.Name, "версия", versionStr)
				found = true
				break
			}

			if !found {
				errMsg := fmt.Sprintf("не найден подходящий пакет: %s (условие: %s)", pkg.Name, pkg.Ver)
				log.Error(errMsg)
				errs <- fmt.Errorf(errMsg)
			}
		}(pkg)
	}

	wg.Wait()
	close(errs)

	var finalErr error
	for err := range errs {
		log.Error("Ошибка при установке пакета: %v", err)
		finalErr = err
	}

	return finalErr
}
