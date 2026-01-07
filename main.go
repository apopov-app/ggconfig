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
	ReturnType string // value type (first return value)
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
	registryEnabled := flag.Bool("registry", false, "enable global registry: generates registry.gen.go in output package and init() self-registration in each generated file")
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
		fmt.Printf("  - %s(%s) (%s, bool)\n", method.Name, method.ParamType, method.ReturnType)
	}

	// Генерируем все реализации в одном файле
	if err := generateImplementation(info, aliasSettings, *outputPath, *registryEnabled); err != nil {
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
									paramType, returnType, err := getMethodSignature(funcType)
									if err != nil {
										// Fail fast: new ggconfig requires (T, bool) return signature
										log.Fatalf("bad method signature %s.%s: %v", interfaceName, methodName, err)
									}

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

func getReturnTypes(funcType *ast.FuncType) []string {
	if funcType.Results == nil || len(funcType.Results.List) == 0 {
		return nil
	}
	var out []string
	for _, r := range funcType.Results.List {
		ident, ok := r.Type.(*ast.Ident)
		if !ok {
			out = append(out, "")
			continue
		}
		out = append(out, ident.Name)
	}
	return out
}

// getEnvValue generates snippet to read env by expression (envKeyExpr) without quoting.
// envKeyExpr must be a valid Go expression producing a string.
func getEnvValue(envKeyExpr, defaultValue, returnType string) string {
	switch returnType {
	case "int":
		return fmt.Sprintf(`if value := os.Getenv(%s); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue, true
		}
	}
	return %s, false`, envKeyExpr, defaultValue)
	case "string":
		return fmt.Sprintf(`if value := os.Getenv(%s); value != "" {
		return value, true
	}
	return %s, false`, envKeyExpr, defaultValue)
	default:
		return fmt.Sprintf(`if value := os.Getenv(%s); value != "" {
		return value, true
	}
	return %s, false`, envKeyExpr, defaultValue)
	}
}

// Генерирует фрагмент кода проверки ENV по конкретному ключу без возврата default
func getEnvCheckSnippet(envKeyExpr, returnType string) string {
	switch returnType {
	case "int":
		return fmt.Sprintf(`if value := os.Getenv(%s); value != "" {
    if intValue, err := strconv.Atoi(value); err == nil {
        return intValue, true
    }
}`, envKeyExpr)
	case "string":
		return fmt.Sprintf(`if value := os.Getenv(%s); value != "" {
    return value, true
}`, envKeyExpr)
	default:
		return fmt.Sprintf(`if value := os.Getenv(%s); value != "" {
    return value, true
}`, envKeyExpr)
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

func getMethodSignature(funcType *ast.FuncType) (string, string, error) {
	// Получаем тип параметра (для простоты берем первый)
	var paramType string
	if funcType.Params != nil && len(funcType.Params.List) > 0 {
		if ident, ok := funcType.Params.List[0].Type.(*ast.Ident); ok {
			paramType = ident.Name
		}
	}

	rets := getReturnTypes(funcType)
	if len(rets) != 2 {
		return "", "", fmt.Errorf("expected 2 return values (T, bool), got %d", len(rets))
	}
	if rets[1] != "bool" {
		return "", "", fmt.Errorf("second return value must be bool, got %q", rets[1])
	}
	if rets[0] != "string" && rets[0] != "int" {
		return "", "", fmt.Errorf("unsupported value return type %q (supported: string, int)", rets[0])
	}
	return paramType, rets[0], nil
}

func generateImplementation(info *InterfaceInfo, aliases AliasSettings, outputPath string, registryEnabled bool) error {
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

	if registryEnabled {
		if err := ensureRegistryFile(fullOutputPath, packageName); err != nil {
			return err
		}
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
		"paramDefaultLiteral": func(paramType string) string {
			switch paramType {
			case "int":
				return "0"
			default:
				return "\"\""
			}
		},
	}).Parse(unifiedTemplate))

	data := struct {
		PackageName    string
		InterfaceName  string
		Methods        []Method
		GenPackageName string
		IsSamePackage  bool
		EnableRegistry bool
	}{
		PackageName:    info.PackageName,
		InterfaceName:  info.InterfaceName,
		Methods:        info.Methods,
		GenPackageName: packageName,
		IsSamePackage:  isSamePackage,
		EnableRegistry: registryEnabled,
	}

	return tmpl.Execute(file, data)
}

func ensureRegistryFile(outputDir string, genPackageName string) error {
	filePath := filepath.Join(outputDir, "registry.gen.go")
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create registry file %s: %w", filePath, err)
	}
	defer f.Close()

	// Registry API: package self-registration via init() in each generated file.
	// GlobalConfig loads YAML once (optional) and provides typed access via Get().
	content := fmt.Sprintf(`// Code generated by ggconfig. DO NOT EDIT.

package %s

import (
	"os"
	"sync"

	"github.com/apopov-app/ggconfig/runtime"
)

type Provider struct {
	Package string
	NewAllFromParsed func(y *runtime.YAML, mapKey func(string) string) any
}

var (
	registryMu sync.RWMutex
	registry = map[string]Provider{}
)

func Register(pkg string, p Provider) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[pkg] = p
}

func Providers() map[string]Provider {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make(map[string]Provider, len(registry))
	for k, v := range registry {
		out[k] = v
	}
	return out
}

// NewAllFromYAML builds a single package AllConfig from YAML bytes (YAML parsed once per call).
// Returns (nil, false, nil) if the package is not registered.
func NewAllFromYAML(pkg string, yamlData []byte) (any, bool, error) {
	y, err := runtime.ParseYAML(yamlData)
	if err != nil {
		return nil, false, err
	}
	registryMu.RLock()
	p, ok := registry[pkg]
	registryMu.RUnlock()
	if !ok || p.NewAllFromParsed == nil {
		return nil, false, nil
	}
	return p.NewAllFromParsed(y, func(k string) string { return k }), true, nil
}

// EnvConfig allows post-processing of env keys before os.Getenv, e.g. to inject prefixes.
type EnvConfig struct {
	mapKey func(string) string
}

func NewEnvConfig(mapKey func(key string) string) *EnvConfig {
	if mapKey == nil {
		mapKey = func(k string) string { return k }
	}
	return &EnvConfig{mapKey: mapKey}
}

type GlobalYamlConfig struct {
	path string
}

func NewGlobalYamlConfig(path string) *GlobalYamlConfig {
	return &GlobalYamlConfig{path: path}
}

type GlobalConfig struct {
	y *runtime.YAML
	mapKey func(string) string
}

// NewGlobalConfig creates app-wide config wrapper. Sources order does not matter.
// Supported sources:
// - *GlobalYamlConfig
// - *EnvConfig
func NewGlobalConfig(sources ...any) (*GlobalConfig, error) {
	g := &GlobalConfig{
		y:      &runtime.YAML{},
		mapKey: func(k string) string { return k },
	}
	var yamlPath string
	for _, s := range sources {
		switch t := s.(type) {
		case *EnvConfig:
			if t != nil && t.mapKey != nil {
				g.mapKey = t.mapKey
			}
		case *GlobalYamlConfig:
			if t != nil && t.path != "" {
				yamlPath = t.path
			}
		}
	}
	if yamlPath != "" {
		b, err := os.ReadFile(yamlPath)
		if err != nil {
			return nil, err
		}
		y, err := runtime.ParseYAML(b)
		if err != nil {
			return nil, err
		}
		g.y = y
	}
	return g, nil
}

`, genPackageName)

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("write registry file: %w", err)
	}
	return nil
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
	"github.com/apopov-app/ggconfig/runtime"
)

