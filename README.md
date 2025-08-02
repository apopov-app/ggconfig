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

1. **Каждый пакет определяет свой интерфейс конфигурации**
2. **Генератор создает реализации ENV, YAML и Mock в одном файле**
3. **Использование через dependency injection**

## Пример использования

### 1. Определяем интерфейс в пакете

```go
// internal/db/config.go
package db

//go:generate ggconfig --interface=Config
type Config interface {
    // Host returns database host address
    Host(defaultValue string) string
    // Port returns database port
    Port(defaultValue string) string
    // User returns database user
    User(defaultValue string) string
    // Password returns database password
    Password(defaultValue string) string
    // Name returns database name
    Name(defaultValue string) string
    // SSLMode returns SSL mode
    SSLMode(defaultValue string) string
}
```

> **💡 Примечание**: Комментарии из Go кода (например, `// Host returns database host address`) автоматически переносятся в сгенерированные YAML примеры как комментарии.

> **⚠️ Важно**: Параметр `--interface` является обязательным. Если он не указан, генератор завершится с ошибкой.

### 2. Генерируем реализацию

```bash
go generate ./...
```

### 3. Используем в коде

```go
// internal/db/connection.go
func NewConnection(config Config) (*Database, error) {
    dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        config.Host("localhost"),      // ← здесь прокидываем дефолт
        config.Port("5432"),
        config.User("postgres"),
        config.Password("password"),
        config.Name("agentschat"),
        config.SSLMode("disable"),
    )
    // ...
}
```

### 4. В main.go

```go
func main() {
    dbConfig := db.NewConfigDbConfig()  // ← получаем ENV реализацию
    db, err := db.NewConnection(dbConfig)  // ← прокидываем как зависимость
    // ...
}
```

> **💡 Примечание**: Генератор создает конкретные реализации, а не интерфейсы. Это позволяет использовать типизированные методы и избежать лишних абстракций.





## Переменные окружения

Генератор автоматически создает ключи переменных окружения:

- `Host` → `DB_HOST`
- `Port` → `DB_PORT` 
- `SSLMode` → `DB_SSL_MODE`
- `ReadTimeout` → `DB_READ_TIMEOUT`

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

 