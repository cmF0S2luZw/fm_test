// cmd/main.go
package main

import (
	"fmt"
	"os"

	"pm/internal/cli"
	"pm/internal/logger"
)

func main() {
	baseLogger := logger.NewBaseLogger()

	cmd, err := cli.Parse()
	if err != nil {
		baseLogger.Error("Ошибка парсинга команды", "error", err.Error())
		os.Exit(1)
	}

	log := logger.NewLogger(cmd.LogLevel)
	log.Info("Запуск пакетного менеджера",
		"версия", "0.1.0",
		"команда", string(cmd.Type),
		"конфиг", cmd.ConfigPath,
	)

	switch cmd.Type {
	case cli.Create:
		if err := handleCreate(log, cmd.ConfigPath); err != nil {
			log.Error("Создание архива завершилось ошибкой", "error", err.Error())
			os.Exit(1)
		}
	case cli.Update:
		if err := handleUpdate(log, cmd.ConfigPath); err != nil {
			log.Error("Обновление пакетов завершилось ошибкой", "error", err.Error())
			os.Exit(1)
		}
	}
}

func handleCreate(log *logger.Logger, configPath string) error {
	fmt.Printf("📦 Создание архива из %s...\n", configPath)

	// Здесь будет ваша логика:
	// 1. Чтение packet.json
	// 2. Сбор файлов через archive.CollectFiles
	// 3. Создание ZIP через archive.CreateZip
	// 4. Отправка по SSH

	fmt.Println("✅ Архив успешно создан (заглушка)")
	return nil
}

func handleUpdate(log *logger.Logger, configPath string) error {
	fmt.Printf("🔄 Обновление пакетов из %s...\n", configPath)

	// Здесь будет ваша логика:
	// 1. Чтение packages.json
	// 2. Скачивание архивов по SSH
	// 3. Распаковка через archive.ExtractZip

	fmt.Println("✅ Пакеты успешно обновлены (заглушка)")
	return nil
}
