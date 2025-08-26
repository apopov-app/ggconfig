package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Method struct {
	Name       string
	ParamType  string
	ReturnType string
	Comment    string // Добавляем поле для комментария
}

type InterfaceInfo struct {
	PackageName   string
	InterfaceName string
	Methods       []Method
}

// Настройки алиасов, передаваемые через --alias
type AliasSettings struct {
	// ENV: MethodName -> []EnvVarAlias (полные имена переменных окружения)
	Env map[string][]string
	// YAML: алиасы имени секции (например, server_name, svc)
	YAMLSection []string
	// YAML: MethodName -> []KeyAlias внутри секции
	YAMLKey map[string][]string
}

// Поддержка повторяющегося флага --alias
type aliasFlag []string

func (a *aliasFlag) String() string {
	return strings.Join(*a, ",")
}

func (a *aliasFlag) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func main() {
	interfaceName := flag.String("interface", "", "interface name")
	outputPath := flag.String("output", "", "output directory path")
	examplePath := flag.String("example", "", "generate example config file")
	var aliasFlags aliasFlag
	flag.Var(&aliasFlags, "alias", "alias mapping: env.<Method>=ALIAS1,ALIAS2 | yaml.section=ALIAS1,ALIAS2 | yaml.key.<Method>=ALIAS1,ALIAS2")
	flag.Parse()

	// Автоматически определяем пакет из текущей директории
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get current directory: %v", err)
	}

	packageName := filepath.Base(currentDir)
	fmt.Printf("Auto-detected package: %s\n", packageName)

	if interfaceName == nil || *interfaceName == "" {
		log.Fatalf("interface name is required")
	}
	fmt.Printf("Generating config for package: %s, interface: %s\n", packageName, *interfaceName)

	// Парсим интерфейс
	info, err := parseInterface(packageName, *interfaceName)
	if err != nil {
		log.Fatalf("failed to parse interface: %v", err)
	}

	// Парсим алиасы
	aliasSettings := parseAliasSettings(aliasFlags)

	fmt.Printf("Found %d methods in interface\n", len(info.Methods))
	for _, method := range info.Methods {
		fmt.Printf("  - %s(%s) %s\n", method.Name, method.ParamType, method.ReturnType)
	}

	// Генерируем все реализации в одном файле
	if err := generateImplementation(info, aliasSettings, *outputPath); err != nil {
		log.Fatalf("failed to generate implementation: %v", err)
	}

	// Генерируем пример конфига если указан путь
	if *examplePath != "" {
		if err := generateExampleConfig(info, *examplePath); err != nil {
			log.Fatalf("failed to generate example config: %v", err)
		}
	}

	outputDisplayPath := *outputPath
	if outputDisplayPath == "" {
		outputDisplayPath = "current package"
	}
	fmt.Printf("✅ Generated config for %s.%s in %s\n", info.PackageName, info.InterfaceName, outputDisplayPath)
}

func parseInterface(packageName, interfaceName string) (*InterfaceInfo, error) {
	// Парсим весь пакет - путь относительно папки с директивой
	packagePath := filepath.Join("..", packageName)

	fmt.Printf("Parsing package: %s\n", packagePath)

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, packagePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse package %s: %w", packagePath, err)
	}

	var methods []Method

	// Ищем интерфейс во всех файлах пакета
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				if typeDecl, ok := n.(*ast.TypeSpec); ok {
					if typeDecl.Name.Name == interfaceName {
						if interfaceType, ok := typeDecl.Type.(*ast.InterfaceType); ok {
							for _, method := range interfaceType.Methods.List {
								if funcType, ok := method.Type.(*ast.FuncType); ok {
									methodName := method.Names[0].Name
									paramType, returnType := getMethodSignature(funcType)

									// Извлекаем комментарий из документации
									comment := ""
									if method.Doc != nil && len(method.Doc.List) > 0 {
										comment = strings.TrimSpace(strings.TrimPrefix(method.Doc.List[0].Text, "//"))
									}

									methods = append(methods, Method{
										Name:       methodName,
										ParamType:  paramType,
										ReturnType: returnType,
										Comment:    comment,
									})
								}
							}
						}
					}
				}
				return true
			})
		}
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("interface %s not found in package %s", interfaceName, packageName)
	}

	return &InterfaceInfo{
		PackageName:   packageName,
		InterfaceName: interfaceName,
		Methods:       methods,
	}, nil
}

