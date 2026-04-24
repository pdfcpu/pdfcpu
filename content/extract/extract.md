---
layout: default
title: "Extract"
---

# Extract

This command lets you extract the following from a PDF file:

* [images](/extract/extract_images)
* [fonts](/extract/extract_fonts)
* raw page [content](/extract/extract_content) in PDF syntax
* actual [pages](/extract/extract_pages) as single side PDFs
* embedded XML [metadata](/extract/extract_metadata)

## Usage

```
pdfcpu extract -m(ode) image|font|content|page|meta [-p(ages) selectedPages] inFile outDir
````

<br>

### Flags

| name                             | description               | required   | values
|:---------------------------------|:--------------------------|:-----------|:-
| m(ode)                             | component to be extracted | yes | [image](/extract/extract_images), [font](/extract/extract_fonts), [content](/extract/extract_content), [page](/extract/extract_pages), [meta](/extract/extract_metadata)
| [p(ages)](/getting_started/page_selection) | page selection  | yes

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name   | description      | required
|:-------|:-----------------|:--------
| inFile | PDF input file   | yes
| outDir | output directory | yes
