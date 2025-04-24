---
layout: default
---

# List Configuration

The configuration file (config.yml) holds carefully selected default values for various aspects of pdfcpu's operations.

pdfcpu will create the config file together with the config directory on the very first execution of a pdfcpu command.

This command prints the configuration file location followed by its content.

Please also check out the [configuration](../getting_started/config_dir.md) docs.


## Usage

```
pdfcpu config list
```

## Output

```
$ pdfcpu config list
config: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml

#############################
#   Default configuration   #
#############################

# Creation date
created: 2025-04-23 10:30

# version (Do not edit!)
version: v0.10.2 dev

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

# preferred certificate revocation checking mechanism:
# crl
# ocsp
preferredCertRevocationChecker: crl
```