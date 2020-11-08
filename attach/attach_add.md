---
layout: default
---

# Add Attachments

This command embeds one or more files by attaching them to a PDF input file. Have a look at some [examples](#examples).

## Usage

```
pdfcpu attachments add [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile file...
```

<br>

### Flags

| name                                          | description       | required
|:----------------------------------------------|:------------------|:--------
| [verbose](../getting_started/common_flags.md) | turn on logging   | no
| [vv](../getting_started/common_flags.md)      | verbose logging   | no
| [quiet](../getting_started/common_flags.md)   | quiet mode        | no
| [upw](../getting_started/common_flags.md)     | user password     | no
| [opw](../getting_started/common_flags.md)     | owner password    | no

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
| file...      | one or more files to be attached | yes

<br>

## Examples

Attach pictures to a coverpage PDF for easy content delivery:

```
pdfcpu attach add album.pdf *.png
adding img1.png
adding img2.png
adding img3.png
```

Attach a file including a description:
```
pdfcpu attach add invoice.pdf 'invoice.doc, my 1st desc'
adding invoice.doc

pdfcpu attach list invoice.pdf
invoice.doc (my 1st desc)
```
