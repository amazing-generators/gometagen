package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/amazing-generators/gometagen"
	projectgit "github.com/amazing-generators/gometagen/git"
	projectsys "github.com/amazing-generators/gometagen/sys"
)

// // // // // // // // // //

const cDefaultTemplateFile = "gometagen.tmpl"

//go:embed template_draft.tmpl
var cTemplateDraft string

// //

type stringSliceFlagObj struct {
	ValuesArr *[]string
}

// //

func (obj *stringSliceFlagObj) String() string {
	if obj == nil || obj.ValuesArr == nil {
		return ""
	}

	return strings.Join(*obj.ValuesArr, ",")
}

func (obj *stringSliceFlagObj) Set(value string) error {
	if obj == nil || obj.ValuesArr == nil {
		return fmt.Errorf("string slice flag is not initialized")
	}

	*obj.ValuesArr = append(*obj.ValuesArr, value)
	return nil
}

// //

func runCLI(argsArr []string) error {
	if len(argsArr) == 0 {
		return runGenerateCommand(nil)
	}

	switch argsArr[0] {
	case "generate":
		return runGenerateCommand(argsArr[1:])
	case "validate":
		return runValidateCommand(argsArr[1:])
	case "manifest-init":
		return runManifestInitCommand(argsArr[1:])
	case "template-init":
		return runTemplateInitCommand(argsArr[1:])
	case "manifest":
		return runManifestCommand(argsArr[1:])
	case "version":
		return runVersionCommand(argsArr[1:])
	case "git":
		return runGitCommand(argsArr[1:])
	case "help", "-h", "--help":
		printRootUsage()
		return nil
	default:
		if strings.HasPrefix(argsArr[0], "-") {
			return runGenerateCommand(argsArr)
		}

		return fmt.Errorf("unknown command: %s", argsArr[0])
	}
}

func runGenerateCommand(argsArr []string) error {
	flagSet := newFlagSet("generate")
	config := gometagen.ConfigObj{}
	contextObj, stopFunc := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopFunc()

	config.Context = contextObj

	flagSet.StringVar(&config.Source, "source", "", "manifest file or project directory; default is current working directory")
	flagSet.StringVar(&config.OutputFile, "out", "", "output file or directory path")
	flagSet.StringVar(&config.PackageName, "pkg", "", "package name for generated Go output")
	flagSet.StringVar(&config.Format, "format", "", "output format: go, json, yaml, yml, or template")
	flagSet.StringVar(&config.TemplateFile, "template", "", "custom template file path")
	flagSet.Var(&stringSliceFlagObj{ValuesArr: &config.HashSourceArr}, "hash-source", "repeatable file or directory path for content-based hash generation")
	flagSet.Var(&stringSliceFlagObj{ValuesArr: &config.HashExcludeArr}, "hash-exclude", "repeatable file or directory name pattern to skip during content hashing")
	flagSet.BoolVar(&config.HashNoAsync, "hash-no-async", false, "disable concurrent content hashing and use a single sequential worker")
	flagSet.BoolVar(&config.Stdout, "stdout", false, "write output to stdout instead of a file")
	flagSet.BoolVar(&config.Force, "force", false, "create missing output directories")

	flagSet.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, "Usage: gometagen generate [flags]")
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(argsArr); err != nil {
		return err
	}

	resultObj, err := gometagen.Run(config)
	if err != nil {
		return err
	}

	if config.Stdout {
		_, err = os.Stdout.Write(resultObj.Data)
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "Generated:", resultObj.OutputFile)
	return nil
}

func runValidateCommand(argsArr []string) error {
	flagSet := newFlagSet("validate")
	sourcePath := ""

	flagSet.StringVar(&sourcePath, "source", "", "manifest file or project directory; default is current working directory")
	flagSet.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, "Usage: gometagen validate [flags]")
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(argsArr); err != nil {
		return err
	}

	sysObj, err := projectsys.New(sourcePath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stdout, "false")
		return nil
	}

	if err = sysObj.Validate(); err != nil {
		_, _ = fmt.Fprintln(os.Stdout, "false")
		return nil
	}

	_, _ = fmt.Fprintln(os.Stdout, "true")
	return nil
}

func runManifestInitCommand(argsArr []string) error {
	flagSet := newFlagSet("manifest-init")
	outputPath := ""
	formatValue := "json"
	nameValue := ""
	golandFlag := false
	forceFlag := false

	flagSet.StringVar(&outputPath, "out", "", "manifest file or target directory")
	flagSet.StringVar(&formatValue, "format", "json", "manifest format: json, yaml, or yml")
	flagSet.StringVar(&nameValue, "name", "", "manifest project name")
	flagSet.BoolVar(&golandFlag, "goland", false, "use Go-style initial version v0.0.1")
	flagSet.BoolVar(&forceFlag, "force", false, "create missing output directories")
	flagSet.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, "Usage: gometagen manifest-init [flags]")
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(argsArr); err != nil {
		return err
	}

	manifestPath, err := projectsys.Create(outputPath, formatValue, nameValue, golandFlag, forceFlag)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "Created:", manifestPath)
	return nil
}

