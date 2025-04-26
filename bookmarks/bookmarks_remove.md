---
layout: default
---

# Remove Bookmarks

This command removes all bookmarks.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu bookmarks remove inFile [outFile]
```

<br>

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
| outFile      | PDF output file     | no

<br>

## Examples

 Remove all bookmarks:

```
$ pdfcpu bookmarks remove bookmarkSimple.pdf

$ pdfcpu bookmarks list bookmarkSimple.pdf
no bookmarks available
```
