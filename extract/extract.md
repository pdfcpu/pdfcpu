---
layout: default
---

# Extract

This command lets you extract the following from a PDF file:

* [images](extract_images.md)
* [fonts](extract_fonts.md)
* raw page [content](extract_content.md) in PDF syntax
* actual [pages](extract_pages.md) as single side PDFs
* embedded XML [metadata](extract_metadata.md)

## Usage

```
pdfcpu extract [-v(erbose)|vv] -mode image|font|content|page|meta [-pages pageSelection] [-upw userpw] [-opw ownerpw] inFile outDir
````

<br>

### Flags

| name                             | description               | required   | values
|:---------------------------------|:--------------------------|:-----------|:-
| [verbose](../getting_started.md) | turn on logging           | no
| [vv](../getting_started.md)      | verbose logging           | no
| mode                             | component to be extracted | yes | [image](extract_images.md), [font](extract_fonts.md), [content](extract_content.md), [page](extract_pages.md), [meta](extract_metadata.md)
| [pages](../getting_started/page_selection) | page selection  | yes
| [upw](../getting_started.md)     | user password             | no
| [opw](../getting_started.md)     | owner password            | no

<br>

### Arguments

| name   | description      | required
|:-------|:-----------------|:--------
| inFile | PDF input file   | yes
| outDir | output directory | yes