func getReturnType(funcType *ast.FuncType) string {
	if funcType.Results != nil && len(funcType.Results.List) > 0 {
		// Для простоты берем только первый возвращаемый тип
		if ident, ok := funcType.Results.List[0].Type.(*ast.Ident); ok {
			return ident.Name
		}
		// Можно добавить поддержку других типов
	}
	return "string"
}

func getEnvValue(envKey, defaultValue, returnType string) string {
	switch returnType {
	case "int":
		return fmt.Sprintf(`if value := os.Getenv("%s"); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue`, envKey)
	case "string":
		return fmt.Sprintf(`if value := os.Getenv("%s"); value != "" {
		return value
	}
	return defaultValue`, envKey)
	default:
		return fmt.Sprintf(`if value := os.Getenv("%s"); value != "" {
		return value
	}
	return defaultValue`, envKey)
	}
}

// Генерирует фрагмент кода проверки ENV по конкретному ключу без возврата default
func getEnvCheckSnippet(envKey, returnType string) string {
	switch returnType {
	case "int":
		return fmt.Sprintf(`if value := os.Getenv("%s"); value != "" {
    if intValue, err := strconv.Atoi(value); err == nil {
        return intValue
    }
}`, envKey)
	case "string":
		return fmt.Sprintf(`if value := os.Getenv("%s"); value != "" {
    return value
}`, envKey)
	default:
		return fmt.Sprintf(`if value := os.Getenv("%s"); value != "" {
    return value
}`, envKey)
	}
}

// Парсинг повторяющихся флагов --alias
// Допустимые формы:
// - env.<Method>=ALIAS1,ALIAS2
// - yaml.section=ALIAS1,ALIAS2
// - yaml.key.<Method>=ALIAS1,ALIAS2
func parseAliasSettings(flags aliasFlag) AliasSettings {
	settings := AliasSettings{
		Env:         map[string][]string{},
		YAMLSection: []string{},
		YAMLKey:     map[string][]string{},
	}

	for _, item := range flags {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			continue
		}
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		var values []string
		if right != "" {
			for _, v := range strings.Split(right, ",") {
				vv := strings.TrimSpace(v)
				if vv != "" {
					values = append(values, vv)
				}
			}
		}

		segs := strings.Split(left, ".")
		if len(segs) == 0 {
			continue
		}
		switch segs[0] {
		case "env":
			if len(segs) == 2 {
				method := segs[1]
				if len(values) > 0 {
					settings.Env[method] = append(settings.Env[method], values...)
				}
			}
		case "yaml":
			if len(segs) >= 2 {
				switch segs[1] {
				case "section":
					settings.YAMLSection = append(settings.YAMLSection, values...)
				case "key":
					if len(segs) == 3 {
						method := segs[2]
						if len(values) > 0 {
							settings.YAMLKey[method] = append(settings.YAMLKey[method], values...)
						}
					}
				}
			}
		}
	}

	return settings
}

func getMethodSignature(funcType *ast.FuncType) (string, string) {
	// Получаем тип параметра (для простоты берем первый)
	var paramType string
	if funcType.Params != nil && len(funcType.Params.List) > 0 {
		if ident, ok := funcType.Params.List[0].Type.(*ast.Ident); ok {
			paramType = ident.Name
		}
	}

	// Получаем возвращаемый тип
	returnType := getReturnType(funcType)

	return paramType, returnType
}

