package gometagen

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// // // // // // // // // //

// ensureOutputDir подготавливает родительскую директорию целевого файла.
// В force-режиме дерево создаётся целиком; иначе отсутствие директории считается ошибкой.
func ensureOutputDir(outputDir string, force bool) error {
	if force {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
		return nil
	}

	info, err := os.Stat(outputDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("output directory does not exist: %s (use -force to create it)", outputDir)
		}
		return fmt.Errorf("stat output directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("output parent is not a directory: %s", outputDir)
	}

	return nil
}

// //

func writeFileAtomically(outputFile string, dataArr []byte, force bool) error {
	if outputFile == "" {
		return fmt.Errorf("output path is empty")
	}

	// Не переписываем файл без изменений, чтобы не трогать timestamp и кеши.
	if existingData, err := os.ReadFile(outputFile); err == nil {
		if bytes.Equal(existingData, dataArr) {
			return nil
		}
	}

	outputDir := filepath.Dir(outputFile)
	if err := ensureOutputDir(outputDir, force); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(outputDir, ".gometagen-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
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
		return fmt.Errorf("write temp file: %w", err)
	}

	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err = os.Rename(tempPath, outputFile); err != nil {
		return fmt.Errorf("replace output file: %w", err)
	}

	removeTempFlag = false
	return nil
}
