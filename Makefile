BINARY=pm
MAIN_DIR=./cmd
CONFIG_DIR=./example/configs

# Цели по умолчанию
.DEFAULT_GOAL := help

# Цвета для вывода
GREEN  := \033[0;32m
YELLOW := \033[1;33m
RESET  := \033[0m

help: ## Показать это меню
	@echo "Доступные команды:"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / { printf "  ${GREEN}%-15s${RESET} ${YELLOW}%s${RESET}\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Собрать бинарник
	@echo "${GREEN}Сборка ${BINARY}...${RESET}"
	go build -o ${BINARY} ${MAIN_DIR}/main.go
	@echo "${GREEN}Готово: ./${BINARY}${RESET}"

test: ## Запустить все тесты
	@echo "${GREEN}Запуск тестов...${RESET}"
	go test -v ./...

test-coverage: ## Запустить тесты с отчётом о покрытии
	@echo "${GREEN}Запуск тестов с покрытием...${RESET}"
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "${GREEN}Отчёт о покрытии: coverage.html${RESET}"

deps: ## Установить зависимости
	@echo "${GREEN}Обновление зависимостей...${RESET}"
	go mod tidy
	go mod verify
	@echo "${GREEN}Зависимости готовы${RESET}"

clean: ## Удалить бинарник и артефакты тестов
	@echo "${GREEN}Очистка...${RESET}"
	rm -f ${BINARY}
	rm -f coverage.out coverage.html
	@echo "${GREEN}Готово${RESET}"

create: build ## Собрать и выполнить pm create [config]
	@if [ -z "${CONFIG}" ]; then \
		echo "${YELLOW}Используйте: make create CONFIG=packet.json${RESET}"; \
		exit 1; \
	fi
	@echo "${GREEN}Выполнение: pm create $${CONFIG}${RESET}"
	./${BINARY} create $${CONFIG}

update: build ## Собрать и выполнить pm update [config]
	@if [ -z "${CONFIG}" ]; then \
		echo "${YELLOW}Используйте: make update CONFIG=packages.json${RESET}"; \
		exit 1; \
	fi
	@echo "${GREEN}Выполнение: pm update $${CONFIG}${RESET}"
	./${BINARY} update $${CONFIG}

example-configs: ## Создать примеры конфигов в ./example/configs/
	mkdir -p ${CONFIG_DIR}
	@echo "${GREEN}Создание примеров конфигурации...${RESET}"

	@echo '{
  "name": "app",
  "ver": "1.0",
  "targets": [
    "./test_data/*.txt",
    { "path": "./test_data/*.log", "exclude": "*.tmp" }
  ],
  "packets": [
    { "name": "utils", "ver": "1.5" }
  ]
}' > ${CONFIG_DIR}/packet.json

	@echo '{
  "packages": [
    { "name": "app", "ver": "1.0" },
    { "name": "utils" }
  ]
}' > ${CONFIG_DIR}/packages.json

	@echo "${GREEN}Примеры сохранены в: ${CONFIG_DIR}/${RESET}"

lint: ## Проверить код с помощью golangci-lint
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "${YELLOW}golangci-lint не установлен. Установите: https://golangci-lint.run/usage/install/${RESET}"; \
	fi