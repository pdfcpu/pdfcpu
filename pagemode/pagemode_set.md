---
layout: default
---

# Set Page Mode

This command configures the page mode that shall be used when the document is opened.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu pagemode set inFile value
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
|:-------------|:--------------------|:--------------------------
| inFile       | PDF input file      | yes
| value        | page layout mode    | yes

<br>

### Page Modes

| name           | description
|:---------------|:-------------------------------------------------
| UseNone        | Neither document outline nor thumbnail images visible (default)
| UseOutlines    | Document outline visible
| UseThumbs      | Thumbnail images visible
| FullScreen     | Optional content group panel visible (since PDF 1.5)
| UseOC          | Display the pages two at a time, with odd-numbered pages on the right
| UseAttachments | Attachments panel visible (since PDF 1.6)

<br>

## Examples

Set pagemode for `test.pdf` (case agnostic):

```sh
$ pdfcpu pagemode set test.pdf usethumbs
$ pdfcpu pagemode list test.pdf
UseThumbs
```
