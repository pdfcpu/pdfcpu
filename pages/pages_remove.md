---
layout: default
---

# Remove Pages

This command removes all selected pages from a PDF file.
Have a look at some [examples](#examples).

## Usage

```
pdfcpu pages remove [-v(erbose)|vv] -pages pageSelection [-upw userpw] [-opw ownerpw] inFile [outFile]
```

<br>

### Flags

| name                             | description       | required
|:---------------------------------|:------------------|:--------
| [verbose](../getting_started/common_flags.md) | turn on logging   | no
| [vv](../getting_started/common_flags.md)      | verbose logging   | no
| [pages](../getting_started/page_selection) | page selection  | yes
| [upw](../getting_started/common_flags.md)     | user password     | no
| [opw](../getting_started/common_flags.md)    | owner password    | no

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
| outFile...   | PDF output file     | no

<br>

## Examples

Remove pages 1-3 and 5 from `notes.pdf`:

```sh
pdfcpu pag rem -pages 1-3,5 notes.pdf
removing pages from notes.pdf ...
writing notes_new.pdf ...
```