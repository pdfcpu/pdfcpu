---
layout: default
---

# Insert Pages

This command inserts empty pages before selected pages or before every page if no page is selected.
Have a look at some [examples](#examples).

## Usage

```
pdfcpu pages insert [-v(erbose)|vv] [-pages pageSelection] [-upw userpw] [-opw ownerpw] inFile [outFile]
```

<br>

### Flags

| name                             | description       | required
|:---------------------------------|:------------------|:--------
| [verbose](../getting_started.md) | turn on logging   | no
| [vv](../getting_started.md)      | verbose logging   | no
| [pages](../getting_started/page_selection) | page selection  | no 
| [upw](../getting_started.md)     | user password     | no
| [opw](../getting_started.md)     | owner password    | no

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
| outFile...   | PDF output file     | no

<br>

## Examples

Insert an empty page before every page of `notes.pdf`. This way you get a PDF that gives you space for adding annotations for pages:

```sh
pdfcpu page insert notes.pdf
inserting pages into notes.pdf ...
writing notes_new.pdf ...
```

<br>

Insert an empty page before pages 1 to 5:

```sh
pdfcpu page insert -pages 1-5 notes1.pdf notes2.pdf
inserting pages into notes1.pdf ...
writing notes2.pdf ...
```