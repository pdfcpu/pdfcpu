---
layout: default
title: "Add Properties"
---

# Add Properties

This command adds property name/value pairs to a PDF document. Have a look at some [examples](#examples).

You can also set the PDFs *Title*, *Subject* and *Author*. 

## Usage

```
pdfcpu properties add inFile nameValuePair... [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| nameValuePair | 'name = value' | yes

<br>

## Examples

Adding a property:

```sh
$ pdfcpu properties add in.pdf name = value
```

```sh
$ pdfcpu properties add in.pdf 'name = value'
```

Adding two properties:
```sh
$ pdfcpu properties add in.pdf 'name1 = value1' 'name2 = value2'
```

Setting Title and Author:
```sh
$ pdfcpu properties add in.pdf 'Title = My title' 'Author = Me'
```

<br>

Add properties while reading and writing the PDF through pipes:

```sh
$ aws s3 cp s3://acme-assets/brochure.pdf - \
   | pdfcpu properties add - - 'Subject = Product Launch' \
   | aws s3 cp - s3://acme-assets/brochure-described.pdf
```
