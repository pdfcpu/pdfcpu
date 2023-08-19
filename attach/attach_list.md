---
layout: default
---

# List Attachments

A PDF attachment is any file previously attached to a PDF document. This command outputs a list of all attachments. Have a look at some [examples](#examples).

## Usage

```
pdfcpu attachments list inFile
```

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](..getting_started/common_flags.md)       | config dir      | $path, disable
| [upw](..getting_started/common_flags.md)          | user password   |
| [opw](..getting_started/common_flags.md)          | owner password  |

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes

<br>

## Examples

 List all attachments embedded into `container.pdf`. You may attach any file to a PDF document.
 Any available attachment description will be shown in braces:

```sh
$ pdfcpu attach list container.pdf
forest.jpg
pdfcpu.zip (description1)
invoice.pdf (description2)
```
