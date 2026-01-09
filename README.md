# ggconfig - Go Configuration Generator

Генератор конфигураций для Go-way подхода к управлению конфигурацией.

## Установка

```bash
go install github.com/apopov-app/ggconfig@latest
```

## Параметры генератора

### Основные параметры

- `--interface=Config` - название интерфейса для генерации (обязательный параметр)
- `--output=internal/configs` - путь для создания сгенерированных файлов (опционально, по умолчанию: создает в текущем пакете)
- `--example=configs` - путь для создания примеров YAML файлов (опционально)
- `--registry` - регистрирует конфигурацию в глобальном реестре для использования с `GlobalConfig` (опционально)
- `--alias` - задаёт алиасы для ключей. Повторяемый флаг. Форматы:
  - `env.<Method>=ALIAS1,ALIAS2` — алиасы для переменной окружения метода (например, `env.Host=SERVER_ADDRESS_ALIASE`)
  - `yaml.section=ALIAS1,ALIAS2` — алиасы имени YAML-секции (например, `server` → `svc`)
  - `yaml.key.<Method>=ALIAS1,ALIAS2` — алиасы ключей внутри секции (например, `yaml.key.Host=hostname`)

### Как влияют параметры

#### Без --output (по умолчанию)
```go
//go:generate ggconfig --interface=Config
```
- Создает файл в том же пакете: `internal/db/db.gen.go`
- Пакет: `package db`
- Функции: `NewConfigDbConfig()`, `NewYAMLConfig()`, `NewMockDbConfig()`

#### С --output
```go
//go:generate ggconfig --interface=Config --output=internal/genconfig
```
- Создает файл в указанной папке: `internal/genconfig/db.gen.go`
- Пакет: `package genconfig` (название папки)
- Функции: `NewConfigDbConfig()`, `NewYAMLConfig()`, `NewMockDbConfig()`

#### С --registry
```go
//go:generate ggconfig --interface=Config --output=../ggconfig --registry
```
- Регистрирует конфигурацию в глобальном реестре
- Позволяет использовать `GlobalConfig` для получения конфигурации через `Get<Pkg>()` методы
- Все конфигурации с `--registry` должны использовать один и тот же `--output` путь

#### С --example
```go
//go:generate ggconfig --interface=Config --example=configs
```
- Создает пример YAML файла: `configs/db_example.yaml`
- Комментарии из Go кода переносятся в YAML как комментарии
- Структура YAML соответствует интерфейсу

**Пример YAML файла:**
```yaml
# Example configuration for db package
# Copy this file to config.yaml or use with your application

db:
  # Host returns database host address
  host: ""
  # Port returns database port  
  port: ""
  # User returns database user
  user: ""
  # Password returns database password
  password: ""
  # Name returns database name
  name: ""
  # SSLMode returns SSL mode
  sslmode: ""
```

## Принцип работы

1. **Каждый пакет определяет свой интерфейс конфигурации** - интерфейс `Config` объявляется в пакете, который его использует
2. **Дефолты определяются в пакете** - значения по умолчанию задаются в функции `NewFromConfig` внутри пакета, а не в `main.go`
3. **Генератор создает реализации ENV, YAML и Mock** - автоматически генерируются реализации для разных источников конфигурации
4. **Использование через dependency injection** - конфигурация передается как зависимость в функции инициализации пакета
5. **Методы интерфейса возвращают `(value T, exists bool)`** - для явного указания наличия значения
6. **`main.go` только читает конфигурацию** - точка входа приложения создает конфигурацию из источников (ENV/YAML) и передает её в пакеты

> **💡 Важно**: Дефолты должны быть определены в пакете, который их использует (например, в `db.NewFromConfig`), а не в `main.go`. Это обеспечивает инкапсуляцию и делает код более поддерживаемым.

## Пример использования

### 1. Определяем интерфейс в пакете

```go
// internal/db/config.go
package db

//go:generate ggconfig --interface=Config
type Config interface {
    // Host returns database host address
    Host(defaultValue string) (string, bool)
    // Port returns database port
    Port(defaultValue int) (int, bool)
    // User returns database user
    User(defaultValue string) (string, bool)
    // Password returns database password
    Password(defaultValue string) (string, bool)
    // Name returns database name
    Name(defaultValue string) (string, bool)
    // SSLMode returns SSL mode
    SSLMode(defaultValue string) (string, bool)
}
```

> **💡 Примечание**: Комментарии из Go кода (например, `// Host returns database host address`) автоматически переносятся в сгенерированные YAML примеры как комментарии.

> **⚠️ Важно**: 
> - Параметр `--interface` является обязательным. Если он не указан, генератор завершится с ошибкой.
> - Методы интерфейса должны возвращать `(value T, exists bool)` для поддержки явной проверки наличия значения.

### 2. Генерируем реализацию

```bash
go generate ./...
```

С алиасами:
```go
//go:generate ggconfig --interface=Config --output=internal/gconfig --alias env.Host=SERVER_ADDRESS_ALIASE
```

С регистрацией в глобальном реестре:
```go
//go:generate ggconfig --interface=Config --output=../ggconfig --registry
```

### 3. Используем в коде

**Важно**: Интерфейсы и дефолты определяются в пакете, который их использует. `main.go` только читает конфигурацию и передает её в пакет.

#### Вариант 1: Прямое использование (без GlobalConfig)

```go
// internal/db/connection.go
// Defaults live here (close to where they're used), not in main.
func NewFromConfig(config Config) (*Connection, error) {
    if config == nil {
        return nil, fmt.Errorf("nil config")
    }
    
    // Defaults are defined in the package that uses them
    host, _ := config.Host("localhost")
    port, _ := config.Port("5432")
    user, _ := config.User("postgres")
    password, _ := config.Password("password")
    name, _ := config.Name("example")
    sslMode, _ := config.SSLMode("disable")
    
    dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        host, port, user, password, name, sslMode,
    )
    // ...
}
```

