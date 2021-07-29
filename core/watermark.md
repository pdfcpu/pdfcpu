---
layout: default
---

# Watermark

Add a watermark to selected pages of `inFile`. Have a look at some [examples](#examples).

Watermarks may be stacked on top of each other. 
This allows for producing more complex page stamps - a mixture of text, images and foreign PDF page content.
Using `description` you can configure various aspects like position, offset, rotation, scaling and opacity. For text based watermarks you can also configure font name, font size, fill color and render mode.


---
WARNING<br>
A watermark resides in the background of a page. How much of the watermark will be rendered visible on a page depends on the layers on top and the transparency involved. This applies to PDF in general. Eg. scanned PDF files usually consist of bitmap images spanning whole pages and will hide anything in the background including any watermark. For these cases use `pdfcpu stamp` with an opacity < 1 instead to get a similar result.

---


## Usage

```
pdfcpu watermark add    [-p(ages) selectedPages] -m(ode) text|image|pdf -- string|file description inFile [outFile]
pdfcpu watermark update [-p(ages) selectedPages] -m(ode) text|image|pdf -- string|file description inFile [outFile]
pdfcpu watermark remove [-p(ages) selectedPages] inFile [outFile]
```



---
NOTE<br>
In the Adobe world a watermark is text or an image that appears either in front of or behind existing document content, unlike a stamp comment aka stamp annotation that anybody reading the PDF can open, edit, move around and delete. The difference here is that a watermark is integrated into a PDF page as a fixed element. Within `pdfcpu` the meaning of these terms is slightly different:

* `stamp` is any accumulated *content* that appears in front of the existing page content - sitting on top of everything else on a page at a fixed position.

* `watermark` is any accumulated *content* that appears behind the existing page content - residing in the page background at a fixed position.

where *content* may be text, an image or a PDF page.


---
<br>

### Flags

| flag                             | description          | required
|:---------------------------------|:---------------------|:--------
| [p(ages)](../getting_started/page_selection) | selected pages | no
| [m(ode)](../getting_started/common_flags.md)    | text, image or pdf       | yes


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

| name         | description          | required |
|:-------------|:---------------------|:---------|
| string       | display string       | for text stamps
| file         | file name            | for image or pdf stamps
| description  | configuration string | yes
| inFile       | PDF input file       | yes
| outFile      | PDF output file      | no

<br>

### Description

A configuration string to specify watermark parameters.

You may use parameter prefixes as long as the parameter can be identified.
eg. `o: .7` is ambiguous because there is `opacity` and `offset`
but `op: .7` will do the job.

| parameter | description                            | values                                              | default
|:-----------------|:--------------------------------|:----------------------------------------------------|:---------
| fontname                    | a basefont                                        | `{name}` <br> Please refer to `pdfcpu fonts list`                                                                                                                                                            | `font:Helvetica`
| points                      | fontsize in points                                | `{size}` <br> in combination with absolute scaling only                                                                                                                                                      | `points:24`
| rtl                         | right to left userfont                            | `{rtl}` <br> one of: `on|true|t`, `off|false|f`                                                                                                                                                              | `rtl:off`
| position                    | position lower left corner                        | `{position}` <br> one of: `f|full` or anchor `tl|top-left`, `tc|top-center`, `tr|top-right`, `l|left`, `c|center`, `r|right`, `bl|bottom-left`, `bc|bottom-center`, `br|bottom-right`                        | `pos:c`
| offset                      | offset of lower left corner                       | `{dx} {dy}` <br> in user units  <br> eg. `15 20`                                                                                                                                                             | `off:0 0`
| scalefactor                 | scale factor                                      | `{factor} [mode]` <br> 0.0 < `{factor}` <= 1.0, optional `{mode}` one of: `abs|absolute`, `rel|relative`                                                                                                     | `sc:0.5 rel`
| aligntext                   | horizontal text alignment                         | `{alignment}` <br> one of: `l|left`, `c|center`, `r|right`, `j|justified`                                                                                                                                    | `al:c`
| strokecolor                 | stroke color of text (see mode)                   | `{r} {g} {b}|#{RRGGBB}` <br> color intensity 0.0 <= `{r|g|b}` <= 1.0, `#{RRGGBB}` 8bit hex RGB  <br> eg. `1.0 0.0 0.0` (red)                                                                                 | `strokec:0.5 0.5 0.5` (gray)
| fillcolor, color            | fill color of text (see mode)                     | `{r} {g} {b}|#{RRGGBB}` <br> color intensity 0.0 <= `{r|g|b}` <= 1.0, `#{RRGGBB}` 8bit hex RGB  <br> eg. `1.0 0.0 0.0` (red)                                                                                 | `fillc:0.5 0.5 0.5` (gray)
| backgroundcolor, bgcolor    | bounding box background                           | `{r} {g} {b}|#{RRGGBB}` <br> color intensity 0.0 <= `{r|g|b}` <= 1.0, `#{RRGGBB}` 8bit hex RGB  <br> eg. `1.0 0.0 0.0` (red)                                                                                 | (none)
| rotation*                   | rotation angle                                    | `{r}` <br> in degrees, -180.0 <= `{r}` <= 180.0                                                                                                                                                              | `rot:0.0`
| diagonal*                   | render along diagonal                             | `{diagonal}` <br> one of: `1` (lower left to upper right) `2` (upper left to lower right)                                                                                                                    | `d:1`
| opacity                     | opacity of insertation                            | `{opacity}` <br> 0.0 <= `{opacity}` <= 1.0                                                                                                                                                                   | `op:1.0`
| mode, rendermode            | apply fill and/or stroke color                    | `{mode}` <br> one of: `0` (fill), `1` (stroke), `2` (fill & stroke)                                                                                                                                          | `mo:0`
| margins                     | bounding box margins for text (requires bgcolor)  | `{a}|{v} {h}|{t} {h} {b}|{t} {r} {b} {l}` <br> one of: `{a}` (all four sides), `{v} {h}` (vertical, horizontal), `{t} {h} {b}` (top,  horizontal,  bottom), `{t} {r} {b} {l}` (top, right, bottom, left)     | `ma:0`
| border                      | bounding box border for text (requires bccolor)   | `{width} [round] [{color}]` <br> `{width}` > 0 in user units, `round` set round bounding box corners, `{color}` border color                                                                                 | `bo:0`
| url                         | add link annotation for insertation               | `{url}` <br> omit https://                                                                                                                                                                                   | (none)

\* = Only one of rotation and diagonal is allowed.

The following description parameters are for text based watermarks only:

* fontname
* points
* aligntext
* strokecolor
* fillcolor, color
* backgroundcolor, bgcolor
* rendermode
* margins
* border
* rtl (for user fonts only)

<br>

#### Anchors for positioning

|||||
|-|-|-|-|
|       | left | center |right
|top    | `tl` | `tc`   | `tr`
|       | `l`  | `c`    |  `r`
|bottom | `bl` | `bc`   | `br`

or

|||||
|-|-|-|-|
|        | left          | center          | right          |
| top    | `top-left`    | `top-center`    | `top-right`    |
|        | `left`        | `center`        | `right`        |
| bottom | `bottom-left` | `bottom-center` | `bottom-right` |


<br>

#### Default description

```sh
'f:Helvetica, points:24, rtl:off, pos:c, off:0 0, sc:0.5 rel, align:c, strokec:#808080, fillc:#808080, rot:0, d:1, op:1, mo:0, ma:0, bo:0'
```

The default watermark configuration is:

* fixed center page position (for 'free' positioning use pos:bl)
* scale factor `0.5 rel`ative to page dimensions
* positive rotation along the diagonale from the lower left to the upper right page corner (`d:1`).
* fully opaque watermark by defining `o`pacity `1`

In addition for text based watermarks:

* font name `Helvetica`
* font size `24` points
* align: `c`
* stroke color gray (`0.5 0.5 0.5`)
* fill color gray (`0.5 0.5 0.5`)
* no background color and therefore no bounding box
* render mode fill (`mo:0`)
* margins `0`
* border `0`
* rtl `off`

You only have to specify parameters that differ from the default.
<br>

## Examples

### Text Based Watermarks

Create a watermark using defaults only:
```sh
pdfcpu watermark add -mode text -- "This is a watermark" "" in.pdf out.pdf
```
<p align="center">
  <img style="border-color:silver" border="1" src="resources/wmt10.png" height="300">
</p>

<br>
Create a watermark using scale factor 1:

```sh
pdfcpu watermark add -mode text -- "This is a watermark" "sc:1" in.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/wmt1.png" height="300"> &nbsp; &nbsp; &nbsp; &nbsp;
  <img style="border-color:silver" border="1" src="resources/wmt11.png" height="100">
</p>

<br>

Create a watermark along the second diagonale using scale factor 0.9, default render mode `fill` and a fill color:

```sh
pdfcpu watermark add -mode text -- "This is a watermark" "sc:.9, d:2, c:.6 .2 .9" in.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/wmt21.png" height="300">
</p>

<br>

Create a watermark with 0 degree rotation using scale factor 0.9 and render mode `stroke`:

```sh
pdfcpu watermark add -mode text -- "This is a watermark" "sc:.9, rot:0, mo:1" in.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/wmt22.png" height="300">
</p>

<br>

Create a watermark with a counterclockwise rotation of 45 degrees using scale factor 1, render mode `fill & stroke` and a fill color:

```sh
pdfcpu watermark add -mode text -- "This is a watermark" "sc:1, rot:45, mo:2, c:.2 .7 .9" in.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/wmt20.png" height="300">
</p>

<br>

Create a watermark using  some multi line text, show its bounding box by setting bgcol, set all margins to 5 and a border width of 7 rendering round corners.

```sh
pdfcpu watermark add -mode text -- "Some multi\nline text" "ma:5, bo:7 round .3 .7 .7, fillc:#3277d3, bgcol:#beded9, rot:0" in.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/stMarginsRoundBorder.png" height="300">
</p>

<br>

Create a watermark with default rotation, using scale factor 1, font size 48, default render mode `fill`, a fill color and set opacity to 0.6:

```sh
pdfcpu watermark add -mode text -- "Draft" "points:48, scale:1, color:.8 .8 .4, op:.6" in.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/wmt3.png" height="300">
</p>

<br>
Let's assume we have a PDF where even pages are blank. We can add a watermark for theses pages saying "Intentionally left blank" like so:

```sh
pdfcpu watermark add -pages even -mode text -- "Intentionally left blank" "" in.pdf out.pdf
```

We also could have used `pdfcpu stamp`. There is really no difference since we apply only to empty pages here.

<br>

### Image Based Watermarks

Create a watermark using defaults only:
```sh
pdfcpu watermark add -mode image -- "pic.jpg" "" in.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/wmi.png" height="300">
</p>

<br>

Create a watermark using 0 degree rotation and relative scaling of 1.0:

```sh
pdfcpu watermark add -mode image -- "pic.jpg" "sc:1 rel, rot:0" in.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/wmi1rel.png" height="300">
</p>

<br>

Create a watermark using 0 degree rotation and absolute scaling of 1.0:

```sh
pdfcpu watermark add -mode image -- "pic.jpg" "sc:1 abs, rot:0" in.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/wmi1abs.png" height="300">
</p>

<br>

Create a watermark using a clockwise rotation of 30 degrees and absolute scaling of 1.0:

```sh
pdfcpu watermark add -mode image -- "pic.jpg" "rotation:-30, scalefactor:1 abs" in.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/wmi2.png" height="300">
</p>

<br>

Create a watermark using a clockwise rotation of 30 degrees and absolute scaling of 0.25:

```sh
pdfcpu watermark add -mode image -- "pic.jpg" "rot:-30, sc:.25 abs" in.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/wmi4.png" height="300">
</p>

### PDF Based Watermarks

Create a watermark using defaults only. This will apply page 1 of `some.pdf`:

```sh
pdfcpu watermark add -mode pdf -- "some.pdf:1" "" in.pdf out.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/wmp.png" height="300">
</p>

<br>

This is how to create a watermark using defaults and page 2 of `some.pdf`:

```sh
pdfcpu watermark add -mode pdf -- "some.pdf:2" "" in.pdf out.pdf
```
