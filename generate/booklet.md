---
layout: default
---

# Booklet

* Arrange a sequence of pages of `inFile` onto larger sheets of paper for a small book or zine and write the result to `outFile`.

* Create your booklet or zine out of a sequence of image files.

* Optionally set the sheet background color and render guidelines for folding and cutting.

* Have a look at some [examples](#examples).

<br>


## Usage

```
pdfcpu booklet [-p(ages) selectedPages] [description] outFile n inFile|imageFiles...
```

<br>

### Flags

| name                                         | description    | required
|:---------------------------------------------|:---------------|---------
| [p(ages)](../getting_started/page_selection) | selected pages | no

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

| name         | description          | required | default
|:-------------|:---------------------|:---------|:-
| description  | configuration string | no
| outFile      | PDF output file      | yes
| n            | the N-up value (2 or 4) | yes
| inFile       | PDF input file       | inFile or imageFile(s)
| imageFile... | one or more images   | inFile or imageFile(s)

<br>

### N-up Value

There are two styles of booklet, depending on your page/input and sheet/output size:

#### n=2

Two of your pages fit on one side of a sheet (eg statement on letter, A5 on A4)
Assemble by printing on both sides (odd pages on the front and even pages on the back) and folding down the middle.

#### n=4

Four of your pages fit on one side of a sheet (eg statement on ledger, A5 on A3, A6 on A4)
Assemble by printing on both sides, then cutting the sheets horizontally.
The sets of pages on the bottom of the sheet are rotated so that the cut side of the
paper is on the bottom of the booklet for every page. After cutting, place the bottom
set of pages after the top set of pages in the booklet. Then fold the half sheets.

| value | portrait | landscape
|:------|:---------|----------
| 2     | 1x2      | 2x1
| 4     | 2x2      | 2x2

<br>

### Description

A configuration string to specify the details of the grid layout.

| parameter            | values                                      | default
|:---------------------|:--------------------------------------------|:--
| dimensions           | (width, height) in user units eg. '400 200' | d: 595 842
| formsize, paper size | [paper size](../paper.md) to be used. Append L or P to enforce landscape/portrait mode| f: A4
| guides               | on/off true/false                           | g:off
| orientation          | one of `rd, dr, ld, dl` for PDF input files | o: rd
| border               | on/off true/false                           | b: on
| margin               | integer >= 0                                | m: 0
| backgroundcolor, bgcol | 0.0 <= r,g,b <= 1.0, eg. 1.0, 0.0 0.0 = red | none
|                      | or the hex RGB value: #RRGGBB               |

<br>

#### Default description

```sh
'f:A4, d:595 842, bo:on, g:off, m:3'
```

* You only have to specify any parameter diverging from the default.

* Only one of dimensions or format is allowed.

* You may use parameter prefixes as long as the parameter can be identified.


## Examples

Create `out.pdf` by applying 4-up to `in.pdf`. Each page fits `4` original pages of `in.pdf` into a 2x2 grid:
```sh
pdfcpu nup out.pdf 4 in.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/nup4pdf.png" height="300">
</p>

<br>

The output file will use the page size of the input file unless explicitly declared by a description string like so:
```sh
pdfcpu nup 'f:A4' out.pdf 9 in.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/nup9pdf.png" height="300">
</p>

<br>

`nup` also accepts a list of image files with the result of rendering all images
in N-up fashion into a PDF file using the specified paper size (default=A4).
Generate `out.pdf` using `A4 L`andscape mode where each page fits 4 images onto a 2x2 grid.
The grid element border lines are rendered by default as well is the default margin of 3 points applied:

```sh
pdfcpu nup 'f:A4L' out.pdf 4 *.jpg *.png *.tif
````


<p align="center">
  <img style="border-color:silver" border="1" src="resources/nup4img.png">
</p>

<br>

A single image input file supplied will produce a single page PDF ouput file.<br>
In the following example `logo.jpg` will be `16`-up'ed onto `out.pdf`.
Both grid borders and margins are suppressed and the output format is `Ledger`:

```sh
pdfcpu nup 'f:Ledger, b:off, m:0' out.pdf 16 logo.jpg
```


<p align="center">
  <img style="border-color:silver" border="1" src="resources/nup16img.png">
</p>