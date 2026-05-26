package gometagen

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// // // // // // // // // //

const (
	cHashCopyBufferSize = 256 * 1024
	cHashQueueSize      = 1024
)

var hashBufferPool = sync.Pool{
	New: func() any {
		bufferArr := make([]byte, cHashCopyBufferSize)
		return &bufferArr
	},
}

// //

type hashTaskObj struct {
	SourceIndex  int
	SourcePath   string
	RelativePath string
	FilePath     string
}

type hashEntryObj struct {
	SourceIndex  int
	RelativePath string
	SizeValue    int64
	DigestArr    [sha256.Size]byte
}

// //

func hashPath(sourcePath string) (string, error) {
	return hashPaths(context.Background(), []string{sourcePath}, nil, false)
}

func hashPaths(
	contextObj context.Context,
	sourcePathArr []string,
	excludePatternArr []string,
	noAsyncFlag bool,
) (string, error) {
	if contextObj == nil {
		contextObj = context.Background()
	}

	normalizedSourceArr := normalizeHashSourcePathArr(sourcePathArr)

	if len(normalizedSourceArr) == 0 {
		return "", fmt.Errorf("hash source list is empty")
	}

	excludePatternArr = normalizeHashExcludePatternArr(excludePatternArr)

	if noAsyncFlag {
		return hashPathsSequential(contextObj, normalizedSourceArr, excludePatternArr)
	}

	contextObj, cancelFunc := context.WithCancel(contextObj)
	defer cancelFunc()

	taskObjCh := make(chan hashTaskObj, cHashQueueSize)
	entryObjCh := make(chan hashEntryObj, cHashQueueSize)
	errorCh := make(chan error, 1)

	var entryArr []hashEntryObj
	var collectWG sync.WaitGroup

	collectWG.Add(1)
	go func() {
		defer collectWG.Done()

		for entryObj := range entryObjCh {
			entryArr = append(entryArr, entryObj)
		}
	}()

	workerCount := resolveHashWorkerCount()

	var workerWG sync.WaitGroup
	workerWG.Add(workerCount)

	for workerIndex := 0; workerIndex < workerCount; workerIndex++ {
		go func() {
			defer workerWG.Done()

			bufferPtr := hashBufferPool.Get().(*[]byte)
			defer hashBufferPool.Put(bufferPtr)
			copyBufferArr := *bufferPtr

			for {
				select {
				case <-contextObj.Done():
					return
				case taskObj, openFlag := <-taskObjCh:
					if !openFlag {
						return
					}

					entryObj, err := processHashTask(contextObj, taskObj, copyBufferArr)
					if err != nil {
						reportHashError(errorCh, cancelFunc, err)
						return
					}

					select {
					case <-contextObj.Done():
						return
					case entryObjCh <- entryObj:
					}
				}
			}
		}()
	}

	var walkWG sync.WaitGroup
	walkWG.Add(len(normalizedSourceArr))

	for sourceIndex, sourcePath := range normalizedSourceArr {
		go func(sourceIndex int, sourcePath string) {
			defer walkWG.Done()

			if err := enqueueHashTasks(contextObj, taskObjCh, sourceIndex, sourcePath, excludePatternArr); err != nil {
				reportHashError(errorCh, cancelFunc, err)
			}
		}(sourceIndex, sourcePath)
	}

	go func() {
		walkWG.Wait()
		close(taskObjCh)
		workerWG.Wait()
		close(entryObjCh)
	}()

	collectWG.Wait()

	if err := contextObj.Err(); err != nil {
		select {
		case hashErr := <-errorCh:
			return "", hashErr
		default:
			return "", err
		}
	}

	select {
	case err := <-errorCh:
		return "", err
	default:
	}

	return finalizeHashEntries(entryArr)
}

