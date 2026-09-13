---
layout: default
title: "Common Flags"
---

# Common Flags
The following flags are used by most commands.<br>
Please refer to `pdfcpu [command] -h or --help` for specific usage information.

## --verbose, -v
Enables logging on the standard output.

## -vv
*Very verbose*.<br>
Enables verbose logging on the standard output.<br>
Please use this flag to [report a bug](https://github.com/pdfcpu/pdfcpu/issues).

## --quiet, -q
Suppresses routine command progress and caller-controlled text output. Errors still use `stderr`; PDF output requested with
`-` remains data-only on `stdout`.

## --force
Allow overwriting existing output files, writing to non-empty output directories and resetting configuration without an
interactive confirmation where supported.

## --offline, -o
Disable outgoing HTTP traffic.<br>
For validating links or filling image boxes.<br>

## --conf, -c
Select or disable the [configuration root](/getting_started/config_dir). The configuration tree is stored in the
`pdfcpu` directory below the selected root:

| command    | value
|:-----------|:-----------
| Select     | path
| Disable    | disable

## --unit, -u
Set input display unit to one of:

| unit | value
|:-----------|:-----------
| points | po(ints)
| inches | in(ches)
| centimetres | cm
| millimetres | mm

## --pages, -p
A comma separated list of expressions defining the [selected pages](/getting_started/page_selection) of a PDF input file.

## --mode, -m
Used by various commands.<br>
Please refer to [validate](/core/validate), [extract](/extract/extract), [encrypt](/encrypt/encryptpdf), [pages](/pages/pages_insert), [stamp](/core/stamp) and [watermark](/core/watermark) for more information.

## --opw
*Owner password*<br>
This is the password needed to change the access permissions.
It is commonly also referred to as the *master password* or the *permissions password*.
Since some PDF readers skip over blank owner passwords pdfcpu makes this mandatory and non-empty if you want to encrypt your documents with pdfcpu.

## --upw
*User password*<br>
This is the password needed to open a PDF for reading.
It is also known as the *open doc password*.

## -
Use `-` in PDF input and output positions to read from `stdin` or write to `stdout`.
See [Piping](/getting_started/piping) for examples and the [piping support matrix](/getting_started/piping/#support-matrix) for command-specific stdin/stdout support.

## Password files
Use `--upw-file` or `--opw-file` wherever the corresponding literal flag is available.
Supply each password either directly or through a file, but not both.
<br>Password-file options require a filename; they cannot read from stdin.

One trailing line ending (LF or CRLF), commonly added by text editors, is allowed and ignored,
eg. both "abcde" and "abcde" followed by `<Enter>` supply the password "abcde".
Spaces and any additional line endings are part of the password.

An empty file supplies an empty password.
<br>Encryption and owner-password replacement require a non-empty owner password.
For password changes, keep the existing positional old/new passwords or replace both with the corresponding file pair:

| Command | Old password file | New password file |
|:--|:--|:--|
| `changeupw` | `--upwold-file` | `--upwnew-file` |
| `changeopw` | `--opwold-file` | `--opwnew-file` |

Both file options must be supplied together, using `inFile [outFile]` without positional passwords. Mixing these two forms
is rejected. Both current passwords are authenticated; supply the other password with `--opw`/`--opw-file` for `changeupw`
or `--upw`/`--upw-file` for `changeopw`. Omitting the other password tries an empty password.

```sh
pdfcpu encrypt --opw-file /run/secrets/owner --upw-file /run/secrets/user input.pdf encrypted.pdf
pdfcpu decrypt --upw-file /run/secrets/user encrypted.pdf plain.pdf
pdfcpu validate --upw-file /run/secrets/user encrypted.pdf
pdfcpu optimize --opw-file /run/secrets/owner encrypted.pdf optimized.pdf
```

### Automation: environment variables and files

Both shell environment variables and password files can supply passwords in automated jobs:

```sh
pdfcpu changeupw input.pdf "$OLD_UPW" "$NEW_UPW" --opw "$OPW"
```

The shell expands the variables into literal command arguments before starting pdfcpu. Quoting preserves spaces and
other characters in the values. The expanded passwords may be visible in process arguments or shell tracing output.
pdfcpu does not read these environment variables directly.

With password files, only the filenames appear in command arguments:

```sh
pdfcpu changeupw input.pdf \
  --upwold-file /run/secrets/old-user \
  --upwnew-file /run/secrets/new-user \
  --opw-file /run/secrets/owner
```

File inputs work without shell expansion and suit mounted secrets in containers or other automated environments.
Restrict access to the secret files to the accounts that need them.

