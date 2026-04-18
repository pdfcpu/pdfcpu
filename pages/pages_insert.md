---
layout: default
---

# Insert Pages

This command inserts empty pages:

- Before or after selected pages OR
- If no page is selected, before or after *every page*

Have a look at some [examples](#examples).

## Usage

```
pdfcpu pages insert [ description ] inFile [ outFile ] [flags]
```

<br>

### Flags

| name                                         | description    | required | values | default
|:---------------------------------------------|:---------------|----------|--------|--------
| [p(ages)](../getting_started/page_selection) | selected pages | no
| [m(ode)]()                                   |                | no       | before, after | before


<br>

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
| outFile...   | PDF output file     | no

<br>

## Examples

Insert an empty page before every page of `in.pdf`. This way you get a PDF that gives you space for adding annotations for pages:

```sh
$ pdfcpu pages insert in.pdf
writing in.pdf...
```

<br>

Insert an empty page before pages 1 to 5:

```sh
$ pdfcpu pages insert in.pdf out.pdf --pages 1-5
writing out.pdf...
```

<br>

Insert an empty page after the last page:

```sh
$ pdfcpu pages insert in.pdf out.pdf --pages l --mode after 
writing out.pdf...
```
