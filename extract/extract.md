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
pdfcpu extract -m(ode) image|font|content|page|meta [-p(ages) selectedPages] inFile outDir
````

<br>

### Flags

| name                             | description               | required   | values
|:---------------------------------|:--------------------------|:-----------|:-
| m(ode)                             | component to be extracted | yes | [image](extract_images.md), [font](extract_fonts.md), [content](extract_content.md), [page](extract_pages.md), [meta](extract_metadata.md)
| [p(ages)](../getting_started/page_selection) | page selection  | yes

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

| name   | description      | required
|:-------|:-----------------|:--------
| inFile | PDF input file   | yes
| outDir | output directory | yes
