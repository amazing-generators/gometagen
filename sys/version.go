package sys

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"
)

// // // // // // // // // //

type VersionObj struct {
	Raw        string
	Major      string
	Minor      uint64
	Patch      uint64
	Prerelease string
	Build      string
}

// //

func ParseVersion(raw string) (*VersionObj, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("version value is empty")
	}

	versionObj, err := parseVersionValue(raw)
	if err != nil {
		return nil, err
	}

	versionObj.Raw = raw
	return versionObj, nil
}

func ValidateVersion(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("version value is empty")
	}

	_, err := parseVersionValue(raw)
	return err
}

func ValidateGoVersion(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("version value is empty")
	}

	if !semver.IsValid(raw) {
		return fmt.Errorf("invalid Go version: %s", raw)
	}

	return nil
}

func (obj *VersionObj) String() string {
	resultValue := fmt.Sprintf("%s.%d.%d", obj.Major, obj.Minor, obj.Patch)

	if obj.Prerelease != "" {
		resultValue += "-" + obj.Prerelease
	}
	if obj.Build != "" {
		resultValue += "+" + obj.Build
	}

	return resultValue
}

func (obj *VersionObj) IncrementMajor() error {
	obj.Build = ""

	if obj.Minor != 0 || obj.Patch != 0 || obj.Prerelease == "" {
		nextMajorValue, err := incrementMajorValue(obj.Major)
		if err != nil {
			return err
		}

		obj.Major = nextMajorValue
	}

	obj.Minor = 0
	obj.Patch = 0
	obj.Prerelease = ""
	return nil
}

func (obj *VersionObj) IncrementMinor() error {
	obj.Build = ""

	if obj.Patch != 0 || obj.Prerelease == "" {
		obj.Minor++
	}

	obj.Patch = 0
	obj.Prerelease = ""
	return nil
}

func (obj *VersionObj) IncrementPatch() error {
	obj.Build = ""

	if obj.Prerelease == "" {
		obj.Patch++
	}

	obj.Prerelease = ""
	return nil
}

func (obj *VersionObj) PreMajor(preID string) error {
	nextMajorValue, err := incrementMajorValue(obj.Major)
	if err != nil {
		return err
	}

	obj.Build = ""
	obj.Prerelease = ""
	obj.Major = nextMajorValue
	obj.Minor = 0
	obj.Patch = 0

	return obj.incrementPre(preID)
}

func (obj *VersionObj) PreMinor(preID string) error {
	obj.Build = ""
	obj.Prerelease = ""
	obj.Minor++
	obj.Patch = 0

	return obj.incrementPre(preID)
}

func (obj *VersionObj) PrePatch(preID string) error {
	obj.Build = ""
	obj.Prerelease = ""

	if err := obj.IncrementPatch(); err != nil {
		return err
	}

	return obj.incrementPre(preID)
}

func (obj *VersionObj) IncrementPrerelease(preID string) error {
	obj.Build = ""

	if obj.Prerelease == "" {
		if err := obj.IncrementPatch(); err != nil {
			return err
		}
	}

	return obj.incrementPre(preID)
}

func (obj *VersionObj) incrementPre(preID string) error {
	if err := validateIdentifierGroup(preID, "preid"); err != nil {
		return err
	}

	prereleasePartArr := splitIdentifierGroup(obj.Prerelease)
	if len(prereleasePartArr) == 0 {
		prereleasePartArr = []string{"0"}
	} else {
		incrementedFlag := false

		for index := len(prereleasePartArr) - 1; index >= 0; index-- {
			nextValue, ok := incrementUintString(prereleasePartArr[index])
			if !ok {
				continue
			}

			prereleasePartArr[index] = nextValue
			incrementedFlag = true
			break
		}

		if !incrementedFlag {
			prereleasePartArr = append(prereleasePartArr, "0")
		}
	}

	if preID != "" {
		fallbackPartArr := []string{preID, "0"}

		if prereleasePartArr[0] == preID {
			if len(prereleasePartArr) < 2 || !isNumericIdentifier(prereleasePartArr[1]) {
				prereleasePartArr = fallbackPartArr
			}
		} else {
			prereleasePartArr = fallbackPartArr
		}
	}

	obj.Prerelease = strings.Join(prereleasePartArr, ".")
	return nil
}

func parseVersionValue(raw string) (*VersionObj, error) {
	buildSplitArr := strings.SplitN(raw, "+", 2)
	coreWithPre := buildSplitArr[0]
	buildValue := ""
	if len(buildSplitArr) == 2 {
		buildValue = buildSplitArr[1]
		if buildValue == "" {
			return nil, fmt.Errorf("invalid build value: %s", raw)
		}
	}

	preSplitArr := strings.SplitN(coreWithPre, "-", 2)
	coreValue := preSplitArr[0]
	preValue := ""
	if len(preSplitArr) == 2 {
		preValue = preSplitArr[1]
		if preValue == "" {
			return nil, fmt.Errorf("invalid prerelease value: %s", raw)
		}
	}

	corePartsArr := strings.Split(coreValue, ".")
	if len(corePartsArr) != 3 {
		return nil, fmt.Errorf("invalid version format: %s", raw)
	}

	majorValue := strings.TrimSpace(corePartsArr[0])
	if majorValue == "" {
		return nil, fmt.Errorf("version major is empty: %s", raw)
	}
	if strings.ContainsAny(majorValue, " \t\r\n+-") {
		return nil, fmt.Errorf("invalid version major: %s", raw)
	}

	minorValue, err := strconv.ParseUint(corePartsArr[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse version minor: %w", err)
	}

	patchValue, err := strconv.ParseUint(corePartsArr[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse version patch: %w", err)
	}

	if err = validateIdentifierGroup(preValue, "prerelease"); err != nil {
		return nil, err
	}
	if err = validateIdentifierGroup(buildValue, "build"); err != nil {
		return nil, err
	}

	return &VersionObj{
		Major:      majorValue,
		Minor:      minorValue,
		Patch:      patchValue,
		Prerelease: preValue,
		Build:      buildValue,
	}, nil
}

func validateIdentifierGroup(raw string, label string) error {
	if raw == "" {
		return nil
	}

	partArr := strings.Split(raw, ".")
	for _, item := range partArr {
		if item == "" {
			return fmt.Errorf("invalid %s value: %s", label, raw)
		}

		for _, symbol := range item {
			switch {
			case symbol >= '0' && symbol <= '9':
			case symbol >= 'a' && symbol <= 'z':
			case symbol >= 'A' && symbol <= 'Z':
			case symbol == '-':
			default:
				return fmt.Errorf("invalid %s value: %s", label, raw)
			}
		}
	}

	return nil
}

func splitIdentifierGroup(raw string) []string {
	if raw == "" {
		return nil
	}

	return strings.Split(raw, ".")
}

func incrementUintString(raw string) (string, bool) {
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return "", false
	}

	return strconv.FormatUint(value+1, 10), true
}

func isNumericIdentifier(raw string) bool {
	_, ok := incrementUintString(raw)
	return ok
}

func incrementMajorValue(raw string) (string, error) {
	splitIndex := len(raw)
	for splitIndex > 0 {
		symbol := raw[splitIndex-1]
		if symbol < '0' || symbol > '9' {
			break
		}

		splitIndex--
	}

	if splitIndex == len(raw) {
		return "", fmt.Errorf("version major is not incrementable: %s", raw)
	}

	nextValue, ok := incrementUintString(raw[splitIndex:])
	if !ok {
		return "", fmt.Errorf("version major is not incrementable: %s", raw)
	}

	return raw[:splitIndex] + nextValue, nil
}
