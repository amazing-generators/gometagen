package sys

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// // // // // // // // // //

const (
	cManifestJSON = "values.json"
	cManifestYML  = "values.yml"
	cManifestYAML = "values.yaml"
)

type ValuesObj struct {
	Name string `json:"name" yaml:"name"`
	Ver  string `json:"ver" yaml:"ver"`
}

// //

func cloneValues(valuesObj *ValuesObj) *ValuesObj {
	if valuesObj == nil {
		return nil
	}

	copyObj := *valuesObj
	return &copyObj
}

// //

func readManifestFile(filePath string) (*ValuesObj, error) {
	dataArr, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}

	valuesObj := ValuesObj{}

	switch filepath.Ext(filePath) {
	case ".json":
		if err = json.Unmarshal(dataArr, &valuesObj); err != nil {
			return nil, fmt.Errorf("decode json manifest: %w", err)
		}
	case ".yml", ".yaml":
		if err = yaml.Unmarshal(dataArr, &valuesObj); err != nil {
			return nil, fmt.Errorf("decode yaml manifest: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported manifest format: %s", filePath)
	}

	if err = validateValues(valuesObj, filePath); err != nil {
		return nil, err
	}

	return &valuesObj, nil
}

func writeManifestFile(filePath string, valuesObj *ValuesObj, force bool) error {
	if valuesObj == nil {
		return fmt.Errorf("manifest value is nil")
	}

	if err := validateValues(*valuesObj, ""); err != nil {
		return err
	}

	outputDir := filepath.Dir(filePath)
	if err := ensureDir(outputDir, force); err != nil {
		return err
	}

	dataArr, err := marshalManifest(valuesObj, filePath)
	if err != nil {
		return err
	}

	if existingArr, readErr := os.ReadFile(filePath); readErr == nil && bytes.Equal(existingArr, dataArr) {
		return nil
	}

	tempFile, err := os.CreateTemp(outputDir, ".gometagen-manifest-*")
	if err != nil {
		return fmt.Errorf("create temp manifest: %w", err)
	}

	tempPath := tempFile.Name()
	removeTempFlag := true

	defer func() {
		_ = tempFile.Close()
		if removeTempFlag {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err = tempFile.Write(dataArr); err != nil {
		return fmt.Errorf("write temp manifest: %w", err)
	}

	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("close temp manifest: %w", err)
	}

	if err = os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("replace manifest file: %w", err)
	}

	removeTempFlag = false
	return nil
}

func resolveManifestPath(sourcePath string) (string, string, error) {
	absPath, err := resolveInputPath(sourcePath)
	if err != nil {
		return "", "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", "", fmt.Errorf("stat manifest source: %w", err)
	}

	if info.IsDir() {
		manifestPath, err := resolveManifestFromDir(absPath)
		if err != nil {
			return "", "", err
		}

		return absPath, manifestPath, nil
	}

	if !isManifestExtension(filepath.Ext(absPath)) {
		return "", "", fmt.Errorf("manifest file must use .json, .yml, or .yaml: %s", absPath)
	}

	return filepath.Dir(absPath), absPath, nil
}

func CreateManifest(targetPath string, format string, valuesObj *ValuesObj, force bool) (string, error) {
	manifestPath, err := resolveManifestWritePath(targetPath, format)
	if err != nil {
		return "", err
	}

	if err = writeManifestFile(manifestPath, valuesObj, force); err != nil {
		return "", err
	}

	return manifestPath, nil
}

func resolveManifestWritePath(targetPath string, format string) (string, error) {
	manifestFormat, err := normalizeManifestFormat(format)
	if err != nil {
		return "", err
	}

	absPath, err := resolveInputPath(targetPath)
	if err != nil {
		return "", err
	}

	info, statErr := os.Stat(absPath)
	switch {
	case statErr == nil && info.IsDir():
		return filepath.Join(resolveManifestDir(absPath), defaultManifestFileName(manifestFormat)), nil
	case statErr == nil:
		if !isManifestExtension(filepath.Ext(absPath)) {
			return "", fmt.Errorf("manifest file must use .json, .yml, or .yaml: %s", absPath)
		}

		return absPath, nil
	case !errors.Is(statErr, os.ErrNotExist):
		return "", fmt.Errorf("stat manifest output path: %w", statErr)
	}

	if isManifestExtension(filepath.Ext(absPath)) {
		return absPath, nil
	}

	if filepath.Ext(absPath) != "" {
		return "", fmt.Errorf("manifest file must use .json, .yml, or .yaml: %s", absPath)
	}

	return filepath.Join(resolveManifestDir(absPath), defaultManifestFileName(manifestFormat)), nil
}

