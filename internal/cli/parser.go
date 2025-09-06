package cli

import (
	"os"
	"strings"

	"pm/internal/errors"

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
		Default("info").
		Enum("debug", "info", "warn", "error")

	createCmd := app.Command(string(Create), "Упаковать файлы в архив")
	createConfig := createCmd.Arg("config", "Путь к packet.json или packet.yaml").Required().ExistingFile()

	updateCmd := app.Command(string(Update), "Скачать и распаковать пакеты")
	updateConfig := updateCmd.Arg("config", "Путь к packages.json").Required().ExistingFile()

	cmd, err := app.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	normalizedLevel := strings.ToLower(*logLevel)

	switch cmd {
	case string(Create):
		return &ParsedCommand{
			Type:       Create,
			ConfigPath: *createConfig,
			LogLevel:   normalizedLevel,
		}, nil
	case string(Update):
		return &ParsedCommand{
			Type:       Update,
			ConfigPath: *updateConfig,
			LogLevel:   normalizedLevel,
		}, nil
	default:
		if cmd == "" {
			return nil, errors.ErrUnknownCommand
		}
		return nil, &errors.UnknownCommandError{Command: cmd}
	}
}
