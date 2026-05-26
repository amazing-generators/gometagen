package gometagen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	projectsys "github.com/amazing-generators/gometagen/sys"
)

// // // // // // // // // //

var templateFuncMap = template.FuncMap{
	"quote":      strconvQuote,
	"jsonQuote":  jsonQuote,
	"yamlInline": yamlInline,
	"yamlValue":  yamlValue,
}

var (
	goTemplateObj   = template.Must(template.New("go").Funcs(templateFuncMap).Parse(goTemplateText))
	jsonTemplateObj = template.Must(template.New("json").Funcs(templateFuncMap).Parse(jsonTemplateText))
	yamlTemplateObj = template.Must(template.New("yaml").Funcs(templateFuncMap).Parse(yamlTemplateText))
)

// //

type templateDataObj struct {
	PackageName  string
	ManifestPath string
	GeneratedAt  string
	HashMode     string
	HashPoint    string
	HashSource   string
	HashSources  []string
	Manifest     *projectsys.ValuesObj
	Meta         *MetaObj
}

// //

func renderNormalized(
	config normalizedConfigObj,
	manifestObj *projectsys.ValuesObj,
	metaObj *MetaObj,
	manifestPath string,
	hashModeValue string,
	hashPointValue string,
) ([]byte, error) {
	dataObj := templateDataObj{
		PackageName:  config.PackageName,
		ManifestPath: manifestPath,
		GeneratedAt:  config.Now.Format(time.RFC3339),
		HashMode:     hashModeValue,
		HashPoint:    hashPointValue,
		HashSource:   strings.Join(config.HashSourceArr, ", "),
		HashSources:  append([]string(nil), config.HashSourceArr...),
		Manifest:     manifestObj,
		Meta:         metaObj,
	}

	switch config.Format {
	case "go":
		return renderGo(dataObj)
	case "json":
		return renderJSON(dataObj)
	case "yaml":
		return renderYAML(dataObj)
	case "template":
		return renderCustomTemplate(config.TemplateFile, dataObj)
	default:
		return nil, fmt.Errorf("unsupported format: %s", config.Format)
	}
}

func renderCustomTemplate(templateFile string, dataObj templateDataObj) ([]byte, error) {
	dataArr, err := os.ReadFile(templateFile)
	if err != nil {
		return nil, fmt.Errorf("read template file: %w", err)
	}

	templateObj, err := template.New(templateFile).Funcs(templateFuncMap).Parse(string(dataArr))
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	return executePreparedTemplate(templateObj, dataObj)
}

func executePreparedTemplate(templateObj *template.Template, dataObj templateDataObj) ([]byte, error) {
	var buffer bytes.Buffer
	if err := templateObj.Execute(&buffer, dataObj); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	return buffer.Bytes(), nil
}

func strconvQuote(value string) string {
	return fmt.Sprintf("%q", value)
}

func jsonQuote(value string) string {
	dataArr, _ := json.Marshal(value)
	return string(dataArr)
}

func yamlInline(value string) string {
	if value == "" {
		return `""`
	}

	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func yamlValue(value string, indent int) string {
	if value == "" {
		return `""`
	}

	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	if !strings.Contains(value, "\n") {
		return yamlInline(value)
	}

	prefix := strings.Repeat(" ", indent)
	linesArr := strings.Split(value, "\n")

	var builder strings.Builder
	builder.WriteString("|-\n")

	for index, line := range linesArr {
		if index > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(prefix)
		builder.WriteString(line)
	}

	return builder.String()
}

func ensureTrailingNewline(dataArr []byte) []byte {
	if len(dataArr) == 0 || dataArr[len(dataArr)-1] == '\n' {
		return dataArr
	}

	return append(dataArr, '\n')
}
