---
layout: default
---

# Remove Attachments

This command removes previously attached files from a PDF document. Have a look at some [examples](#examples).

## Usage

```
pdfcpu attachments remove [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile [file...]
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
| file...      | one or more attachments to be removed | yes

<br>

## Examples

Remove a specific attachment from container.pdf:

```sh
pdfcpu attach remove container.pdf pdfcpu.zip
removing pdfcpu.zip
```

<br>

Remove all attachments:

```sh
pdfcpu attach remove container.pdf
removing all attachments
```