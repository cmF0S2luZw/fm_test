# 📦 Пакетный менеджер `pm`

Реализация пакетного менеджера на Go, способного:
- Упаковывать файлы в архив по маске из `.json` или `.yaml`
- Загружать архивы на сервер по SSH
- Скачивать и распаковывать пакеты

## ⚙️ Настройка SSH (обязательно)

Установите переменные окружения:

```bash
export PM_SSH_USER="ваш_пользователь"
export PM_SSH_HOST="ваш.сервер.com"
export PM_SSH_KEY="$HOME/.ssh/id_rsa"
export PM_SSH_PORT="22"
export PM_REMOTE_PATH="/tmp/pm/"
```

> 🔐 Убедитесь, что SSH-ключ не требует пароля или добавлен в `ssh-agent`.

На сервере создайте папку:
```bash
ssh $PM_SSH_USER@$PM_SSH_HOST "mkdir -p $PM_REMOTE_PATH"
```

---

## 📄 Формат конфигов

### Пример: `packet.json` (для упаковки)

```json
{
  "name": "app",
  "ver": "1.0",
  "targets": [
    "./test_data/*.txt",
    { "path": "./test_data/*.log", "exclude": "*.tmp" }
  ],
  "packets": [
    { "name": "utils", "ver": ">=1.5" }
  ]
}
```

### Пример: `packages.json` (для установки)

```json
{
  "packages": [
    { "name": "app", "ver": ">=1.0" },
    { "name": "utils" }
  ]
}
```

Поддерживаемые форматы: `.json`, `.yaml`, `.yml`

---

## 🧰 Команды из тестового задания

### `pm create ./packet.json` — упаковать и загрузить

```bash
./pm create ./packet.json
```

или с `make`:

```bash
make create CONFIG=packet.json
```

**Что делает:**
1. Собирает файлы по маскам из `targets`
2. Исключает файлы по `exclude`
3. Упаковывает в `app-1.0.zip`
4. Загружает на сервер в `$PM_REMOTE_PATH`

---

### `pm update ./packages.json` — скачать и распаковать

```bash
./pm update ./packages.json
```

или:

```bash
make update CONFIG=packages.json
```

**Что делает:**
1. Для каждого пакета формирует имя архива (например, `app-1.0.zip`)
2. Скачивает с сервера по SSH
3. Распаковывает в текущую директорию

---

## 🛠 Makefile: Удобные команды

| Команда | Описание |
|--------|--------|
| `make` | Показать справку |
| `make build` | Собрать `pm` |
| `make test` | Запустить тесты |
| `make test-coverage` | Покрытие кода (`coverage.html`) |
| `make deps` | Обновить зависимости |
| `make clean` | Удалить `pm` и артефакты |
| `make create CONFIG=file.json` | Выполнить `pm create` |
| `make update CONFIG=file.json` | Выполнить `pm update` |
| `make example-configs` | Создать примеры конфигов |
| `make lint` | Проверить код (если установлен `golangci-lint`) |

---

## 🧪 Пример использования

### 1. Создайте тестовые данные

```bash
mkdir -p test_data
echo "Hello" > test_data/file1.txt
echo "Temp" > test_data/temp.log.tmp
echo "Log" > test_data/app.log
```

### 2. Создайте `packet.json`

```json
{
  "name": "test-pkg",
  "ver": "1.0",
  "targets": [
    "./test_data/*.txt",
    { "path": "./test_data/*.log", "exclude": "*.tmp" }
  ]
}
```

### 3. Упакуйте и загрузите

```bash
export PM_SSH_USER=ubuntu
export PM_SSH_HOST=192.168.1.100
export PM_SSH_KEY=~/.ssh/id_rsa
export PM_REMOTE_PATH=/tmp/pm/

./pm create packet.json
```

> Архив `test-pkg-1.0.zip` будет загружен на сервер.

### 4. Скачайте и распакуйте

Создайте `packages.json`:

```json
{
  "packages": [
    { "name": "test-pkg", "ver": "1.0" }
  ]
}
```

Запустите:

```bash
./pm update packages.json
```

> Архив будет скачан и распакован в текущую папку.
