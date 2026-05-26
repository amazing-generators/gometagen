package sys

import (
	"fmt"
	"strings"
	"sync"
)

// // // // // // // // // //

type Obj struct {
	sourcePath   string
	manifestPath string
	mutexObj     sync.Mutex
	manifestObj  *ValuesObj
	manifestErr  error
	loadedFlag   bool
}

// //

func New(sourcePath string) (*Obj, error) {
	resolvedSourcePath, manifestPath, err := resolveManifestPath(sourcePath)
	if err != nil {
		return nil, err
	}

	return &Obj{
		sourcePath:   resolvedSourcePath,
		manifestPath: manifestPath,
	}, nil
}

func (obj *Obj) SourcePath() string {
	return obj.sourcePath
}

func (obj *Obj) ManifestPath() string {
	return obj.manifestPath
}

func (obj *Obj) Manifest() (*ValuesObj, error) {
	obj.mutexObj.Lock()
	defer obj.mutexObj.Unlock()

	if !obj.loadedFlag {
		obj.manifestObj, obj.manifestErr = readManifestFile(obj.manifestPath)
		obj.loadedFlag = true
	}

	if obj.manifestErr != nil {
		return nil, obj.manifestErr
	}

	return cloneValues(obj.manifestObj), nil
}

func (obj *Obj) Validate() error {
	_, err := obj.Manifest()
	return err
}

func (obj *Obj) Name() (string, error) {
	valuesObj, err := obj.Manifest()
	if err != nil {
		return "", err
	}

	if err = validateNameLen(valuesObj.Name); err != nil {
		return "", err
	}

	return valuesObj.Name, nil
}

func (obj *Obj) Version() (*VersionObj, error) {
	valuesObj, err := obj.Manifest()
	if err != nil {
		return nil, err
	}

	return ParseVersion(valuesObj.Ver)
}

func (obj *Obj) VersionString() (string, error) {
	versionObj, err := obj.Version()
	if err != nil {
		return "", err
	}

	return versionObj.Raw, nil
}

func (obj *Obj) Major() (string, error) {
	versionObj, err := obj.Version()
	if err != nil {
		return "", err
	}

	return versionObj.Major, nil
}

func (obj *Obj) Minor() (uint64, error) {
	versionObj, err := obj.Version()
	if err != nil {
		return 0, err
	}

	return versionObj.Minor, nil
}

func (obj *Obj) Patch() (uint64, error) {
	versionObj, err := obj.Version()
	if err != nil {
		return 0, err
	}

	return versionObj.Patch, nil
}

func (obj *Obj) IncrementMinor() (string, error) {
	versionObj, err := obj.Version()
	if err != nil {
		return "", err
	}

	if err = versionObj.IncrementMinor(); err != nil {
		return "", err
	}

	return obj.writeVersion(versionObj)
}

func (obj *Obj) IncrementPatch() (string, error) {
	versionObj, err := obj.Version()
	if err != nil {
		return "", err
	}

	if err = versionObj.IncrementPatch(); err != nil {
		return "", err
	}

	return obj.writeVersion(versionObj)
}

func (obj *Obj) IncrementMajor() (string, error) {
	versionObj, err := obj.Version()
	if err != nil {
		return "", err
	}

	if err = versionObj.IncrementMajor(); err != nil {
		return "", err
	}

	return obj.writeVersion(versionObj)
}

func (obj *Obj) PreMajor(preID string) (string, error) {
	versionObj, err := obj.Version()
	if err != nil {
		return "", err
	}

	if err = versionObj.PreMajor(preID); err != nil {
		return "", err
	}

	return obj.writeVersion(versionObj)
}

func (obj *Obj) PreMinor(preID string) (string, error) {
	versionObj, err := obj.Version()
	if err != nil {
		return "", err
	}

	if err = versionObj.PreMinor(preID); err != nil {
		return "", err
	}

	return obj.writeVersion(versionObj)
}

func (obj *Obj) PrePatch(preID string) (string, error) {
	versionObj, err := obj.Version()
	if err != nil {
		return "", err
	}

	if err = versionObj.PrePatch(preID); err != nil {
		return "", err
	}

	return obj.writeVersion(versionObj)
}

func (obj *Obj) Prerelease(preID string) (string, error) {
	versionObj, err := obj.Version()
	if err != nil {
		return "", err
	}

	if err = versionObj.IncrementPrerelease(preID); err != nil {
		return "", err
	}

	return obj.writeVersion(versionObj)
}

func (obj *Obj) ValidateVersion() error {
	valuesObj, err := obj.Manifest()
	if err != nil {
		return err
	}

	return ValidateVersion(valuesObj.Ver)
}

func (obj *Obj) writeVersion(versionObj *VersionObj) (string, error) {
	valuesObj, err := obj.Manifest()
	if err != nil {
		return "", err
	}

	versionObj.Raw = versionObj.String()
	if err = ValidateVersion(versionObj.Raw); err != nil {
		return "", err
	}

	valuesObj.Ver = versionObj.Raw

	if err = writeManifestFile(obj.manifestPath, valuesObj, true); err != nil {
		return "", err
	}

	obj.mutexObj.Lock()
	obj.manifestObj = cloneValues(valuesObj)
	obj.manifestErr = nil
	obj.loadedFlag = true
	obj.mutexObj.Unlock()

	return versionObj.Raw, nil
}

func Create(targetPath string, format string, name string, golandFlag bool, force bool) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("manifest name is empty")
	}
	if err := validateNameLen(name); err != nil {
		return "", err
	}

	versionValue := "0.0.1"
	if golandFlag {
		versionValue = "v0.0.1"
	}

	return CreateManifest(targetPath, format, &ValuesObj{
		Name: name,
		Ver:  versionValue,
	}, force)
}

func validateNameLen(value string) error {
	if len([]rune(value)) > 40 {
		return fmt.Errorf("manifest name exceeds 40 characters")
	}

	return nil
}