func generateImplementation(info *InterfaceInfo, aliases AliasSettings, outputPath string) error {
	// Определяем путь для генерации
	var fullOutputPath string
	var packageName string
	var isSamePackage bool

	if outputPath == "" {
		// По умолчанию - создаем в текущем пакете
		fullOutputPath = "."
		packageName = info.PackageName
		isSamePackage = true
	} else {
		// Пользователь указал свой путь
		fullOutputPath = outputPath
		packageName = filepath.Base(outputPath)
		isSamePackage = false
	}

	// Создаем директорию если не существует
	if err := os.MkdirAll(fullOutputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Генерируем один файл со всеми реализациями
	fileName := fmt.Sprintf("%s.gen.go", info.PackageName)
	filePath := filepath.Join(fullOutputPath, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()

	// Шаблон для генерации всех реализаций
	tmpl := template.Must(template.New("config").Funcs(template.FuncMap{
		"title":  strings.Title,
		"envKey": func(methodName string) string { return getEnvKey(info.PackageName, methodName) },
		// Проверка ENV по ключу без возврата default
		"envCheck": func(returnType, key string) string { return getEnvCheckSnippet(key, returnType) },
		// Возврат ENV по основному ключу с fallback на default
		"envReturn": func(returnType, key string) string { return getEnvValue(key, "defaultValue", returnType) },
		"hasIntType": func(methods []Method) bool {
			for _, method := range methods {
				if method.ReturnType == "int" {
					return true
				}
			}
			return false
		},
		"toLower": strings.ToLower,
		// Алиасы
		"envAliasKeys": func(methodName string) []string {
			if aliases.Env == nil {
				return nil
			}
			return aliases.Env[methodName]
		},
		"yamlSectionAliases": func() []string { return aliases.YAMLSection },
		"yamlKeyAliases": func(methodName string) []string {
			if aliases.YAMLKey == nil {
				return nil
			}
			return aliases.YAMLKey[methodName]
		},
		"yamlAssertType": func(returnType string) string {
			switch returnType {
			case "int":
				return "int"
			default:
				return "string"
			}
		},
		"sentinelValue": func(returnType string) string {
			switch returnType {
			case "int":
				return "-2147483648"
			default:
				return "\"__GGCONFIG_SENTINEL__\""
			}
		},
	}).Parse(unifiedTemplate))

	data := struct {
		PackageName    string
		InterfaceName  string
		Methods        []Method
		GenPackageName string
		IsSamePackage  bool
	}{
		PackageName:    info.PackageName,
		InterfaceName:  info.InterfaceName,
		Methods:        info.Methods,
		GenPackageName: packageName,
		IsSamePackage:  isSamePackage,
	}

	return tmpl.Execute(file, data)
}

func generateExampleConfig(info *InterfaceInfo, examplePath string) error {
	// Создаем директорию если не существует
	var fullOutputPath string
	if examplePath == "" {
		// По умолчанию - создаем в текущем пакете
		fullOutputPath = "."
	} else {
		// Пользователь указал свой путь - путь относительно корня проекта
		// Нужно подняться на два уровня вверх от internal/database или internal/server
		fullOutputPath = filepath.Join("..", "..", examplePath)
	}

	if err := os.MkdirAll(fullOutputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Генерируем файл с именованием originalfile.yaml.go
	fileName := fmt.Sprintf("%s_example.yaml", info.PackageName)
	filePath := filepath.Join(fullOutputPath, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()

	// Шаблон для генерации моков
	tmpl := template.Must(template.New("example").Funcs(template.FuncMap{
		"title":  strings.Title,
		"envKey": func(methodName string) string { return getEnvKey(info.PackageName, methodName) },
		"defaultValue": func(paramType string) string {
			switch paramType {
			case "string":
				return "\"\""
			case "int":
				return "0"
			default:
				return "\"\""
			}
		},
	}).Parse(exampleTemplate))

	data := struct {
		PackageName   string
		InterfaceName string
		Methods       []Method
	}{
		PackageName:   info.PackageName,
		InterfaceName: info.InterfaceName,
		Methods:       info.Methods,
	}

	return tmpl.Execute(file, data)
}

func toEnvKey(methodName string) string {
	// Преобразуем имя метода в ключ переменной окружения
	// Например: Host -> HOST, SSLMode -> SSL_MODE, UserName -> USER_NAME
	var result strings.Builder

	for i, char := range methodName {
		isUpper := char >= 'A' && char <= 'Z'

		// Добавляем подчеркивание если:
		// 1. Текущий символ заглавный и предыдущий был строчным
		// 2. Или если следующий символ строчной (для аббревиатур)
		if isUpper && i > 0 {
			nextChar := byte(0)
			if i+1 < len(methodName) {
				nextChar = methodName[i+1]
			}

			// Если следующий символ строчной, то это начало слова
			if nextChar >= 'a' && nextChar <= 'z' {
				result.WriteByte('_')
			}
		}

		result.WriteByte(byte(char))
	}

	return strings.ToUpper(result.String())
}

func getEnvKey(packageName, methodName string) string {
	// Добавляем префикс пакета к ключу
	prefix := strings.ToUpper(packageName)
	return prefix + "_" + toEnvKey(methodName)
}

const unifiedTemplate = `// Code generated by ggconfig. DO NOT EDIT.

package {{.GenPackageName}}

import (
	"os"
	{{if hasIntType .Methods}}"strconv"{{end}}
	"gopkg.in/yaml.v3"
)

// ===== ENV Implementation =====

type {{.PackageName}}EnvConfig struct{}

{{range .Methods}}
func (c *{{$.PackageName}}EnvConfig) {{.Name}}(defaultValue {{.ParamType}}) {{.ReturnType}} {
	{{- $ret := .ReturnType -}}
	{{- range envAliasKeys .Name}}
	{{envCheck $ret .}}
	{{- end}}
	{{envReturn .ReturnType (envKey .Name)}}
}
{{end}}

func New{{.PackageName | title}}{{.InterfaceName | title}}() *{{.PackageName}}EnvConfig {
	return &{{.PackageName}}EnvConfig{}
}

// ===== YAML Implementation =====

type {{.PackageName}}YAMLConfig struct {
	data []byte
}

func New{{.PackageName | title}}{{.InterfaceName | title}}YAML(data []byte) *{{.PackageName}}YAMLConfig {
	return &{{.PackageName}}YAMLConfig{
		data: data,
	}
}

{{range .Methods}}
func (c *{{$.PackageName}}YAMLConfig) {{.Name}}(defaultValue {{.ParamType}}) {{.ReturnType}} {
	var config map[string]interface{}
	if err := yaml.Unmarshal(c.data, &config); err != nil {
		return defaultValue
	}

	// Алиасные секции
	{{- $assert := (yamlAssertType .ReturnType) -}}
	{{- range yamlSectionAliases }}
	if section, ok := config["{{.}}"].(map[string]interface{}); ok {
		{{- range yamlKeyAliases $.Name }}
		if value, ok := section["{{.}}"].({{$assert}}); ok {
			return value
		}
		{{- end}}
		if value, ok := section["{{$.Name | toLower}}"].({{$assert}}); ok {
			return value
		}
	}
	{{- end}}

	// Основная секция {{$.PackageName}}
	if section, ok := config["{{$.PackageName}}"].(map[string]interface{}); ok {
		{{- range yamlKeyAliases .Name }}
		if value, ok := section["{{.}}"].({{$assert}}); ok {
			return value
		}
		{{- end}}
		if value, ok := section["{{.Name | toLower}}"].({{$assert}}); ok {
			return value
		}
	}

	return defaultValue
}
{{end}}

// ===== Mock Implementation =====

type {{.PackageName}}MockConfig struct{}

{{range .Methods}}
func (c *{{$.PackageName}}MockConfig) {{.Name}}(defaultValue {{.ParamType}}) {{.ReturnType}} {
	return defaultValue
}
{{end}}

func New{{.PackageName | title}}{{.InterfaceName | title}}Mock() *{{.PackageName}}MockConfig {
	return &{{.PackageName}}MockConfig{}
}

// ===== Composite Implementation =====

type {{.PackageName}}AllConfig struct {
	sources []interface{
		{{- range .Methods}}
		{{.Name}}(defaultValue {{.ParamType}}) {{.ReturnType}}
		{{- end}}
	}
}

func New{{.PackageName | title}}{{.InterfaceName | title}}All(sources ...interface{
	{{- range .Methods}}
	{{.Name}}(defaultValue {{.ParamType}}) {{.ReturnType}}
	{{- end}}
}) *{{.PackageName}}AllConfig {
	return &{{.PackageName}}AllConfig{sources: sources}
}

{{range .Methods}}
func (c *{{$.PackageName}}AllConfig) {{.Name}}(defaultValue {{.ParamType}}) {{.ReturnType}} {
	sentinel := {{sentinelValue .ReturnType}}
	for _, s := range c.sources {
		v := s.{{.Name}}(sentinel)
		if v != sentinel {
			return v
		}
	}
	return defaultValue
}
{{end}}
`

const exampleTemplate = `# Example configuration for {{.PackageName}} package
# Copy this file to config.yaml or use with your application

{{.PackageName}}:
{{range .Methods}}  # {{.Name}} - {{.ParamType}} parameter{{if .Comment}} - {{.Comment}}{{end}}
  {{.Name}}: {{.ParamType | defaultValue}}
{{end}}
# Usage:
# 1. Copy this file to config.yaml
# 2. Or use with viper/cobra for config management
# 3. Or convert to environment variables
`