func normalizeHashSourcePathArr(sourcePathArr []string) []string {
	normalizedSourceArr := make([]string, 0, len(sourcePathArr))

	for _, sourcePath := range sourcePathArr {
		sourcePath = strings.TrimSpace(sourcePath)
		if sourcePath == "" {
			continue
		}

		normalizedSourceArr = append(normalizedSourceArr, sourcePath)
	}

	return normalizedSourceArr
}

func resolveHashWorkerCount() int {
	workerCount := runtime.GOMAXPROCS(0) * 4
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > cHashQueueSize {
		workerCount = cHashQueueSize
	}

	return workerCount
}

func hashPathsSequential(
	contextObj context.Context,
	sourcePathArr []string,
	excludePatternArr []string,
) (string, error) {
	entryArr := make([]hashEntryObj, 0)

	bufferPtr := hashBufferPool.Get().(*[]byte)
	defer hashBufferPool.Put(bufferPtr)
	copyBufferArr := *bufferPtr

	for sourceIndex, sourcePath := range sourcePathArr {
		err := walkHashSource(contextObj, sourceIndex, sourcePath, excludePatternArr, func(taskObj hashTaskObj) error {
			entryObj, err := processHashTask(contextObj, taskObj, copyBufferArr)
			if err != nil {
				return err
			}

			entryArr = append(entryArr, entryObj)
			return nil
		})
		if err != nil {
			return "", err
		}
	}

	return finalizeHashEntries(entryArr)
}

func enqueueHashTasks(
	contextObj context.Context,
	taskObjCh chan<- hashTaskObj,
	sourceIndex int,
	sourcePath string,
	excludePatternArr []string,
) error {
	return walkHashSource(contextObj, sourceIndex, sourcePath, excludePatternArr, func(taskObj hashTaskObj) error {
		return sendHashTask(contextObj, taskObjCh, taskObj)
	})
}

func walkHashSource(
	contextObj context.Context,
	sourceIndex int,
	sourcePath string,
	excludePatternArr []string,
	visitFunc func(taskObj hashTaskObj) error,
) error {
	if err := contextObj.Err(); err != nil {
		return err
	}

	info, err := os.Lstat(sourcePath)
	if err != nil {
		return fmt.Errorf("stat hash source: %w", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlinks are not supported in hash source: %s", sourcePath)
	}

	if matchExcludedName(filepath.Base(sourcePath), excludePatternArr) {
		return nil
	}

	if info.IsDir() {
		return filepath.WalkDir(sourcePath, func(filePath string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if err := contextObj.Err(); err != nil {
				return err
			}

			if filePath != sourcePath && matchExcludedName(entry.Name(), excludePatternArr) {
				if entry.IsDir() {
					return fs.SkipDir
				}

				return nil
			}

			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlinks are not supported in hash source: %s", filePath)
			}

			if entry.IsDir() {
				return nil
			}

			if !entry.Type().IsRegular() {
				return fmt.Errorf("unsupported file type in hash source: %s", filePath)
			}

			relativePath, err := filepath.Rel(sourcePath, filePath)
			if err != nil {
				return fmt.Errorf("resolve relative hash path: %w", err)
			}

			return visitFunc(hashTaskObj{
				SourceIndex:  sourceIndex,
				SourcePath:   sourcePath,
				RelativePath: relativePath,
				FilePath:     filePath,
			})
		})
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("hash source must be a regular file or directory: %s", sourcePath)
	}

	return visitFunc(hashTaskObj{
		SourceIndex:  sourceIndex,
		SourcePath:   sourcePath,
		RelativePath: ".",
		FilePath:     sourcePath,
	})
}

func sendHashTask(
	contextObj context.Context,
	taskObjCh chan<- hashTaskObj,
	taskObj hashTaskObj,
) error {
	select {
	case <-contextObj.Done():
		return contextObj.Err()
	case taskObjCh <- taskObj:
		return nil
	}
}

