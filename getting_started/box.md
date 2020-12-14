---
layout: default
---

# Box Description

Used by the commands:

* [boxes add](../boxes/boxes_add.md)
* [crop](../core/crop.md)

A box is a rectangular region in user space describing one of PDF's page boundaries:

      media box:  boundaries of the physical medium on which the page is to be printed.
       crop box:  region to which the contents of the page shall be clipped (cropped) when displayed or printed.
      bleed box:  region to which the contents of the page shall be clipped when output in a production environment.
       trim box:  intended dimensions of the finished page after trimming.
        art box:  extent of the page’s meaningful content as intended by the page’s creator.

   Please refer to the PDF Specification 14.11.2 Page Boundaries for details.

   All values are in display units (po, in, mm, cm)

General rules:

    The media box is mandatory and serves as default for the crop box and is its parent box.

    The crop box serves as default for art box, bleed box and trim box and is their parent box.

<br>

## Arbitrary rectangular region in user space

| config string | description
|:-----------|:-----------
|'[0 10 200 150]'  | lower left corner at (0/10) and upper right corner at (200/150) or xmin:0 ymin:10 xmax:200 ymax:150

<br>

## Via margins within parent box

| config string | description
|:-----------|:-----------
|'0.5 0.5 20 20'     | absolute, top:.5 right:.5 bottom:20 left:20
|'0.5 0.5 .1 .1 abs' | absolute, top:.5 right:.5 bottom:.1 left:.1
|'0.5 0.5 .1 .1 rel' | relative, top:.5 right:.5 bottom:20 left:20
|'10'   |              absolute, top,right,bottom,left:10
|'10 5'  |             absolute, top,bottom:10  left,right:5
|'10 5 15'  |            absolute, top:10 left,right:5 bottom:15
| '5%'  |                 relative, top,right,bottom,left:5% of parent box width/height
| '.1 .5'  |              absolute, top,bottom:.1  left,right:.5
| '.1 .3 rel'  |         relative, top,bottom:.1=10%  left,right:.3=30%
|'-10' |                absolute, top,right,bottom,left:-10 relative to parent box (for crop box the media box gets expanded)

<br>

## Anchored within parent box

Use dim and optionally pos, off. The default position is: center.

| config string | description
|:-----------|:-----------
| 'dim: 200 300 abs' | centered, 200x300 display units
|'pos:c, off:0 0, dim: 200 300 abs' |  centered, 200x300 display units
|'pos:tl, off:5 5, dim: 50% 50% rel' | anchored to top left corner, 50% width/height of parent box, offset by 5/5 display units
|'pos:br, off:-5 -5, dim: .5 .5 rel' | anchored to bottom right corner, 50% width/height of parent box, offset by -5/-5 display units

<br>

### Anchors for positioning
|||||
|-|-|-|-|
|       | left | center |right
|top    | `tl` | `tc`   | `tr`
|       | `l`  | `c`    |  `r`
|bottom | `bl` | `bc`   | `br`