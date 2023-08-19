---
layout: default
---

# Create

allows you to generate PDF via JSON

* Create a new PDF file based on JSON input with optional form definition/data.

* Repeatedly add pages to an existing PDF file serving an incremental PDF generation approach.

* Append to content of specific pages.

* Include page headers and footers.

* Include boxes, images, text, tables.

* Create a PDF form by including date fields, text fields, checkboxes, radio button groups, comboboxes and listboxes.

* Supports Unicode / pdfcpu user fonts (installed Open/TrueType fonts).

* Use layout guides and visible crop/content box through out the layout process.

* Choose your preferred layout coordinate system.

<br>


## Usage

```
pdfcpu create inFileJSON [inFile] outFile
```

<br>


### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](../getting_started/common_flags.md)    | config dir      | $path, disable
| [upw](../getting_started/common_flags.md)       | user password   |
| [opw](../getting_started/common_flags.md)       | owner password  |

<br>

### Arguments

| name         | description             | required |
|:-------------|:------------------------|:---------|
| inFileJSON   | JSON input file         | yes
| inFile       | PDF input file          | no
| outFile      | PDF output file         | yes

<br>

## Futher Reference

```
$ pdfcpu help create
``` 

and

```
pdfcpu/pkg/testdata/json/*
pdfcpu/pkg/samples/create/*
```