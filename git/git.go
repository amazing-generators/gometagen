package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// // // // // // // // // //

var ErrGitNotFound = errors.New("git executable not found")

// //

type Obj struct {
	rootPath string
}

// //

func New(rootPath string) (*Obj, error) {
	if strings.TrimSpace(rootPath) == "" {
		return nil, fmt.Errorf("root path is empty")
	}

	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("resolve root path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("stat root path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root path is not a directory: %s", absPath)
	}

	return &Obj{rootPath: absPath}, nil
}

func (obj *Obj) Branch() (string, error) {
	return obj.runTrimmedContext(context.Background(), "rev-parse", "--abbrev-ref", "HEAD")
}

func (obj *Obj) Hash() (string, error) {
	return obj.HashContext(context.Background())
}

func (obj *Obj) HashContext(contextObj context.Context) (string, error) {
	return obj.runTrimmedContext(contextObj, "rev-parse", "HEAD^{commit}")
}

func (obj *Obj) Tag() (string, error) {
	return obj.runTrimmedContext(context.Background(), "describe", "--tags", "--abbrev=0")
}

func (obj *Obj) AddCommitHook() error {
	return obj.writeHookFile("commit-msg", CommitMsgHookArr)
}

func (obj *Obj) AddPushHook() error {
	return obj.writeHookFile("pre-push", PrePushHookArr)
}

func (obj *Obj) DelCommitHook() error {
	return obj.writeHookFile("commit-msg", NoopHookArr)
}

func (obj *Obj) DelPushHook() error {
	return obj.writeHookFile("pre-push", NoopHookArr)
}

func (obj *Obj) HooksPath() (string, error) {
	hooksPath, err := obj.runTrimmedContext(context.Background(), "rev-parse", "--git-path", "hooks")
	if err != nil {
		return "", err
	}

	if filepath.IsAbs(hooksPath) {
		return hooksPath, nil
	}

	return filepath.Join(obj.rootPath, hooksPath), nil
}

func (obj *Obj) writeHookFile(fileName string, dataArr []byte) error {
	hooksPath, err := obj.HooksPath()
	if err != nil {
		return err
	}

	if err = os.MkdirAll(hooksPath, 0755); err != nil {
		return fmt.Errorf("create hooks directory: %w", err)
	}

	filePath := filepath.Join(hooksPath, fileName)
	if err = os.WriteFile(filePath, cloneBytes(dataArr), 0755); err != nil {
		return fmt.Errorf("write hook file %s: %w", filePath, err)
	}

	return nil
}

func (obj *Obj) runTrimmedContext(contextObj context.Context, argsArr ...string) (string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", ErrGitNotFound
	}

	if contextObj == nil {
		contextObj = context.Background()
	}

	commandObj := exec.CommandContext(contextObj, "git", argsArr...)
	commandObj.Dir = obj.rootPath
	dataArr, err := commandObj.Output()
	if err != nil {
		return "", fmt.Errorf("run git %s: %w", strings.Join(argsArr, " "), err)
	}

	value := strings.TrimSpace(string(dataArr))
	if value == "" {
		return "", fmt.Errorf("empty git output for %s", strings.Join(argsArr, " "))
	}

	return value, nil
}

func cloneBytes(dataArr []byte) []byte {
	resultArr := make([]byte, len(dataArr))
	copy(resultArr, dataArr)
	return resultArr
}
