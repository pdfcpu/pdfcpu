---
layout: default
---

# List Page Mode

This command prints the page mode that shall be used when the document is opened.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu pagemode list inFile
```

<br>

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name    | description         | required
|:--------|:--------------------|:--------------------------
| inFile  | PDF input file                             | yes

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

List page mode of `test1.pdf`:

```sh
$ pdfcpu pagemode list test1.pdf
No page mode set, PDF viewers will default to "UseNone"
```

List page mode of `test2.pdf`:

```sh
$ pdfcpu pagemode list test2.pdf
FullScreen
```