```go
// cmd/main.go
// Main only reads configuration, doesn't define defaults.
func main() {
    configPath := flag.String("config", "", "path to YAML config file (optional)")
    flag.Parse()
    
    var cfg db.Config
    if *configPath != "" {
        yamlData, _ := os.ReadFile(*configPath)
        cfg = db.NewYAMLConfig(yamlData)
    } else {
        cfg = db.NewConfigDbConfig() // ENV
    }
    
    // Package handles defaults internally
    conn, err := db.NewFromConfig(cfg)
    // ...
}
```

#### Вариант 2: Использование GlobalConfig (с --registry)

```go
// internal/db/connection.go
// Defaults live here (close to where they're used), not in main.
func NewFromConfig(config Config) (*Connection, error) {
    if config == nil {
        return nil, fmt.Errorf("nil config")
    }
    
    // Defaults are defined in the package that uses them
    host, _ := config.Host("localhost")
    port, _ := config.Port("5432")
    // ...
}
```

```go
// cmd/api/main.go
package main

import (
    "log"
    "flag"
    
    "your-project/internal/ggconfig"
    "your-project/internal/db"
    "your-project/internal/httpapi"
)

func main() {
    configPath := flag.String("config", "", "path to YAML config file (optional)")
    flag.Parse()
    
    // Main only reads configuration, doesn't define defaults.
    // Defaults are defined in the packages that use them.
    
    // Create GlobalConfig with sources (order matters: ENV → YAML → default)
    global, err := ggconfig.NewGlobalConfig(
        ggconfig.NewEnvConfig(func(key string) string { return key }),
        ggconfig.NewGlobalYamlConfig(*configPath),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Get configurations from registry
    dbCfg, ok := global.GetDb()
    if !ok {
        log.Fatal("db config not registered")
    }
    
    httpCfg, ok := global.GetHttpapi()
    if !ok {
        log.Fatal("httpapi config not registered")
    }
    
    // Packages handle defaults internally
    conn, err := db.NewFromConfig(dbCfg)
    srv, addr, err := httpapi.NewFromConfig(store, httpCfg)
    // ...
}
```

> **💡 Примечание**: Генератор создает конкретные реализации, а не интерфейсы. Это позволяет использовать типизированные методы и избежать лишних абстракций.

## GlobalConfig API

`GlobalConfig` позволяет централизованно управлять конфигурациями нескольких пакетов:

### Создание GlobalConfig

```go
global, err := ggconfig.NewGlobalConfig(
    ggconfig.NewEnvConfig(func(key string) string { return key }),
    ggconfig.NewGlobalYamlConfig("path/to/config.yaml"),
)
```

**Параметры:**
- `NewEnvConfig(mapKey func(string) string)` - источник из переменных окружения. `mapKey` позволяет трансформировать ключи (например, для префиксов).
- `NewGlobalYamlConfig(path string)` - источник из YAML файла. Если путь пустой, YAML не загружается.

**Порядок источников важен:** значения ищутся в порядке перечисления (ENV → YAML → default).

### Получение конфигураций

Для каждого пакета, зарегистрированного с `--registry`, генерируется метод `Get<Pkg>()`:

```go
dbCfg, ok := global.GetDb()        // для пакета "db"
httpCfg, ok := global.GetHttpapi() // для пакета "httpapi"
pgxCfg, ok := global.GetPgx()      // для пакета "pgx"
```

Метод возвращает `(config, bool)`, где `bool` указывает, была ли конфигурация зарегистрирована.

### Структура YAML для GlobalConfig

```yaml
# config.yaml
db:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "secret"
  name: "mydb"
  sslmode: "disable"

httpapi:
  host: "0.0.0.0"
  port: 8080
  adminToken: "admin-secret-token"

pgx:
  databaseURL: "postgres://user:pass@localhost/dbname?sslmode=disable"
```

## Переменные окружения

Генератор автоматически создает ключи переменных окружения:

- `Host` → `DB_HOST`
- `Port` → `DB_PORT` 
- `SSLMode` → `DB_SSL_MODE`
- `ReadTimeout` → `DB_READ_TIMEOUT`

Формат: `<PACKAGE_NAME>_<METHOD_NAME>` (в верхнем регистре).

## Поддерживаемые типы

- `string` - строковые значения
- `int` - целые числа (с автоматическим парсингом)

## Пример проекта

Полные примеры использования находятся в папках `example/` и `example2/`:

### Пример 1: Генерация в том же пакете
```bash
cd example
go generate ./...
go build -o example-app cmd/main.go
./example-app
```

### Пример 2: Генерация в отдельном пакете
```bash
cd example2
go generate ./...
go build -o example2-app cmd/main.go
./example2-app
```

## Преимущества

✅ **Go-way** - интерфейсы + code generation  
✅ **Dependency Injection** - конфиг прокидывается как зависимость  
✅ **Явные дефолты** - дефолты видны в коде использования  
✅ **Типобезопасность** - все через интерфейсы  
✅ **Тестируемость** - легко мокать интерфейсы  
✅ **Гибкость** - можно передавать разные дефолты в разных местах  
✅ **Множественные источники** - ENV, YAML, Mock в одном генераторе  
✅ **Автоматическая установка** - через `go install`  
✅ **Интеграция с go generate** - стандартный Go инструмент  
✅ **Глобальный реестр** - централизованное управление конфигурациями через `GlobalConfig`  
✅ **Явная проверка наличия** - методы возвращают `(value, exists bool)` для контроля источников