// ===== ENV Implementation =====

type {{.PackageName}}EnvConfig struct{
	mapKey func(string) string
}

{{range .Methods}}
func (c *{{$.PackageName}}EnvConfig) {{.Name}}(defaultValue {{.ParamType}}) ({{.ReturnType}}, bool) {
	{{- $ret := .ReturnType -}}
	{{- range envAliasKeys .Name}}
	{{envCheck $ret (printf "c.mapKey(%q)" .)}}
	{{- end}}
	{{envReturn .ReturnType (printf "c.mapKey(%q)" (envKey .Name))}}
}
{{end}}

func New{{.PackageName | title}}{{.InterfaceName | title}}EnvConfig() *{{.PackageName}}EnvConfig {
	return New{{.PackageName | title}}{{.InterfaceName | title}}EnvConfigWithMap(nil)
}

func New{{.PackageName | title}}{{.InterfaceName | title}}EnvConfigWithMap(mapKey func(string) string) *{{.PackageName}}EnvConfig {
	if mapKey == nil {
		mapKey = func(k string) string { return k }
	}
	return &{{.PackageName}}EnvConfig{mapKey: mapKey}
}

// ===== YAML Implementation =====

type {{.PackageName}}YAMLConfig struct {
	y *runtime.YAML
	err error
}

func New{{.PackageName | title}}{{.InterfaceName | title}}YAMLConfig(path string) *{{.PackageName}}YAMLConfig {
	b, err := os.ReadFile(path)
	if err != nil {
		return &{{.PackageName}}YAMLConfig{y: &runtime.YAML{}, err: err}
	}
	y, err := runtime.ParseYAML(b)
	if err != nil {
		return &{{.PackageName}}YAMLConfig{y: &runtime.YAML{}, err: err}
	}
	return &{{.PackageName}}YAMLConfig{y: y}
}

