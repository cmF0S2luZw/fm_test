// internal/cli/parser.go
package cli

import (
	"os"
	"strings"

	"pm/internal/errors"
	"pm/internal/logger"

	"github.com/alecthomas/kingpin/v2"
)

type CommandType string

const (
	Create CommandType = "create"
	Update CommandType = "update"
)

type ParsedCommand struct {
	Type       CommandType
	ConfigPath string
	LogLevel   string
}

func Parse() (*ParsedCommand, error) {
	app := kingpin.New("pm", "Пакетный менеджер для работы с архивами")
	app.Version("pm v0.1.0")
	app.HelpFlag.Short('h')

	logLevel := app.Flag("log-level", "Уровень логирования").
		Default(logger.LevelInfo).
		Enum(
			logger.LevelDebug,
			logger.LevelInfo,
			logger.LevelWarn,
			logger.LevelError,
		)

	createCmd := app.Command(string(Create), "Упаковать файлы в архив")
	createConfig := createCmd.Arg("config", "Путь к packet.json или packet.yaml").Required().ExistingFile()

	updateCmd := app.Command(string(Update), "Скачать и распаковать пакеты")
	updateConfig := updateCmd.Arg("config", "Путь к packages.json").Required().ExistingFile()

	cmd, err := app.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	switch cmd {
	case string(Create):
		return &ParsedCommand{
			Type:       Create,
			ConfigPath: *createConfig,
			LogLevel:   strings.ToLower(*logLevel),
		}, nil
	case string(Update):
		return &ParsedCommand{
			Type:       Update,
			ConfigPath: *updateConfig,
			LogLevel:   strings.ToLower(*logLevel),
		}, nil
	default:
		return nil, &errors.UnknownCommandError{Command: cmd}
	}
}
