---
layout: default
---

# List Attachments

A PDF attachment is any file previously attached to a PDF document. This command outputs a list of all attachments. Have a look at some [examples](#examples).

## Usage

```
pdfcpu attachments list [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile
```

<br>

### Flags

| name                                          | description       | required
|:----------------------------------------------|:------------------|:--------
| [verbose](../getting_started/common_flags.md) | turn on logging   | no
| [vv](../getting_started/common_flags.md)      | verbose logging   | no
| [quiet](../getting_started/common_flags.md)   | verbose logging   | no
| [upw](../getting_started/common_flags.md)     | user password     | no
| [opw](../getting_started/common_flags.md)     | owner password    | no

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes

<br>

## Examples

 List all attachments embedded into `container.pdf`. You may attach any file to a PDF document:

```sh
pdfcpu attach list container.pdf
forest.jpg
pdfcpu.zip
invoice.pdf
```
