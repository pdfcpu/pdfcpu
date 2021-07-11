---
layout: default
---

# List Annotations

* This command prints a list of annotations by type for selected pages

Have a look at some [examples](#examples).

## Usage

```
pdfcpu annotations list [-p(ages) selectedPages] inFile
```

<br>

### Flags

| name                             | description     | required
|:---------------------------------|:----------------|---------
| [p(ages)](../getting_started/page_selection) | selected pages | no


<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes

<br>

## Examples

 List all annotations of test.pdf:
```
pdfcpu annot list test.pdf
4 annotations available

Page 3:

  Link:
     obj#                 rect id              content
    ==================================================
      539 (230, 610, 365, 625)    https://www.abc.com/
      538 (271, 637, 326, 693)    https://www.def.com/

Page 4:

  Link:
     obj#                 rect id    content
    ========================================
      563 ( 70, 354, 197, 377)    (internal)
      575 ( 70, 634, 120, 657)    (internal)

