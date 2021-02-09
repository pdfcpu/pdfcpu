---
layout: default
---

# Grid

* Rearrange the pages of a PDF file into grid pages for enhanced reading experience.

* Arrange image files into grid pages for enhanced browsing experience.

* Have a look at some [examples](#examples).


## Usage

```
pdfcpu grid [-p(ages) selectedPages] [description] outFile m n inFile|imageFiles...
```

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

| name         | description          | required
|:-------------|:---------------------|:--------
| description  | configuration string | no
| outFile      | PDF output file      | yes
| m            | horizontal span      | yes
| n            | vertical span        | yes
| inFile       | PDF input file       | inFile or imageFile(s)
| imageFile... | one or more images   | inFile or imageFile(s)

<br>

### Description

A configuration string to specify the details of the grid layout.

| parameter            | values                                      | default
|:---------------------|:--------------------------------------------|:--
| dimensions           | (width, height) in user units eg. '400 200' | d: 595 842
| formsize, paper size | [paper size](../paper.md) to be used. Append L or P to enforce landscape/portrait mode| f: A4
| orientation          | one of `rd, dr, ld, dl` for PDF input files | o: rd
| border               | on/off true/false                           | b: on
| margin               | integer >= 0                                | m: 0

<br>

#### Orientation

For PDF input files only.<br>
This is usually associated with the writing direction used in the document to be processed.

| value | description |
|:------|-------------|
| rd    | right down, default |
| dr    | down right  |
| ld    | left down   |
| dl    | down left   |

<br>

#### Default description

```sh
'f:A4, d:595 842, o:rd, b:on, m:3'
```

* You only have to specify any parameter diverging from the default.

* Only one of dimensions or format is allowed.

* You may use parameter prefixes as long as the parameter can be identified.

<br>

## Examples

The page size of the output file is a grid of specified dimensions in original page units. Output pages may be big but that's ok since they are not supposed to be printed.

One use case mentioned by the community was to produce PDF files for source code listings eg. in the form of 1x10 grid pages.<br>
In the following example we use a 1x4 grid since this is easier to visualize.

Rearrange pages of in.pdf into pages composed of 1x4 grids and write the result to out.pdf using the default orientation. The output page size is the result of a 1(horizontal) x 4(vertical) grid using in.pdf's page size:

```sh
pdfcpu grid 'b:off' out.pdf 1 4 in.pdf
```


<p align="center">
  <img style="border-color:silver" border="1" src="resources/gridpdf.png" height="300">
</p>

<br>
When applied to image files this command produces simple photo galleries of arbitrary dimensions in PDF form.<br>
Arrange imagefiles onto a 5x2 page grid and write the result to out.pdf using a grid cell size of 500x500:

```sh
pdfcpu grid 'd:500 500, m:20, b:off' out.pdf 5 2 *.jpg
```


<p align="center">
  <img style="border-color:silver" border="1" src="resources/gridimg.png">
</p>
<br>


Rearrange pages of in.pdf into 2x2 grids and write result to out.pdf using the default orientation.
The output page size is the result of a 2(hor)x2(vert) page grid using page size Legal in landscape mode:

```sh
pdfcpu grid 'LegalL' out.pdf 2 2 in.pdf
```

<br>
Rearrange pages of in.pdf into 3x2 grids and write result to out.pdf using orientation 'right down'.
The output page size is the result of a 3(hor)x2(vert) page grid using in.pdf's page size:

```sh
pdfcpu grid 'o:rd' out.pdf 3 2 in.pdf
```