---
layout: default
title: "List Configuration"
---

# List Configuration

The configuration file (config.yml) holds carefully selected default values for various aspects of pdfcpu's operations.

In automatic mode, pdfcpu creates the config file and configuration directory when a configuration-dependent command
requires them. Configuration-free commands such as `version` do not initialize configuration.

This command prints the configuration file location followed by its content exactly as stored. Command-line overrides are
not reflected. Use [config inspect](/config/config_inspect) for selected paths and configured policy.

Before printing, it verifies that the selected file uses the schema required by this pdfcpu build. An older schema produces
the standard `configuration reset required` message and a reset command; a newer schema asks for a newer pdfcpu version.
The command does not modify an incompatible file.

See also [inspect configuration](/config/config_inspect), [validate configuration](/config/config_validate) and the
[configuration directory](/getting_started/config_dir) documentation.


## Usage

```
pdfcpu config list
```

## Output

```
$ pdfcpu config list
config: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml

# configuration schema version, independent of the pdfcpu version.
schemaVersion: 1

# toggle for inFilename extension check (.pdf)
checkFileNameExt: true

reader15: true

decodeAllStreams: false

# validationMode:
# ValidationStrict,
# ValidationRelaxed,
validationMode: ValidationRelaxed

# validate cross reference table right before writing.
postProcessValidate: true

# eol for writing:
# EolLF
# EolCR
# EolCRLF
eol: EolLF

writeObjectStream: true
writeXRefStream: true
encryptUsingAES: true

# encryptKeyLength: max 256
encryptKeyLength: 256

# permissions for encrypted files:
# 0xF0C3 (PermissionsNone)
# 0xF8C7 (PermissionsPrint)
# 0xFFFF (PermissionsAll)
# See more at model.PermissionFlags and PDF spec table 22
permissions: 0xF0C3

# displayUnit:
# points
# inches
# cm
# mm
unit: points

# timestamp format: yyyy-mm-dd hh:mm
# Switch month and year by using: 2006-02-01 15:04
# See more at https://pkg.go.dev/time@go1.22#pkg-constants
timestampFormat: 2006-01-02 15:04

# date format: yyyy-mm-dd
dateFormat: 2006-01-02

# toggle optimization.
optimize: true

# optimize page resources via content stream analysis.
optimizeResourceDicts: true

# optimize duplicate content streams across pages.
optimizeDuplicateContentStreams: false

# merge creates bookmarks.
createBookmarks: true

# viewer is expected to supply appearance streams for form fields.
needAppearances: false

# internet availability.
offline: false

# http timeout in seconds.
timeout: 5

# http timeout in seconds for CRL revocation checking.
timeoutCRL: 10

# http timeout in seconds for OCSP revocation checking.
timeoutOCSP: 10

# private hosts explicitly trusted for CRL and OCSP requests.
# example: [ocsp.example.corp, crl.example.corp]
allowedRevocationHosts: []

# preferred certificate revocation checking mechanism:
# crl
# ocsp
preferredCertRevocationChecker: crl

# limit form field content for display purposes when using pdfcpu form list.
# if > 0 affects the columns AltName, Default and Value.
formFieldListMaxColWidth: 0

# encoded stream bytes read from a PDF.
maxStreamBytes: 512 MB

# decoded stream bytes produced by filters.
maxDecodeBytes: 512 MB

# decoded/rendered image dimensions and buffers.
maxImagePixels: 100 MP
maxImageBytes: 512 MB
```

## Validation mode

`validationMode` sets the default validation policy used when reading PDFs:

* `ValidationStrict` rejects specification violations detected by pdfcpu's implemented validation rules.
* `ValidationRelaxed` is the default. It applies the same validation baseline while permitting supported real-world PDF
  writer deviations and safe, unambiguous recovery.

Both modes use the same security and resource limits. The setting controls conformance decisions; it does not certify
complete ISO 32000 compliance or disable all low-level reader recovery.

See [Validate](/core/validate) for the full mode definitions and current validation scope.