func processHashTask(
	contextObj context.Context,
	taskObj hashTaskObj,
	copyBufferArr []byte,
) (hashEntryObj, error) {
	digestArr, sizeValue, err := hashFile(contextObj, taskObj.FilePath, copyBufferArr)
	if err != nil {
		return hashEntryObj{}, err
	}

	return hashEntryObj{
		SourceIndex:  taskObj.SourceIndex,
		RelativePath: filepath.ToSlash(taskObj.RelativePath),
		SizeValue:    sizeValue,
		DigestArr:    digestArr,
	}, nil
}

func hashFile(
	contextObj context.Context,
	filePath string,
	copyBufferArr []byte,
) ([sha256.Size]byte, int64, error) {
	fileObj, err := os.Open(filePath)
	if err != nil {
		return [sha256.Size]byte{}, 0, fmt.Errorf("open file for hashing: %w", err)
	}

	defer fileObj.Close()

	fileHasher := sha256.New()
	readerObj := &contextReaderObj{
		Context: contextObj,
		Reader:  fileObj,
	}

	sizeValue, err := io.CopyBuffer(fileHasher, readerObj, copyBufferArr)
	if err != nil {
		return [sha256.Size]byte{}, 0, fmt.Errorf("read file for hashing: %w", err)
	}

	var digestArr [sha256.Size]byte
	copy(digestArr[:], fileHasher.Sum(nil))
	return digestArr, sizeValue, nil
}

func finalizeHashEntries(entryArr []hashEntryObj) (string, error) {
	if len(entryArr) == 0 {
		return "", fmt.Errorf("hash source contains no files")
	}

	sort.Slice(entryArr, func(leftIndex int, rightIndex int) bool {
		leftObj := entryArr[leftIndex]
		rightObj := entryArr[rightIndex]

		if leftObj.SourceIndex != rightObj.SourceIndex {
			return leftObj.SourceIndex < rightObj.SourceIndex
		}

		return leftObj.RelativePath < rightObj.RelativePath
	})

	finalHasher := sha1.New()

	for _, entryObj := range entryArr {
		if err := writeHashEntry(finalHasher, entryObj); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(finalHasher.Sum(nil)), nil
}

func writeHashEntry(writer io.Writer, entryObj hashEntryObj) error {
	if _, err := io.WriteString(writer, "file\x00"); err != nil {
		return err
	}

	var sourceIndexArr [8]byte
	binary.BigEndian.PutUint64(sourceIndexArr[:], uint64(entryObj.SourceIndex))

	if _, err := writer.Write(sourceIndexArr[:]); err != nil {
		return err
	}
	if _, err := io.WriteString(writer, "\x00"); err != nil {
		return err
	}
	if _, err := io.WriteString(writer, entryObj.RelativePath); err != nil {
		return err
	}
	if _, err := io.WriteString(writer, "\x00"); err != nil {
		return err
	}

	var sizeArr [8]byte
	binary.BigEndian.PutUint64(sizeArr[:], uint64(entryObj.SizeValue))

	if _, err := writer.Write(sizeArr[:]); err != nil {
		return err
	}
	if _, err := writer.Write(entryObj.DigestArr[:]); err != nil {
		return err
	}

	return nil
}

func matchExcludedName(nameValue string, patternArr []string) bool {
	for _, patternValue := range patternArr {
		matchFlag, err := filepath.Match(patternValue, nameValue)
		if err == nil && matchFlag {
			return true
		}
	}

	return false
}

func reportHashError(errorCh chan<- error, cancelFunc context.CancelFunc, err error) {
	if err == nil {
		return
	}

	cancelFunc()

	select {
	case errorCh <- err:
	default:
	}
}

// //

type contextReaderObj struct {
	Context context.Context
	Reader  io.Reader
}

// //

func (obj *contextReaderObj) Read(dataArr []byte) (int, error) {
	if err := obj.Context.Err(); err != nil {
		return 0, err
	}

	return obj.Reader.Read(dataArr)
}
