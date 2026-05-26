package gometagen

import (
	"fmt"
	"strings"

	projectgit "github.com/amazing-generators/gometagen/git"
	projectsys "github.com/amazing-generators/gometagen/sys"
)

// // // // // // // // // //

func validateMetaName(value string) error {
	if len([]rune(value)) > 40 {
		return fmt.Errorf("manifest name exceeds 40 characters")
	}

	return nil
}

func buildMetaObj(manifestObj *projectsys.ValuesObj, nowValue string) (*MetaObj, error) {
	if err := validateMetaName(manifestObj.Name); err != nil {
		return nil, err
	}

	versionObj, err := projectsys.ParseVersion(manifestObj.Ver)
	if err != nil {
		return nil, err
	}

	if versionObj.Minor > cMaxVersionPart {
		return nil, fmt.Errorf("version minor exceeds uint16: %d", versionObj.Minor)
	}
	if versionObj.Patch > cMaxVersionPart {
		return nil, fmt.Errorf("version patch exceeds uint16: %d", versionObj.Patch)
	}

	return &MetaObj{
		Name:         manifestObj.Name,
		DateUpdate:   nowValue,
		Hash:         "",
		Version:      versionObj.Raw,
		VersionMajor: versionObj.Major,
		VersionMinor: uint16(versionObj.Minor),
		VersionPatch: uint16(versionObj.Patch),
	}, nil
}

// //

func collectNormalized(config normalizedConfigObj) (*projectsys.ValuesObj, *MetaObj, string, string, string, error) {
	sysObj, err := projectsys.New(config.SourcePath)
	if err != nil {
		return nil, nil, "", "", "", err
	}

	manifestObj, err := sysObj.Manifest()
	if err != nil {
		return nil, nil, "", "", "", err
	}

	metaObj, err := buildMetaObj(manifestObj, config.Now.Format(cDateLayout))
	if err != nil {
		return nil, nil, "", "", "", err
	}

	hashValue, hashModeValue, hashPointValue, err := collectHash(config, sysObj.SourcePath())
	if err != nil {
		return nil, nil, "", "", "", err
	}

	metaObj.Hash = hashValue

	return manifestObj, metaObj, sysObj.ManifestPath(), hashModeValue, hashPointValue, nil
}

func collectHash(config normalizedConfigObj, sourcePath string) (string, string, string, error) {
	if len(config.HashSourceArr) > 0 {
		hashValue, err := hashPaths(config.Context, config.HashSourceArr, config.HashExcludeArr, config.HashNoAsync)
		if err != nil {
			return "", "", "", err
		}

		return hashValue, "content", strings.Join(config.HashSourceArr, ", "), nil
	}

	gitObj, err := projectgit.New(sourcePath)
	if err != nil {
		return "", "", "", err
	}

	hashValue, err := gitObj.HashContext(config.Context)
	if err != nil {
		return "", "", "", fmt.Errorf("read git hash: %w", err)
	}

	return hashValue, "git", sourcePath, nil
}
