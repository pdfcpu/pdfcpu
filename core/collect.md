---
layout: default
---

# Collect

* Create a custom PDF page sequence.

* Arrange your pages in any order you like.

* Pages may appear multiple times.

* Have a look at some [examples](#examples).


## Usage

```
pdfcpu collect -p(ages) selectedPages inFile [outFile]
```

<br>

### Flags

| name                                         | description    | required
|:---------------------------------------------|:---------------|---------
| [p(ages)](../getting_started/page_selection) | selected pages | yes

<br>

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name         | description         | required | default
|:-------------|:--------------------|:---------|:-
| inFile       | PDF input file      | yes
| outFile      | PDF output file     | no       | inFile

<br>

## Examples

Create a custom page collection from `in.pdf` and write the result to `out.pdf`.
Begin with 3 instances of page 1 then append the rest of the file excluding the last page:

```sh
$ pdfcpu collect -pages 1,1,1,2-l-1 in.pdf out.pdf
writing sequ.pdf ...
```
