# gometagen

`gometagen` is a small command-line tool that keeps project metadata in a manifest and works with it in three ways:

- reads fields from the manifest
- increments the version in npm-like style
- generates Go/JSON/YAML/template output from the manifest plus a hash

## Install

### Prebuilt binaries

Every [GitHub release](https://github.com/amazing-generators/gometagen/releases)
ships statically linked, stripped binaries for all supported platforms. Download
the one matching your OS/architecture, make it executable, and put it on your
`PATH`:

```bash
chmod +x gometagen-linux-amd64
./gometagen-linux-amd64 --help
```

Asset names follow the pattern `gometagen-<os>-<arch>` (Windows assets end with
`.exe`, ARM assets use the `armv6`/`armv7` suffix).

#### Supported platforms

| OS      | Architectures                                                                                                             |
|---------|---------------------------------------------------------------------------------------------------------------------------|
| Windows | `amd64`, `arm64`, `armv7`, `386`                                                                                          |
| Linux   | `amd64`, `arm64`, `armv7`, `armv6`, `386`, `mips`, `mipsle`, `mips64`, `mips64le`, `ppc64`, `ppc64le`, `riscv64`, `s390x` |
| macOS   | `amd64` (Intel), `arm64` (Apple Silicon)                                                                                  |
| FreeBSD | `amd64`, `arm64`, `armv7`, `armv6`, `386`                                                                                 |
| OpenBSD | `amd64`, `arm64`, `armv7`, `armv6`, `386`                                                                                 |
| NetBSD  | `amd64`, `arm64`, `armv7`, `armv6`, `386`                                                                                 |
| Android | `amd64`, `arm64`, `armv7`, `386`                                                                                          |

### Install with Go

Install the binary onto your `PATH` from source:

```bash
go install github.com/amazing-generators/gometagen/cmd/gometagen@latest
```

After that the `gometagen` command is available directly:

```bash
gometagen --help
```

### Run without installing

You can run the tool ad-hoc without adding it to `PATH`:

```bash
go run github.com/amazing-generators/gometagen/cmd/gometagen@latest --help
```

All examples below use the installed `gometagen` command. Replace it with
`go run github.com/amazing-generators/gometagen/cmd/gometagen@latest` if you
prefer the ad-hoc form.

## Canonical Layout

The default project layout is:

```text
.
├── _run/
│   └── values.yml
└── ...
```

Canonical manifest file:

```yaml
name: gometagen
ver: v0.1.0
```

## Quick Start

Create the manifest in `_run`:

```bash
gometagen manifest-init -out ./_run -format yml -name my-project -goland -force
```

Read fields:

```bash
gometagen manifest get -source ./_run/values.yml -field name
gometagen version print -source ./_run/values.yml
```

Increment the version:

```bash
gometagen version patch -source ./_run/values.yml
gometagen version prerelease -source ./_run/values.yml --preid beta
```

Generate metadata:

```bash
gometagen generate -source . -format json -stdout
gometagen generate -source . -hash-source . -format go -out ./meta_gen.go
```

## Manifest Lookup

If `-source` points to a file, that file is used directly.

If `-source` points to a directory, `gometagen` searches in this order set:

- `values.json`
- `values.yml`
- `values.yaml`
- `_run/values.json`
- `_run/values.yml`
- `_run/values.yaml`
- `_run/values/values.json`
- `_run/values/values.yml`
- `_run/values/values.yaml`

If more than one candidate exists, the command fails.

## Reading

Read the raw manifest fields:

```bash
gometagen manifest get -source ./_run/values.yml -field name
gometagen manifest get -source ./_run/values.yml -field ver
```

Read the current version:

```bash
gometagen version print -source ./_run/values.yml
```

Validate the manifest:

```bash
gometagen validate -source ./_run/values.yml
```

Output is always `true` or `false`.

## Version Increments

All version commands work only with the manifest file.

They do not create git tags, do not create commits, and do not touch anything except the manifest version field.

Supported commands:

- `version major`
- `version minor`
- `version patch`
- `version premajor`
- `version preminor`
- `version prepatch`
- `version prerelease`

Examples:

```bash
gometagen version major -source ./_run/values.yml
gometagen version minor -source ./_run/values.yml
gometagen version patch -source ./_run/values.yml
```

Prerelease examples:

```bash
gometagen version premajor -source ./_run/values.yml --preid beta
gometagen version preminor -source ./_run/values.yml --preid beta
gometagen version prepatch -source ./_run/values.yml --preid beta
gometagen version prerelease -source ./_run/values.yml --preid beta
```

Typical transitions:

- `v1.2.3` + `patch` -> `v1.2.4`
- `v1.2.3` + `minor` -> `v1.3.0`
- `v1.2.3` + `major` -> `v2.0.0`
- `v1.2.3` + `prepatch --preid beta` -> `v1.2.4-beta.0`
- `v1.2.3-beta.0` + `prerelease --preid beta` -> `v1.2.3-beta.1`
- `v1.2.3-beta.1` + `prerelease --preid alpha` -> `v1.2.3-alpha.0`
- `v1.2.3-beta.1` + `patch` -> `v1.2.3`
- `v1.2.0-beta.1` + `minor` -> `v1.2.0`
- `v1.0.0-beta.1` + `major` -> `v1.0.0`

### `--preid`

`--preid` is supported for:

- `version premajor`
- `version preminor`
- `version prepatch`
- `version prerelease`

Examples:

```bash
gometagen version prerelease -source ./_run/values.yml --preid beta
gometagen version prerelease -source ./_run/values.yml --preid rc
gometagen version preminor -source ./_run/values.yml --preid release-candidate
```

If `--preid` is omitted, the prerelease numeric form is used:

- `v1.2.3` + `prerelease` -> `v1.2.4-0`
- `v1.2.3-0` + `prerelease` -> `v1.2.3-1`

### Major-Part Caveat

This project intentionally allows the first version component to be a string like:

- `v1`
- `1`
- `release9`
- `release`

`major` and `premajor` require an incrementable numeric tail in that first component.

This means:

- `v1.2.3` -> major bump works
- `release9.2.3` -> major bump works
- `release.2.3` -> major bump fails

Non-major commands such as `minor`, `patch`, `preminor`, `prepatch`, and `prerelease` still work with `release.2.3`.

## Generation

`generate` builds metadata from:

- manifest `name`
- manifest `ver`
- current date
- hash value

Supported formats:

- `go`
- `json`
- `yaml`
- `template`

Examples:

```bash
gometagen generate -source . -format go -out ./meta_gen.go
gometagen generate -source . -format json -stdout
gometagen generate -source . -format yaml -out ./artifacts -force
```

Template mode:

```bash
gometagen template-init -out ./gometagen.tmpl -force
gometagen generate -source . -template ./gometagen.tmpl -out ./meta.txt
```

`-format` and `-template` are mutually exclusive: passing `-format go|json|yaml`
together with `-template` is an error rather than being silently ignored.

## Hash Modes

By default `gometagen` uses the current git commit hash.

If one or more `-hash-source` flags are provided, the tool switches to content-hash mode.

Examples:

```bash
gometagen generate -source . -hash-source . -stdout
gometagen generate -source . -hash-source ./cmd -hash-source ./templates -stdout
gometagen generate -source . -hash-source . -hash-exclude dist -hash-exclude '*.exe' -stdout
```

Default exclusions:

- `tmp`
- `target`

Content-hash mode fails if the resolved source set contains no files.

### Sequential Hashing

Content-hash mode is concurrent by default.

If you need a single-worker deterministic execution path without concurrent hashing, use:

```bash
gometagen generate -source . -hash-source . -hash-no-async -stdout
```

`-hash-no-async` only affects content-hash mode.

## Go Output

The built-in Go renderer produces these constants:

- `Name`
- `DateUpdate`
- `Hash`
- `Version`
- `VersionMajor`
- `VersionMinor`
- `VersionPatch`

`VersionMinor` and `VersionPatch` are constrained to `uint16` in every output
format (go, json, yaml, template).

If either value is greater than `65535`, generation fails before any output is produced.

The generated date (`DateUpdate`) uses ISO 8601 (`YYYY-MM-DD`).

## Command Summary

Root command without a subcommand behaves like `generate`.

Main commands:

- `generate`
- `validate`
- `manifest-init`
- `manifest get`
- `template-init`
- `version print`
- `version major`
- `version minor`
- `version patch`
- `version premajor`
- `version preminor`
- `version prepatch`
- `version prerelease`
- `git branch`
- `git add-commit-hook`
- `git add-push-hook`
- `git del-commit-hook`
- `git del-push-hook`

For built-in CLI help:

```bash
gometagen --help
gometagen generate --help
gometagen version prerelease --help
```