func runTemplateInitCommand(argsArr []string) error {
	flagSet := newFlagSet("template-init")
	outputPath := ""
	forceFlag := false

	flagSet.StringVar(&outputPath, "out", "", "template file or target directory")
	flagSet.BoolVar(&forceFlag, "force", false, "create missing output directories")
	flagSet.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, "Usage: gometagen template-init [flags]")
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(argsArr); err != nil {
		return err
	}

	templatePath, err := resolveGenericOutputPath(outputPath, cDefaultTemplateFile)
	if err != nil {
		return err
	}

	if err = writeTextFile(templatePath, []byte(cTemplateDraft), forceFlag); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, "Created:", templatePath)
	return nil
}

func runVersionCommand(argsArr []string) error {
	if len(argsArr) == 0 {
		return fmt.Errorf("version subcommand is required")
	}

	subCommand := argsArr[0]
	subArgs := argsArr[1:]

	switch subCommand {
	case "print":
		return runVersionMutationCommand(subCommand, subArgs, false, func(sysObj *projectsys.Obj, _ string) (string, error) {
			return sysObj.VersionString()
		})
	case "major":
		return runVersionMutationCommand(subCommand, subArgs, false, func(sysObj *projectsys.Obj, _ string) (string, error) {
			return sysObj.IncrementMajor()
		})
	case "minor":
		return runVersionMutationCommand(subCommand, subArgs, false, func(sysObj *projectsys.Obj, _ string) (string, error) {
			return sysObj.IncrementMinor()
		})
	case "patch":
		return runVersionMutationCommand(subCommand, subArgs, false, func(sysObj *projectsys.Obj, _ string) (string, error) {
			return sysObj.IncrementPatch()
		})
	case "premajor":
		return runVersionMutationCommand(subCommand, subArgs, true, func(sysObj *projectsys.Obj, preID string) (string, error) {
			return sysObj.PreMajor(preID)
		})
	case "preminor":
		return runVersionMutationCommand(subCommand, subArgs, true, func(sysObj *projectsys.Obj, preID string) (string, error) {
			return sysObj.PreMinor(preID)
		})
	case "prepatch":
		return runVersionMutationCommand(subCommand, subArgs, true, func(sysObj *projectsys.Obj, preID string) (string, error) {
			return sysObj.PrePatch(preID)
		})
	case "prerelease":
		return runVersionMutationCommand(subCommand, subArgs, true, func(sysObj *projectsys.Obj, preID string) (string, error) {
			return sysObj.Prerelease(preID)
		})
	default:
		return fmt.Errorf("unknown version subcommand: %s", subCommand)
	}
}

// runVersionMutationCommand разбирает общие флаги версионных подкоманд и печатает итог.
// Флаг -preid подключается только для prerelease-форм.
func runVersionMutationCommand(
	name string,
	argsArr []string,
	usePreID bool,
	runFunc func(sysObj *projectsys.Obj, preID string) (string, error),
) error {
	flagSet := newFlagSet("version " + name)
	sourcePath := ""
	preIDValue := ""

	flagSet.StringVar(&sourcePath, "source", "", "manifest file or project directory; default is current working directory")
	if usePreID {
		flagSet.StringVar(&preIDValue, "preid", "", "prerelease identifier prefix")
	}
	flagSet.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, "Usage: gometagen version "+name+" [flags]")
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(argsArr); err != nil {
		return err
	}

	sysObj, err := projectsys.New(sourcePath)
	if err != nil {
		return err
	}

	versionValue, err := runFunc(sysObj, preIDValue)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, versionValue)
	return nil
}

func runManifestCommand(argsArr []string) error {
	if len(argsArr) == 0 {
		return fmt.Errorf("manifest subcommand is required")
	}

	switch argsArr[0] {
	case "get":
		return runManifestGetCommand(argsArr[1:])
	default:
		return fmt.Errorf("unknown manifest subcommand: %s", argsArr[0])
	}
}

