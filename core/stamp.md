---
layout: default
---

# Stamp

Add stamps to selected pages of `inFile`. Have a look at some [examples](#examples).

Stamps may be stacked on top of each other. 
This allows for producing more complex page stamps - a mixture of text, images and foreign PDF page content.
Using `description` you can configure various aspects like position, offset, rotation, scaling and opacity. For text based stamps you can also configure font name, font size, fill color and render mode.

## Usage

```
pdfcpu stamp add    [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] -mode text|image|pdf string|file description inFile [outFile]
pdfcpu stamp remove [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] inFile [outFile]
pdfcpu stamp update [-v(erbose)|vv] [-q(uiet)] [-pages selectedPages] [-upw userpw] [-opw ownerpw] -mode text|image|pdf string|file description inFile [outFile]
```

<br>

---
NOTE

In the Adobe world a watermark is text or an image that appears either in front of or behind existing document content, like a stamp comment aka stamp annotation that anybody reading the PDF can open, edit, move around and delete. The difference here is that a watermark is integrated into a PDF page as a fixed element. Within `pdfcpu` the meaning of these terms is slightly different:

* `stamp` is any accumulated *content* that appears in front of the existing page content - sitting on top of everything else on a page

* `watermark` is any accumulated *content* that appears behind the existing page content - residing in the page background

where *content* may be text, an image or a PDF page.

---
<br>

### Flags

| flag                             | description          | required
|:---------------------------------|:---------------------|:--------
| [verbose](../getting_started/common_flags.md) | turn on logging      | no
| [vv](../getting_started/common_flags.md)      | verbose logging      | no
| [quiet](../getting_started/common_flags.md)   | quiet mode      | no
| [pages](../getting_started/page_selection) | page selection  | no
| [upw](../getting_started/common_flags.md)     | user password        | no
| [opw](../getting_started/common_flags.md)    | owner password       | no
| [mode](../getting_started/common_flags.md)    | text, image or pdf       | yes


<br>

### Arguments

| name         | description          | required | 
|:-------------|:---------------------|:---------|
| string       | display string       | for text stamps
| file         | file name            | for image or pdf stamps
| description  | configuration string | yes
| inFile       | PDF input file       | yes
| outFile      | PDF output file      | no

<br>

### Description

A configuration string to specify the stamp parameters.

You may use parameter prefixes as long as the parameter can be identified.
eg. `o: .7` is ambiguous because there is `opacity` and `offset`
but `op: .7` will do the job.

| parameter | description                            | values                                              | default
|:-----------------|:--------------------------------|:----------------------------------------------------|:---------
| fontname         | a basefont                      | Please refer to `pdfcpu fonts list`                 | font: Helvetica
| points           | fontsize in points              | in combination with absolute scaling only           | points: 24
| position         | the stamps lower left corner    | one of `full` or the anchors: `tl, tc, tr, l, c, r, bl, bc, br`| pos: c
| offset           |                                 | (dx,dy) in user units eg. '15 20'                   | off: 0 0
| scalefactor      |                                 | 0.0 < i <= 1.0 followed by optional `abs` or `rel`  | s: 0.5 rel
| color            | 3 fill color intensities        | 0.0 <= r,g,b <= 1.0, eg. 1.0, 0.0 0.0 = red         | c: 0.5 0.5 0.5 = gray
| rotation         | rotation angle                  | -180.0 <= i <= 180.0                                | rot: 0.0
| diagonal         | render along diagonal           | 1 .. lower left to upper right                      | d:1
|                  |                                 | 2 .. upper left to lower right                      |
| opacity          |                                 | 0.0 <= i <= 1.0                                     | op:1
| mode, rendermode |                                 | 0 .. fill                                           | m:0
|                  |                                 | 1 .. stroke                                         |
|                  |                                 | 2 .. fill & stroke                                  |

Only one of rotation and diagonal is allowed.

The following description parameters are for text based stamps only:

* font name
* font size
* color
* render mode

<br>

#### Anchors for positioning

|||||
|-|-|-|-|
|       | left | center |right
|top    | `tl` | `tc`   | `tr`
|       | `l`  | `c`    |  `r`
|bottom | `bl` | `bc`   | `br`

<br>

#### Default description

```sh
'f:Helvetica, points:24, s:0.5 rel, pos:c, off: 0 0, c:0.5 0.5 0.5, rot:0, d:1, op:1, m:0'
```
The default stamp configuration is:

* fixed center page position (for 'free' positioning use pos:bl)
* scale factor `0.5 rel`ative to page dimensions
* positive rotation along the diagonale from the lower left to the upper right page corner (`d:1`).
* fully opaque stamp by defining `op`acity `1`

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
pdfcpu stamp add -mode text 'This is a stamp' '' test.pdf out.pdf
```
<p align="center">
  <img style="border-color:silver" border="1" src="resources/stt10.png" height="300">
</p>

<br>
Create a stamp using scale factor 1:

```sh
pdfcpu stamp add 'This is a stamp, s:1' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/stt1.png" height="300"> &nbsp; &nbsp; &nbsp; &nbsp;
  <img style="border-color:silver" border="1" src="resources/stt11.png" height="100">
</p>

<br>

Create a stamp along the second diagonale using scale factor 0.9, default render mode `fill` and a fill color:

```sh
pdfcpu stamp add -mode text 'This is a stamp' 's:.9, d:2, c:.6 .2 .9' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/stt41.png" height="300">
</p>

<br>

Create a stamp with 0 degree rotation using scale factor 0.9 and render mode `stroke`:

```sh
pdfcpu stamp add -mode text 'This is a stamp' 's:.9, rot:0, m:1' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/stt42.png" height="300">
</p>

<br>

Create a stamp with a counterclockwise rotation of 45 degrees using scale factor 1, render mode `fill & stroke` and a fill color:

```sh
pdfcpu stamp add -mode text 'This is a stamp' 'scale:1, rot:45, mode:2, color:.2 .7 .9' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/stt43.png" height="300">
</p>

<br>

Create a stamp with default rotation, using scale factor 1, font size 48, default render mode `fill`, a fill color and increasing opacity from 0.3 to 1. By setting an opacity < 1 you can fake a watermark. This may be useful in scenarios where `pdfcpu watermark` does not produce satisfying results for a particular PDF file:

```sh
pdfcpu stamp add -mode text 'Draft' 'points:48, s:1, c:.8 .8 .4, op:.3' test.pdf out1.pdf
pdfcpu stamp add -mode text 'Draft' 'points:48, s:1, c:.8 .8 .4, op:0.6' test.pdf out2.pdf
pdfcpu stamp add -mode text 'Draft' 'points:48, s:1, c:.8 .8 .4, op:1' test.pdf out3.pdf
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
pdfcpu stamp add -mode image 'pic.jpg' '' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/sti.png" height="300">
</p>

<br>

Create a stamp using 0 degree rotation and relative scaling of 1.0:

```sh
pdfcpu stamp add -mode image 'pic.jpg' 'scalef:1 rel, rot:0' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/sti1.png" height="300">
</p>

<br>

### PDF Based Stamps

Create a stamp using defaults only. This will apply page 1 of `some.pdf`:

```sh
pdfcpu stamp add -mode pdf 'some.pdf' '' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/stp.png" height="300">
</p>

<br>

Create a stamp using defaults and page 2 of `some.pdf`, apply a 0 degree rotation and 0.3 relative scaling:

```sh
pdfcpu stamp add -mode pdf 'some.pdf:2' 'rot:0, scalef:.3' test.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/stp3.png" height="300">
</p>

<br>

### Stamp Lifecycle

Create a stamp using the default options.
```sh
pdfcpu stamp add -mode text 'Draft' '' template.pdf work.pdf
```
<p align="center">
  <img style="border-color:silver" border="1" src="resources/1exp.png" height="300">
</p>

<br>

Let's edit the stamps color, render mode and opacity
```sh
pdfcpu stamp update -mode text 'Draft' 'c: .2 .6 .5, m:2, op:.7' work.pdf
```
<p align="center">
  <img style="border-color:silver" border="1" src="resources/2exp.png" height="300">
</p>

<br>

We add a centered footer on the bottom of the page.
```sh
pdfcpu stamp add -mode text 'Footer' 'pos:bc, scale: 1.0 abs, rot:0, c: .5 .5 .9' work.pdf
```
<p align="center">
  <img style="border-color:silver" border="1" src="resources/3exp.png" height="300">
</p>

<br>

Let's add a logo in the top right corner.
```sh
pdfcpu stamp add -mode image 'logo.png' 'pos:tr, rot:0, s:.2' work.pdf
```
<p align="center">
  <img style="border-color:silver" border="1" src="resources/4exp.png" height="300">
</p>

<br>

Let's get rid of the stamp on page 1
```
pdfcpu stamp remove -pages 1 work.pdf
```
<p align="center">
  <img style="border-color:silver" border="1" src="resources/t.png" height="300">
</p>

Finally let's remove all stamps of this file.
```
pdfcpu stamp remove work.pdf
``` 
