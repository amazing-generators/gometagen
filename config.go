package gometagen

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// // // // // // // // // //

const (
	cDefaultFormat      = "go"
	cDefaultPackageName = "target"
	cDefaultGoOutput    = "meta_gen.go"
	cDefaultJSONOutput  = "meta_gen.json"
	cDefaultYAMLOutput  = "meta_gen.yml"
	cDateLayout         = "2006-01-02"
	// Минор и патч укладываются в uint16 во всех форматах вывода.
	cMaxVersionPart = 65535
)

var cDefaultHashExcludePatternArr = []string{
	"tmp",
	"target",
}

// //

type ConfigObj struct {
	Context        context.Context
	Source         string
	OutputFile     string
	PackageName    string
	Format         string
	TemplateFile   string
	HashSource     string
	HashSourceArr  []string
	HashExcludeArr []string
	HashNoAsync    bool
	Stdout         bool
	Force          bool
	Now            time.Time
}

type MetaObj struct {
	Name         string `json:"name" yaml:"name"`
	DateUpdate   string `json:"date_update" yaml:"date_update"`
	Hash         string `json:"hash" yaml:"hash"`
	Version      string `json:"version" yaml:"version"`
	VersionMajor string `json:"version_major" yaml:"version_major"`
	VersionMinor uint16 `json:"version_minor" yaml:"version_minor"`
	VersionPatch uint16 `json:"version_patch" yaml:"version_patch"`
}

type ResultObj struct {
	OutputFile string
	Meta       *MetaObj
	Data       []byte
}

type normalizedConfigObj struct {
	Context        context.Context
	SourcePath     string
	OutputFile     string
	PackageName    string
	Format         string
	TemplateFile   string
	HashSourceArr  []string
	HashExcludeArr []string
	HashNoAsync    bool
	Stdout         bool
	Force          bool
	Now            time.Time
}

// //

func normalizeConfig(config ConfigObj) (*normalizedConfigObj, error) {
	formatValue, err := normalizeFormat(config.Format, config.TemplateFile)
	if err != nil {
		return nil, err
	}

	sourcePath, err := resolveInputPath(config.Source)
	if err != nil {
		return nil, err
	}

	outputFile, err := resolveOutputFile(config.OutputFile, formatValue, config.Stdout)
	if err != nil {
		return nil, err
	}

	packageName := strings.TrimSpace(config.PackageName)
	if packageName == "" && outputFile != "" {
		packageName = sanitizePackageName(filepath.Base(filepath.Dir(outputFile)))
	}
	if packageName == "" {
		packageName = cDefaultPackageName
	}
	packageName = sanitizePackageName(packageName)

	templateFile := strings.TrimSpace(config.TemplateFile)
	if templateFile != "" {
		templateFile, err = filepath.Abs(templateFile)
		if err != nil {
			return nil, fmt.Errorf("resolve template path: %w", err)
		}
	}

	hashSourceArr, err := normalizeHashSourceArr(config.HashSource, config.HashSourceArr)
	if err != nil {
		return nil, err
	}

	hashExcludeArr := normalizeHashExcludePatternArr(config.HashExcludeArr)

	now := config.Now
	if now.IsZero() {
		now = time.Now()
	}

	contextObj := config.Context
	if contextObj == nil {
		contextObj = context.Background()
	}

	return &normalizedConfigObj{
		Context:        contextObj,
		SourcePath:     sourcePath,
		OutputFile:     outputFile,
		PackageName:    packageName,
		Format:         formatValue,
		TemplateFile:   templateFile,
		HashSourceArr:  hashSourceArr,
		HashExcludeArr: hashExcludeArr,
		HashNoAsync:    config.HashNoAsync,
		Stdout:         config.Stdout,
		Force:          config.Force,
		Now:            now,
	}, nil
}

func normalizeHashSourceArr(singleSource string, sourceArr []string) ([]string, error) {
	rawSourceArr := make([]string, 0, len(sourceArr)+1)

	if strings.TrimSpace(singleSource) != "" {
		rawSourceArr = append(rawSourceArr, singleSource)
	}

	rawSourceArr = append(rawSourceArr, sourceArr...)

	resultArr := make([]string, 0, len(rawSourceArr))
	seenMap := make(map[string]struct{}, len(rawSourceArr))

	for _, rawSource := range rawSourceArr {
		rawSource = strings.TrimSpace(rawSource)
		if rawSource == "" {
			continue
		}

		absPath, err := filepath.Abs(rawSource)
		if err != nil {
			return nil, fmt.Errorf("resolve hash source path: %w", err)
		}

		if _, existsFlag := seenMap[absPath]; existsFlag {
			continue
		}

		seenMap[absPath] = struct{}{}
		resultArr = append(resultArr, absPath)
	}

	return resultArr, nil
}

