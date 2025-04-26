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

### [Common Flags](../getting_started/common_flags)

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
