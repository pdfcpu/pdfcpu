---
layout: default
title: "Add Attachments"
---

# Add Attachments

This command embeds one or more files by attaching them to a PDF input file. Have a look at some [examples](#examples).

## Usage

```
pdfcpu attachments add inFile file [ , desc ]... [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| file...      | one or more files to be attached | yes
| desc         | description         | no

<br>

## Examples

Attach pictures to a coverpage PDF for easy content delivery:

```
$ pdfcpu attachments add album.pdf *.png
adding img1.png
adding img2.png
adding img3.png
```

Attach a file including a description:
```
$ pdfcpu attachments add invoice.pdf 'invoice.doc, my 1st desc'
adding invoice.doc

$ pdfcpu attachments list invoice.pdf
invoice.doc (my 1st desc)
```

<br>

Add an attachment while reading the PDF from stdin:

```sh
$ aws s3 cp s3://acme-archive/report.pdf - \
   | pdfcpu attachments add - source.xlsx > report-with-source.pdf
```
