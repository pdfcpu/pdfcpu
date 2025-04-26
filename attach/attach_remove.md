---
layout: default
---

# Remove Attachments

This command removes previously attached files from a PDF document. Have a look at some [examples](#examples).

## Usage

```
pdfcpu attachments remove inFile [file...]
```

<br>

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
| file...      | one or more attachments to be removed | yes

<br>

## Examples

Remove a specific attachment from container.pdf:

```sh
$ pdfcpu attach remove container.pdf pdfcpu.zip
removing pdfcpu.zip
```

<br>

Remove all attachments:

```sh
$ pdfcpu attach remove container.pdf
removing all attachments
```