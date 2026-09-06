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
