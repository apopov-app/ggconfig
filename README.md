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
- `--name=custom_name` - переопределяет автоматически генерируемое имя пакета (опционально). По умолчанию имя генерируется автоматически на основе пути относительно корня модуля Go для избежания конфликтов
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
- `[]CustomType` - массивы структур (автоматическая сериализация через JSON)

### Работа с массивами структур

Генератор поддерживает методы, возвращающие массивы пользовательских структур:

```go
// internal/server/config.go
package server

//go:generate ggconfig --interface=Config --output=../gconfig --registry

type Config interface {
    // Realms returns list of realm configurations
    Realms(defaultValue []RealmInfo) ([]RealmInfo, bool)
}

type RealmInfo struct {
    ID         string   `yaml:"id" json:"id"`
    ClientHost string   `yaml:"clientHost" json:"clientHost"`
    ClientPort int      `yaml:"clientPort" json:"clientPort"`
    Regions    []string `yaml:"regions" json:"regions"`
    Version    string   `yaml:"version" json:"version"`
}
```

**YAML конфигурация:**
```yaml
internal_server:
  realms:
    - id: "realm-dev-1"
      clientHost: "localhost"
      clientPort: 8080
      regions: ["en", "ru"]
      version: "dev"
    - id: "realm-prod-1"
      clientHost: "api.example.com"
      clientPort: 443
      regions: ["en", "ru", "de", "fr"]
      version: "v1.2.3"
```

**ENV конфигурация:**
```bash
# JSON формат для массивов в переменных окружения
export INTERNAL_SERVER_REALMS='[{"id":"realm-dev-1","clientHost":"localhost","clientPort":8080,"regions":["en","ru"],"version":"dev"}]'
```

**Использование в коде:**
```go
// internal/server/server.go
package server

import "fmt"

type Server struct {
    Host   string
    Port   int
    Realms []RealmInfo
}

func NewFromConfig(cfg Config) (*Server, error) {
    if cfg == nil {
        return nil, fmt.Errorf("config is nil")
    }

    // Дефолтные значения определяются в пакете, который их использует
    host, _ := cfg.Host("localhost")
    port, _ := cfg.Port(8080)
    
    // Для массивов можно передать nil или пустой массив как default
    realms, ok := cfg.Realms(nil)
    if !ok {
        // Если конфигурация не найдена, используем пустой массив
        realms = []RealmInfo{}
    }

    return &Server{
        Host:   host,
        Port:   port,
        Realms: realms,
    }, nil
}
```

```go
// cmd/main.go
package main

import (
    "log"
    "github.com/yourproject/internal/gconfig"
    "github.com/yourproject/internal/server"
)

func main() {
    // Создаем глобальную конфигурацию
    global, err := gconfig.NewGlobalConfig(
        gconfig.NewEnvConfig(func(key string) string { return key }),
        gconfig.NewGlobalYamlConfig("config.yaml"),
    )
    if err != nil {
        log.Fatalf("Failed to create config: %v", err)
    }
    
    // Получаем конфигурацию сервера
    serverCfg, ok := global.GetInternalServer()
    if !ok {
        log.Fatal("server config not registered")
    }
    
    // Создаем сервер с конфигурацией
    srv, err := server.NewFromConfig(serverCfg)
    if err != nil {
        log.Fatalf("Failed to create server: %v", err)
    }
    
    // Используем конфигурацию realms
    log.Printf("Server configured with %d realms:\n", len(srv.Realms))
    for _, realm := range srv.Realms {
        log.Printf("  - %s: %s:%d (regions: %v, version: %s)\n", 
            realm.ID, realm.ClientHost, realm.ClientPort, 
            realm.Regions, realm.Version)
    }
}
```

**Примеры различных источников конфигурации:**

```bash
# 1. Из YAML файла
go run cmd/main.go --config=config.yaml

# 2. Из переменных окружения
export INTERNAL_SERVER_REALMS='[{"id":"realm-1","clientHost":"localhost","clientPort":8080,"regions":["en"],"version":"v1.0.0"}]'
export INTERNAL_SERVER_HOST="0.0.0.0"
export INTERNAL_SERVER_PORT="9000"
go run cmd/main.go

# 3. Комбинированный подход (ENV переопределяет YAML)
# Сначала читаются ENV переменные, затем YAML, затем defaults
export INTERNAL_SERVER_PORT="9000"
go run cmd/main.go --config=config.yaml
```