func New{{.PackageName | title}}{{.InterfaceName | title}}YAMLConfigParsed(y *runtime.YAML) *{{.PackageName}}YAMLConfig {
	return &{{.PackageName}}YAMLConfig{
		y: y,
	}
}

func (c *{{.PackageName}}YAMLConfig) Err() error { return c.err }

{{range .Methods}}
func (c *{{$.PackageName}}YAMLConfig) {{.Name}}(defaultValue {{.ParamType}}) ({{.ReturnType}}, bool) {
	{{- $methodName := .Name -}}
	{{- $keyPrimary := (.Name | toLower) -}}
	{{- if eq .ReturnType "int" }}
	// Алиасные секции
	{{- range yamlSectionAliases }}
	{{- $section := . -}}
	if v, ok := c.y.GetInt("{{$section}}", {{- range yamlKeyAliases $methodName }}"{{.}}",{{- end}} "{{$keyPrimary}}"); ok {
		return v, true
	}
	{{- end}}
	// Основная секция {{$.PackageName}}
	if v, ok := c.y.GetInt("{{$.PackageName}}", {{- range yamlKeyAliases $methodName }}"{{.}}",{{- end}} "{{$keyPrimary}}"); ok {
		return v, true
	}
	return defaultValue, false
	{{- else }}
	// Алиасные секции
	{{- range yamlSectionAliases }}
	{{- $section := . -}}
	if v, ok := c.y.GetString("{{$section}}", {{- range yamlKeyAliases $methodName }}"{{.}}",{{- end}} "{{$keyPrimary}}"); ok {
		return v, true
	}
	{{- end}}
	// Основная секция {{$.PackageName}}
	if v, ok := c.y.GetString("{{$.PackageName}}", {{- range yamlKeyAliases $methodName }}"{{.}}",{{- end}} "{{$keyPrimary}}"); ok {
		return v, true
	}
	return defaultValue, false
	{{- end }}
}
{{end}}

// ===== Mock Implementation =====

type {{.PackageName}}MockConfig struct{}

{{range .Methods}}
func (c *{{$.PackageName}}MockConfig) {{.Name}}(defaultValue {{.ParamType}}) ({{.ReturnType}}, bool) {
	return defaultValue, false
}
{{end}}

func New{{.PackageName | title}}{{.InterfaceName | title}}Mock() *{{.PackageName}}MockConfig {
	return &{{.PackageName}}MockConfig{}
}

// ===== Composite Implementation =====

type {{.PackageName}}AllConfig struct {
	sources []interface{
		{{- range .Methods}}
		{{.Name}}(defaultValue {{.ParamType}}) ({{.ReturnType}}, bool)
		{{- end}}
	}
}

func New{{.PackageName | title}}{{.InterfaceName | title}}All(sources ...interface{
	{{- range .Methods}}
	{{.Name}}(defaultValue {{.ParamType}}) ({{.ReturnType}}, bool)
	{{- end}}
}) *{{.PackageName}}AllConfig {
	return &{{.PackageName}}AllConfig{sources: sources}
}

{{range .Methods}}
func (c *{{$.PackageName}}AllConfig) {{.Name}}(defaultValue {{.ParamType}}) ({{.ReturnType}}, bool) {
	for _, s := range c.sources {
		v, ok := s.{{.Name}}(defaultValue)
		if ok {
			return v, true
		}
	}
	return defaultValue, false
}
{{end}}

{{if .EnableRegistry}}
func init() {
	Register("{{.PackageName}}", Provider{
		Package: "{{.PackageName}}",
		NewAllFromParsed: func(y *runtime.YAML, mapKey func(string) string) any {
			envCfg := New{{.PackageName | title}}{{.InterfaceName | title}}EnvConfigWithMap(mapKey)
			yamlCfg := New{{.PackageName | title}}{{.InterfaceName | title}}YAMLConfigParsed(y)
			return New{{.PackageName | title}}{{.InterfaceName | title}}All(envCfg, yamlCfg)
		},
	})
}

// Get{{.PackageName | title}} returns the concrete AllConfig type for this package.
// It can be passed anywhere the original interface is expected (structural typing).
func (g *GlobalConfig) Get{{.PackageName | title}}() (*{{.PackageName}}AllConfig, bool) {
	registryMu.RLock()
	p, ok := registry["{{.PackageName}}"]
	registryMu.RUnlock()
	if !ok || p.NewAllFromParsed == nil {
		return nil, false
	}
	v := p.NewAllFromParsed(g.y, g.mapKey)
	cfg, ok := v.(*{{.PackageName}}AllConfig)
	return cfg, ok
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
