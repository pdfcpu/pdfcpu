---
layout: default
---

# List Attachments

A PDF attachment is any file previously attached to a PDF document. This command outputs a list of all attachments. Have a look at some [examples](#examples).

## Usage

```
pdfcpu attach list [-v(erbose)|vv] [-upw userpw] [-opw ownerpw] inFile
```

<br>

### Flags

| name                             | description       | required
|:---------------------------------|:------------------|:--------
| [verbose](../getting_started.md) | turn on logging   | no
| [vv](../getting_started.md)      | verbose logging   | no
| [upw](../getting_started.md)     | user password     | no
| [opw](../getting_started.md)     | owner password    | no

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