func normalizeHashExcludePatternArr(rawPatternArr []string) []string {
	normalizedArr := make([]string, 0, len(cDefaultHashExcludePatternArr)+len(rawPatternArr))
	seenMap := make(map[string]struct{}, len(cDefaultHashExcludePatternArr)+len(rawPatternArr))

	appendPattern := func(patternValue string) {
		patternValue = strings.TrimSpace(patternValue)
		if patternValue == "" {
			return
		}

		if _, existsFlag := seenMap[patternValue]; existsFlag {
			return
		}

		seenMap[patternValue] = struct{}{}
		normalizedArr = append(normalizedArr, patternValue)
	}

	for _, patternValue := range cDefaultHashExcludePatternArr {
		appendPattern(patternValue)
	}

	for _, patternValue := range rawPatternArr {
		appendPattern(patternValue)
	}

	return normalizedArr
}

func normalizeFormat(formatValue string, templateFile string) (string, error) {
	formatValue = strings.TrimSpace(strings.ToLower(formatValue))
	hasTemplate := strings.TrimSpace(templateFile) != ""

	switch formatValue {
	case "":
		if hasTemplate {
			return "template", nil
		}

		return cDefaultFormat, nil
	case "go", "json", "yml", "yaml":
		if hasTemplate {
			return "", fmt.Errorf("format %s conflicts with -template flag", formatValue)
		}

		if formatValue == "yml" {
			return "yaml", nil
		}

		return formatValue, nil
	case "template":
		if !hasTemplate {
			return "", fmt.Errorf("template file is required for format template")
		}

		return "template", nil
	default:
		return "", fmt.Errorf("unsupported format: %s", formatValue)
	}
}

func resolveInputPath(rawPath string) (string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("read working directory: %w", err)
		}

		return cwd, nil
	}

	absPath, err := filepath.Abs(rawPath)
	if err != nil {
		return "", fmt.Errorf("resolve source path: %w", err)
	}

	return absPath, nil
}

func resolveOutputFile(outputValue string, formatValue string, stdout bool) (string, error) {
	if formatValue == "template" {
		return resolveTemplateOutputFile(outputValue, stdout)
	}

	outputValue = strings.TrimSpace(outputValue)
	if outputValue == "" && stdout {
		return "", nil
	}

	defaultFileName := defaultOutputFileName(formatValue)
	if outputValue == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("read working directory: %w", err)
		}

		return filepath.Join(cwd, defaultFileName), nil
	}

	isDirHint := strings.HasSuffix(outputValue, string(os.PathSeparator)) || filepath.Ext(outputValue) == ""

	outputPath, err := filepath.Abs(outputValue)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}

	info, err := os.Stat(outputPath)
	if err == nil {
		if info.IsDir() {
			return filepath.Join(outputPath, defaultFileName), nil
		}

		return outputPath, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat output path: %w", err)
	}

	if isDirHint {
		return filepath.Join(outputPath, defaultFileName), nil
	}

	return outputPath, nil
}

func resolveTemplateOutputFile(outputValue string, stdout bool) (string, error) {
	outputValue = strings.TrimSpace(outputValue)
	if outputValue == "" {
		if stdout {
			return "", nil
		}

		return "", fmt.Errorf("output file is required for custom template")
	}

	if strings.HasSuffix(outputValue, string(os.PathSeparator)) {
		return "", fmt.Errorf("custom template output must be an explicit file path: %s", outputValue)
	}

	outputPath, err := filepath.Abs(outputValue)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}

	info, err := os.Stat(outputPath)
	if err == nil {
		if info.IsDir() {
			return "", fmt.Errorf("custom template output must be an explicit file path: %s", outputPath)
		}

		return outputPath, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat output path: %w", err)
	}

	return outputPath, nil
}

func defaultOutputFileName(formatValue string) string {
	switch formatValue {
	case "json":
		return cDefaultJSONOutput
	case "yaml":
		return cDefaultYAMLOutput
	default:
		return cDefaultGoOutput
	}
}

func sanitizePackageName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return cDefaultPackageName
	}

	var out strings.Builder

	for index, symbol := range raw {
		switch {
		case symbol >= 'a' && symbol <= 'z':
			out.WriteRune(symbol)
		case symbol >= 'A' && symbol <= 'Z':
			out.WriteRune(symbol + ('a' - 'A'))
		case symbol >= '0' && symbol <= '9':
			if index == 0 {
				out.WriteByte('p')
			}
			out.WriteRune(symbol)
		default:
			out.WriteByte('_')
		}
	}

	packageName := out.String()
	if packageName == "" {
		return cDefaultPackageName
	}

	return packageName
}
