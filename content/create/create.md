---
layout: default
title: "Create"
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
pdfcpu create inFileJSON [ inFile ] outFile [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description             | required |
|:-------------|:------------------------|:---------|
| inFileJSON   | JSON input file         | yes
| inFile       | PDF input file, use `-` to read from stdin | no
| outFile      | PDF output file, use `-` to write to stdout      | yes

<br>

## Further Reference

```
$ pdfcpu create --help
``` 

and

```
pdfcpu/pkg/testdata/json/*
pdfcpu/pkg/samples/create/*
```

<br>

Create a PDF and write it to stdout:

```sh
$ pdfcpu create invoice.json - \
   | aws s3 cp - s3://acme-billing/invoice.pdf
```

<br>

Overlay generated content onto a streamed input PDF:

```sh
$ aws s3 cp s3://acme-forms/template.pdf - \
   | pdfcpu create overlay.json - - \
   | aws s3 cp - s3://acme-forms/template-filled.pdf
```
