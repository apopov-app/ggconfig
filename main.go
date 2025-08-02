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

func main() {
	interfaceName := flag.String("interface", "", "interface name")
	outputPath := flag.String("output", "", "output directory path")
	examplePath := flag.String("example", "", "generate example config file")
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

	fmt.Printf("Found %d methods in interface\n", len(info.Methods))
	for _, method := range info.Methods {
		fmt.Printf("  - %s(%s) %s\n", method.Name, method.ParamType, method.ReturnType)
	}

	// Генерируем все реализации в одном файле
	if err := generateImplementation(info, *outputPath); err != nil {
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

func generateImplementation(info *InterfaceInfo, outputPath string) error {
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
		fullOutputPath = filepath.Join("..", "..", outputPath)
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
		"envValue": func(methodName, returnType string) string {
			// methodName здесь на самом деле returnType, а returnType - это methodName
			// Это из-за того, как работает шаблон
			actualMethodName := returnType
			actualReturnType := methodName
			envKey := getEnvKey(info.PackageName, actualMethodName)
			return getEnvValue(envKey, "defaultValue", actualReturnType)
		},
		"hasIntType": func(methods []Method) bool {
			for _, method := range methods {
				if method.ReturnType == "int" {
					return true
				}
			}
			return false
		},
		"toLower": strings.ToLower,
	}).Parse(unifiedTemplate))

	data := struct {
		PackageName    string
		InterfaceName  string
		Methods        []Method
		ImportPath     string
		GenPackageName string
		IsSamePackage  bool
	}{
		PackageName:    info.PackageName,
		InterfaceName:  info.InterfaceName,
		Methods:        info.Methods,
		ImportPath:     fmt.Sprintf("github.com/apopov-app/ggconfig/example2/internal/%s", info.PackageName),
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
		ImportPath    string
	}{
		PackageName:   info.PackageName,
		InterfaceName: info.InterfaceName,
		Methods:       info.Methods,
		ImportPath:    fmt.Sprintf("github.com/apopov-app/ggconfig/internal/%s", info.PackageName),
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
	{{if not .IsSamePackage}}"{{.ImportPath}}"{{end}}
)

// ===== ENV Implementation =====

type {{.PackageName}}EnvConfig struct{}

{{range .Methods}}
func (c *{{$.PackageName}}EnvConfig) {{.Name}}(defaultValue {{.ParamType}}) {{.ReturnType}} {
	{{.Name | envValue .ReturnType}}
}
{{end}}

func NewConfig{{.PackageName | title}}() {{.PackageName}}.{{.InterfaceName}} {
	return &{{.PackageName}}EnvConfig{}
}

// ===== YAML Implementation =====

type {{.PackageName}}YAMLConfig struct {
	data []byte
}

func NewYAMLConfig(data []byte) {{.PackageName}}.{{.InterfaceName}} {
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
	
	// Читаем секцию {{$.PackageName}}
	if section, ok := config["{{$.PackageName}}"].(map[string]interface{}); ok {
		if value, ok := section["{{.Name | toLower}}"].({{.ReturnType}}); ok {
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

func NewMock{{.PackageName | title}}() {{.PackageName}}.{{.InterfaceName}} {
	return &{{.PackageName}}MockConfig{}
}
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
