---
layout: default
title: "Remove Attachments"
---

# Remove Attachments

This command removes previously attached files from a PDF document. Have a look at some [examples](#examples).

## Usage

```
pdfcpu attachments remove inFile [ file... ] [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| file...      | one or more attachments to be removed | yes

<br>

## Examples

Remove a specific attachment from container.pdf:

```sh
$ pdfcpu attachments remove container.pdf pdfcpu.zip
removing pdfcpu.zip
```

<br>

Remove all attachments:

```sh
$ pdfcpu attachments remove container.pdf
removing all attachments
```

<br>

Remove an attachment while reading the PDF from stdin:

```sh
$ cat report-with-source.pdf \
   | pdfcpu attachments remove - source.xlsx > report-without-source.pdf
```

<br>
