---
layout: default
---

# Common Flags

The following flags are used by most commands.<br>
Please refer to `pdfcpu help` + *command* for specific usage information.

## verbose, v

Enables logging on the standard output.

## vv

*Very verbose*.<br>
Enables verbose logging on the standard output.<br>
Please use this flag to [report a bug](https://github.com/hhrutter/pdfcpu/issues).

## quiet, q

Disables all output to stdOut.

## pages, p

A comma separated list of expressions defining the [selected pages](page_selection.md) of a PDF input file.

## mode, m

Used by various commands.<br>
Please refer to [validate](../core/validate.md), [extract](../extract/extract.md) and [encrypt](../encrypt/encryptPDF.md) for more information. 

## opw

*Owner password*<br>
This is the password needed to change the access permissions.
It is commonly also referred to as the *master password* or the *permissions password*.
Since some PDF readers skip over blank owner passwords pdfcpu makes this mandatory and non empty if you want to encrypt your documents with pdfcpu.

## upw

*User password*<br>
This is the password needed to open a PDF for reading.
It is also known as the *open doc password*.