> **💡 Примечание**: 
> - При генерации в отдельный пакет (с флагом `--output`), генератор автоматически добавляет необходимые импорты для пользовательских типов
> - Массивы в ENV должны быть в JSON формате
> - Структуры должны иметь теги `json` для корректной сериализации/десериализации
> - Порядок источников в `NewGlobalConfig` важен: первый найденный источник с значением будет использован

## Пример проекта

Полные примеры использования находятся в папках `example/`, `example2/`, `example3/` и `example4/`:

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

### Пример 3: Автоматическое разрешение конфликтов
```bash
cd example3
go generate ./...
go build -o abin-app ./cmd/Abin
go build -o bbin-app ./cmd/Bbin
./abin-app
./bbin-app
```

Демонстрирует автоматическое разрешение конфликтов: два пакета `server` в разных местах (`cmd/Abin/internal/server` и `cmd/Bbin/internal/server`) генерируются в одну папку `internal/gconfig` без конфликтов благодаря автоматически сгенерированным уникальным именам.

### Пример 4: Массивы структур
```bash
cd example4
go generate ./...
go run cmd/main.go
```

Демонстрирует работу с массивами структур: конфигурация с поддержкой списка realms, где каждый realm содержит ID, хост, порт, регионы и версию. Показывает автоматическую сериализацию/десериализацию массивов пользовательских типов через JSON для ENV и прямой парсинг из YAML.

## FAQ: Работа с массивами

### Как валидировать массивы?

```go
func NewFromConfig(cfg Config) (*Server, error) {
    realms, ok := cfg.Realms(nil)
    if !ok || len(realms) == 0 {
        return nil, fmt.Errorf("at least one realm is required")
    }
    
    // Валидация каждого элемента
    for i, realm := range realms {
        if realm.ID == "" {
            return nil, fmt.Errorf("realm[%d]: ID is required", i)
        }
        if realm.ClientPort < 1 || realm.ClientPort > 65535 {
            return nil, fmt.Errorf("realm[%d]: invalid port %d", i, realm.ClientPort)
        }
    }
    
    return &Server{Realms: realms}, nil
}
```

### Можно ли использовать массивы примитивных типов?

Да! Поддерживаются массивы любых типов:

```go
type Config interface {
    // Массивы примитивов
    AllowedIPs(defaultValue []string) ([]string, bool)
    Ports(defaultValue []int) ([]int, bool)
    
    // Массивы структур
    Servers(defaultValue []ServerInfo) ([]ServerInfo, bool)
}
```

YAML:
```yaml
myconfig:
  allowedIPs: ["192.168.1.1", "10.0.0.1"]
  ports: [8080, 8443, 9000]
  servers:
    - host: "server1.example.com"
      port: 8080
    - host: "server2.example.com"
      port: 8443
```

### Как обновить конфигурацию без перезапуска?

```go
type Server struct {
    cfg    Config
    realms []RealmInfo
    mu     sync.RWMutex
}

func (s *Server) ReloadConfig() error {
    realms, ok := s.cfg.Realms(nil)
    if !ok {
        return fmt.Errorf("failed to reload realms")
    }
    
    s.mu.Lock()
    s.realms = realms
    s.mu.Unlock()
    
    return nil
}

func (s *Server) GetRealms() []RealmInfo {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    // Возвращаем копию для безопасности
    result := make([]RealmInfo, len(s.realms))
    copy(result, s.realms)
    return result
}
```

### Как тестировать код с массивами?

Используйте Mock конфигурацию:

```go
func TestServerWithRealms(t *testing.T) {
    // Создаем mock конфигурацию
    mockCfg := &MockConfig{
        realms: []server.RealmInfo{
            {
                ID:         "test-realm",
                ClientHost: "localhost",
                ClientPort: 8080,
                Regions:    []string{"en"},
                Version:    "test",
            },
        },
    }
    
    srv, err := server.NewFromConfig(mockCfg)
    if err != nil {
        t.Fatalf("Failed to create server: %v", err)
    }
    
    if len(srv.Realms) != 1 {
        t.Errorf("Expected 1 realm, got %d", len(srv.Realms))
    }
}

type MockConfig struct {
    realms []server.RealmInfo
}

func (m *MockConfig) Realms(defaultValue []server.RealmInfo) ([]server.RealmInfo, bool) {
    if m.realms != nil {
        return m.realms, true
    }
    return defaultValue, false
}

func (m *MockConfig) Host(defaultValue string) (string, bool) {
    return "localhost", true
}

func (m *MockConfig) Port(defaultValue int) (int, bool) {
    return 8080, true
}
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
✅ **Поддержка массивов структур** - автоматическая сериализация/десериализация сложных типов  
✅ **Автоматические импорты** - генератор добавляет необходимые импорты при использовании кастомных типов
