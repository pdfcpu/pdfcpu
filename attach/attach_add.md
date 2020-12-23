---
layout: default
---

# Add Attachments

This command embeds one or more files by attaching them to a PDF input file. Have a look at some [examples](#examples).

## Usage

```
pdfcpu attachments add inFile file...
```

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](../getting_started/common_flags.md)       | config dir      | $path, disable
| [upw](../getting_started/common_flags.md)          | user password   |
| [opw](../getting_started/common_flags.md)          | owner password  |

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
