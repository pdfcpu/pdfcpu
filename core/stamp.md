---
layout: default
---

# Stamp

Add a stamp to selected pages of `inFile`. Have a look at some [examples](#examples).

The stamp is centered on the page and using `description` you can configure various aspects like rotation, scaling and opacity. For text based stamps you can also configure font name, font size, fill color and render mode.

## Usage

```
pdfcpu stamp [-v(erbose)|vv] [-pages pageSelection] [-upw userpw] [-opw ownerpw] description inFile [outFile]
```

You can stamp or watermark `inFile` exactly once. It is highly recommended to make a backup of `inFile` before running this command or even better use `outFile`.
<br>

---
NOTE

In the Adobe world a watermark is text or an image that appears either in front of or behind existing document content, like a stamp comment aka stamp annotation that anybody reading the PDF can open, edit, move around and delete. The difference here is that a watermark is integrated into a PDF page as a fixed element. Within `pdfcpu` the meaning of these terms is slightly different:

* `stamp` is any *content* that appears in front of the existing page content - sitting on top of everything else on a page

* `watermark` is any *content* that appears behind the existing page content - residing in the page background

where *content* may be text, an image or a PDF page.

---
<br>

### Flags

| flag                             | description          | required
|:---------------------------------|:---------------------|:--------
| [verbose](../getting_started.md) | turn on logging      | no
| [vv](../getting_started.md)      | verbose logging      | no
| [pages](../getting_started/page_selection) | page selection  | no
| [upw](../getting_started.md)     | user password        | no
| [opw](../getting_started.md)     | owner password       | no

<br>

### Arguments

| name         | description          | required | default
|:-------------|:---------------------|:---------|:-
| description  | configuration string | yes
| inFile       | PDF input file       | yes
| outFile      | PDF output file      | no       | inFile_new.pdf

<br>

### Description

A configuration string to specify the stamp parameters.

The first entry of the description configures the type. It is one of the following:

* text string
* image file name
* PDF file name followed by an optional page number

| parameter | description                     | values                                              | default
|:----------|:--------------------------------|:----------------------------------------------------|:-
| f         | fontname, a basefont            | Helvetica, Times-Roman, Courier                     | Helvetica
| p         | fontsize in points              | in combination with absolute scaling only           | 24
| s         | scale factor                    | 0.0 < i <= 1.0 followed by optional `abs` or `rel`  | 0.5 rel
| c         | color, 3 fill color intensities | 0.0 <= r,g,b <= 1.0, eg. 1.0, 0.0 0.0 = red         | 0.5 0.5 0.5 = gray
| r         | rotation angle                  | -180.0 <= i <= 180.0                                | 0.0
| d         | render along diagonal           | 1 .. lower left to upper right                      | 1
|           |                                 | 2 .. upper left to lower right                      |
| o         | opacity                         | 0.0 <= i <= 1.0                                     | 1
| m         | render mode                     | 0 .. fill                                           | 0
|           |                                 | 1 .. stroke                                         |
|           |                                 | 2 .. fill & stroke                                  |

Only one of rotation and diagonal is allowed.

The following description parameters are for text based stamps only:

* font name
* font size
* color
* render mode

<br>

#### Default description

```sh
'f:Helvetica, p:24, s:0.5 rel, c:0.5 0.5 0.5, r:0, d:1, o:1, m:0'
```

The default stamp configuration is:

* fixed center page position (free positioning will be part of a future release)
* scale factor `0.5 rel`ative to page dimensions
* positive rotation along the diagonale from the lower left to the upper right page corner (`d:1`).
* fully opaque stamp by defining `o`pacity `1`

In addition for text based stamps:

* font name `Helvetica`
* font size `24` points
* fill color grey (`0.5 0.5 0.5`)
* render mode fill (`m:0`)

You only have to specify parameters that differ from the default.

<br>

## Examples

### Text Based Stamps

Create a stamp using defaults only:
```sh
pdfcpu stamp 'This is a stamp' test.pdf out.pdf
```
<p align="center">
  <img style="border-color:silver" border="1" src="resources/stt10.png" height="300">
</p>

<br>
Create a stamp using scale factor 1:

```sh
pdfcpu stamp 'This is a stamp, s:1' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/stt1.png" height="300"> &nbsp; &nbsp; &nbsp; &nbsp;
  <img style="border-color:silver" border="1" src="resources/stt11.png" height="100">
</p>

<br>

Create a stamp along the second diagonale using scale factor 0.9, default render mode `fill` and a fill color:

```sh
pdfcpu stamp 'This is a stamp, s:.9, d:2, c:.6 .2 .9' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/stt41.png" height="300">
</p>

<br>

Create a stamp with 0 degree rotation using scale factor 0.9 and render mode `stroke`:

```sh
pdfcpu stamp 'This is a stamp, s:.9, r:0, m:1' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/stt42.png" height="300">
</p>

<br>

Create a stamp with a counterclockwise rotation of 45 degrees using scale factor 1, render mode `fill & stroke` and a fill color:

```sh
pdfcpu stamp 'This is a stamp, s:1, r:45, m:2, c:.2 .7 .9' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/stt43.png" height="300">
</p>

<br>

Create a stamp with default rotation, using scale factor 1, font size 48, default render mode `fill`, a fill color and increasing opacity from 0.3 to 1. By setting an opacity < 1 you can fake a watermark. This may be useful in scenarios where `pdfcpu watermark` does not produce satisfying results for a particular PDF file:

```sh
pdfcpu stamp 'Draft, p:48, s:1, c:.8 .8 .4, o:.3' test.pdf out1.pdf
pdfcpu stamp 'Draft, p:48, s:1, c:.8 .8 .4, o:0.6' test.pdf out2.pdf
pdfcpu stamp 'Draft, p:48, s:1, c:.8 .8 .4, o:1' test.pdf out3.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/stt33.png" height="200"> &nbsp;
  <img style="border-color:silver" border="1" src="resources/stt32.png" height="200"> &nbsp;
  <img style="border-color:silver" border="1" src="resources/stt31.png" height="200">
</p>

<br>

### Image Based Stamps

Create a stamp using defaults only:
```sh
pdfcpu stamp 'pic.jpg' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/sti.png" height="300">
</p>

<br>

Create a stamp using 0 degree rotation and relative scaling of 1.0:

```sh
pdfcpu stamp 'pic.jpg, s:1 rel, r:0' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/sti1.png" height="300">
</p>

<br>

### PDF Based Stamps

Create a stamp using defaults only. This will apply page 1 of `some.pdf`:

```sh
pdfcpu stamp 'some.pdf' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/stp.png" height="300">
</p>

<br>

Create a stamp using defaults and page 2 of `some.pdf`, apply a 0 degree rotation and 0.3 relative scaling:

```sh
pdfcpu stamp 'some.pdf:2, r:0, s:.3' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/stp3.png" height="300">
</p>