func validateValues(valuesObj ValuesObj, filePath string) error {
	valuesObj.Name = strings.TrimSpace(valuesObj.Name)
	valuesObj.Ver = strings.TrimSpace(valuesObj.Ver)

	if valuesObj.Name == "" {
		if filePath == "" {
			return fmt.Errorf("manifest name is empty")
		}

		return fmt.Errorf("manifest name is empty: %s", filePath)
	}

	if valuesObj.Ver == "" {
		if filePath == "" {
			return fmt.Errorf("manifest ver is empty")
		}

		return fmt.Errorf("manifest ver is empty: %s", filePath)
	}

	if err := ValidateVersion(valuesObj.Ver); err != nil {
		if filePath == "" {
			return err
		}

		return fmt.Errorf("%w: %s", err, filePath)
	}

	return nil
}

func marshalManifest(valuesObj *ValuesObj, filePath string) ([]byte, error) {
	normalizedObj := ValuesObj{
		Name: strings.TrimSpace(valuesObj.Name),
		Ver:  strings.TrimSpace(valuesObj.Ver),
	}

	var (
		dataArr []byte
		err     error
	)

	switch filepath.Ext(filePath) {
	case ".json":
		dataArr, err = json.MarshalIndent(normalizedObj, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode json manifest: %w", err)
		}
	case ".yml", ".yaml":
		dataArr, err = yaml.Marshal(normalizedObj)
		if err != nil {
			return nil, fmt.Errorf("encode yaml manifest: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported manifest format: %s", filePath)
	}

	return append(dataArr, '\n'), nil
}

func resolveManifestFromDir(dirPath string) (string, error) {
	searchDirArr := []string{
		dirPath,
		resolveManifestDir(dirPath),
		resolveLegacyValuesDir(dirPath),
	}

	foundArr := make([]string, 0, len(searchDirArr)*3)
	seenMap := make(map[string]struct{}, len(searchDirArr)*3)

	for _, baseDir := range searchDirArr {
		for _, fileName := range []string{cManifestJSON, cManifestYML, cManifestYAML} {
			filePath := filepath.Join(baseDir, fileName)
			if _, existsFlag := seenMap[filePath]; existsFlag {
				continue
			}

			seenMap[filePath] = struct{}{}

			info, err := os.Stat(filePath)
			if err == nil {
				if info.IsDir() {
					return "", fmt.Errorf("manifest path is a directory: %s", filePath)
				}

				foundArr = append(foundArr, filePath)
				continue
			}

			if !errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("stat manifest path %s: %w", filePath, err)
			}
		}
	}

	switch len(foundArr) {
	case 1:
		return foundArr[0], nil
	case 0:
		return "", fmt.Errorf("manifest file not found in %s", dirPath)
	default:
		return "", fmt.Errorf("multiple manifest files found near %s", dirPath)
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
		return "", fmt.Errorf("resolve path: %w", err)
	}

	return absPath, nil
}

func ensureDir(dirPath string, force bool) error {
	if force {
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}

		return nil
	}

	info, err := os.Stat(dirPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("directory does not exist: %s (use -force to create it)", dirPath)
		}

		return fmt.Errorf("stat directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", dirPath)
	}

	return nil
}

func normalizeManifestFormat(raw string) (string, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch raw {
	case "", "json":
		return "json", nil
	case "yml", "yaml":
		return "yaml", nil
	default:
		return "", fmt.Errorf("unsupported manifest format: %s", raw)
	}
}

func defaultManifestFileName(format string) string {
	if format == "yaml" {
		return cManifestYML
	}

	return cManifestJSON
}

func resolveManifestDir(dirPath string) string {
	cleanPath := filepath.Clean(dirPath)
	if filepath.Base(cleanPath) == "_run" {
		return cleanPath
	}
	if filepath.Base(cleanPath) == "values" && filepath.Base(filepath.Dir(cleanPath)) == "_run" {
		return filepath.Dir(cleanPath)
	}

	return filepath.Join(cleanPath, "_run")
}

func resolveLegacyValuesDir(dirPath string) string {
	cleanPath := filepath.Clean(dirPath)
	if filepath.Base(cleanPath) == "values" && filepath.Base(filepath.Dir(cleanPath)) == "_run" {
		return cleanPath
	}

	return filepath.Join(resolveManifestDir(cleanPath), "values")
}

func isManifestExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case ".json", ".yml", ".yaml":
		return true
	default:
		return false
	}
}