func runManifestGetCommand(argsArr []string) error {
	flagSet := newFlagSet("manifest get")
	sourcePath := ""
	fieldValue := ""

	flagSet.StringVar(&sourcePath, "source", "", "manifest file or project directory; default is current working directory")
	flagSet.StringVar(&fieldValue, "field", "", "manifest field: name or ver")
	flagSet.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, "Usage: gometagen manifest get [flags]")
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(argsArr); err != nil {
		return err
	}

	sysObj, err := projectsys.New(sourcePath)
	if err != nil {
		return err
	}

	manifestObj, err := sysObj.Manifest()
	if err != nil {
		return err
	}

	switch strings.ToLower(strings.TrimSpace(fieldValue)) {
	case "name":
		_, _ = fmt.Fprintln(os.Stdout, manifestObj.Name)
	case "ver":
		_, _ = fmt.Fprintln(os.Stdout, manifestObj.Ver)
	default:
		return fmt.Errorf("unsupported manifest field: %s", fieldValue)
	}

	return nil
}

func runGitCommand(argsArr []string) error {
	if len(argsArr) == 0 {
		return fmt.Errorf("git subcommand is required")
	}

	switch argsArr[0] {
	case "branch":
		return runGitBranchCommand(argsArr[1:])
	case "add-commit-hook":
		return runGitHookCommand(argsArr[1:], func(gitObj *projectgit.Obj) error {
			return gitObj.AddCommitHook()
		})
	case "add-push-hook":
		return runGitHookCommand(argsArr[1:], func(gitObj *projectgit.Obj) error {
			return gitObj.AddPushHook()
		})
	case "del-commit-hook":
		return runGitHookCommand(argsArr[1:], func(gitObj *projectgit.Obj) error {
			return gitObj.DelCommitHook()
		})
	case "del-push-hook":
		return runGitHookCommand(argsArr[1:], func(gitObj *projectgit.Obj) error {
			return gitObj.DelPushHook()
		})
	default:
		return fmt.Errorf("unknown git subcommand: %s", argsArr[0])
	}
}

func runGitBranchCommand(argsArr []string) error {
	flagSet := newFlagSet("git branch")
	sourcePath := ""

	flagSet.StringVar(&sourcePath, "source", "", "repository directory; default is current working directory")
	flagSet.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, "Usage: gometagen git branch [flags]")
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(argsArr); err != nil {
		return err
	}

	gitObj, err := loadGitObj(sourcePath)
	if err != nil {
		return err
	}

	branchValue, err := gitObj.Branch()
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, branchValue)
	return nil
}

func runGitHookCommand(argsArr []string, runFunc func(gitObj *projectgit.Obj) error) error {
	flagSet := newFlagSet("git hook")
	sourcePath := ""

	flagSet.StringVar(&sourcePath, "source", "", "repository directory; default is current working directory")
	flagSet.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, "Usage: gometagen git <hook-command> [flags]")
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(argsArr); err != nil {
		return err
	}

	gitObj, err := loadGitObj(sourcePath)
	if err != nil {
		return err
	}

	return runFunc(gitObj)
}

func loadGitObj(sourcePath string) (*projectgit.Obj, error) {
	sourcePath, err := resolveSourceDir(sourcePath)
	if err != nil {
		return nil, err
	}

	return projectgit.New(sourcePath)
}

func resolveSourceDir(sourcePath string) (string, error) {
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("read working directory: %w", err)
		}

		return cwd, nil
	}

	absPath, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", fmt.Errorf("resolve source path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("stat source path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("source path is not a directory: %s", absPath)
	}

	return absPath, nil
}

func newFlagSet(name string) *flag.FlagSet {
	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)
	flagSet.SetOutput(os.Stderr)
	return flagSet
}

func printRootUsage() {
	_, _ = fmt.Fprintln(os.Stderr, "Usage:")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen generate [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen validate [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen manifest-init [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen manifest get [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen template-init [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen version print [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen version major [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen version minor [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen version patch [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen version premajor [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen version preminor [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen version prepatch [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen version prerelease [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen git branch [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen git add-commit-hook [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen git add-push-hook [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen git del-commit-hook [flags]")
	_, _ = fmt.Fprintln(os.Stderr, "  gometagen git del-push-hook [flags]")
}

func resolveGenericOutputPath(outputPath string, defaultFileName string) (string, error) {
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("read working directory: %w", err)
		}

		return filepath.Join(cwd, defaultFileName), nil
	}

	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err == nil {
		if info.IsDir() {
			return filepath.Join(absPath, defaultFileName), nil
		}

		return absPath, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat output path: %w", err)
	}

	if strings.HasSuffix(outputPath, string(os.PathSeparator)) || filepath.Ext(absPath) == "" {
		return filepath.Join(absPath, defaultFileName), nil
	}

	return absPath, nil
}

func writeTextFile(filePath string, dataArr []byte, forceFlag bool) error {
	outputDir := filepath.Dir(filePath)
	if forceFlag {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	} else {
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
	}

	if err := os.WriteFile(filePath, dataArr, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
