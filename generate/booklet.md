---
layout: default
---

# Booklet

is a useful variation of the N-up command.

- Arrange a sequence of pages of `inFile` onto larger sheets of paper for a small book or zine and write the result to `outFile`.

- Create your booklet or zine out of a sequence of image files.

- Optionally set the sheet background color and render guidelines for folding and cutting.

- Have a look at some [examples](#examples).

<br>

## Usage

```
pdfcpu booklet [-p(ages) selectedPages] -- [description] outFile n inFile|imageFiles...
```

<br>

### Flags

| name                                         | description    | required |
| :------------------------------------------- | :------------- | -------- |
| [p(ages)](../getting_started/page_selection) | selected pages | no       |

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [-o(ffline)](../getting_started/common_flags.md)| disable http traffic |                                 | 
| [c(onf)](../getting_started/common_flags.md)    | config dir      | $path, disable
| [opw](../getting_started/common_flags.md)       | owner password  |
| [upw](../getting_started/common_flags.md)       | user password   |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm

<br>

### Arguments

| name         | description          | required               | values     |
| :----------- | :------------------- | :--------------------- | :--------- |
| description  | configuration string | no                     |
| outFile      | PDF output file      | yes                    |
| n            | the N-up value       | yes                    | 2, 4, 6, 8 |
| inFile       | PDF input file       | inFile or imageFile(s) |
| imageFile... | one or more images   | inFile or imageFile(s) |

<br>

There are many styles of booklet you can choose from, depending on the page size and orientation of your booklet or zine,
the sheet size of the paper you will be printing on, and the method you will be using for assembling.

### N-up Value

The n-up value will be how many pages of your booklet will fit on one side of the sheet you will be printing on. The following options are available:

| n-up | portrait            | landscape            |
| :--- | :------------------ | -------------------- |
| 2    | 1x2, long-edge only | 2x1, long-edge only  |
| 4    | 2x2                 | 2x2                  |
| 6    | 2x3, long-edge only | ---                  |
| 8    | 2x4 (all long-edge) | 4x2 (all short-edge) |

#### n=2

This is the simplest case and the most common for those printing at home.
Two of your pages fit on one side of a sheet (eg statement on letter, A5 on A4).
Assemble by printing on both sides (odd pages on the front and even pages on the back), folding down the middle, and binding the booklet with staples, sewing, etc.

#### n=4

Four of your pages fit on one side of a sheet (eg statement on ledger, A5 on A3, A6 on A4).
When printing 4-up, your booklet can be bound either along the long-edge (for portrait this is the left side of the paper, for landscape the top) or the short-edge (for portrait this is the top of the paper, for landscape the left side).
Using a different binding will change the ordering of the pages on the sheet.
You can set long or short-edge with the `binding` option.

In 4-up printing, the sets of pages on the bottom of the sheet are rotated so that the cut side of the
paper is on the bottom of the booklet for every page (for the default portrait, long-edge binding case.
Similar rotation logic applies for the other three orientations).
Having the cut edge always on bottom makes for more uniform pages within the book and less work in trimming.

To assemble booklets with the default 4-up binding method (`btype=booklet`):

- print on both sides
- cut the sheets in half
- arrange the stacks of half sheets for collation in the following order: top half sheet 1, bottom half sheet 1, top half sheet 2, ...
- collate the stacks into individual sets of booklets
- fold, bind, and trim (if desired)

##### Advanced 4-up

The `btype=advanced` is a special method for assembling, only for 4-up booklets.
Printers that are used to collating first and then cutting may prefer this method.
To assemble:

- print on both sides
- collate the whole sheets
- cut each of the collated sets in half horizontally and place the bottom half (un-rotated) under the top half. Alternately, you can fold each set horizontally and again vertically, then trim the bottom fold off. Both of these methods will produce a correctly ordered booklet.
- bind and trim (if desired)

#### n=6

Six of your pages fit on one side of a sheet. This produces an unusual sized booklet, but can be an efficient use of paper. To assemble:

- print on both sides
- cut the sheets in thirds horizontally
- arrange the sheet stacks for collation: moving top to bottom, then by sheet (ie top third sheet 1, middle third sheet 1, bottom third sheet 1, top third sheet 2, ...)
- collate the stacks into individual sets of booklets
- fold, bind, and trim (if desired)

_Only available for portrait, long-edge orientation._

#### n=8

Eight of your pages fit on one side of a sheet (eg A6 on A3).

For long-edge binding, pages are arranged similar to 4-up with btype=booklet, but with an additonal rotation to fit the pages on the sheet. For short-edge binding, pages are arranged similar to 6-up (left-to-right, top-to-bottom order). To assemble:

- print on both sides
- For long-edge binding: cut the sheets in half horizontally and then cutting those half-sheets in half vertically. For short-edge binding: cut the sheets in quarters horizontally.
- arrange the sheet stacks for collation: moving left to right, then top to bottom, then by sheet (ie top-left sheet 1, top-right sheet 1, middle-left sheet 1, ...)
- collate the stacks into individual sets of booklets
- fold, bind, and trim (if desired)

<br>

### Perfect binding

Perfect binding is a special type of booklet. The main difference is that the binding is glued into the book's spine,
meaning that all pages are cut from the sheet and not folded as in the other forms of booklet.
This results in a different page ordering on the sheet than the other methods. If you intend to perfect bind your booklet,
use `btype:perfectbound`. To assemble:

- print on both sides
- cut the sheets in half horizontally and then cutting those half-sheets in half vertically
- arrange the sheet stacks for collation: moving left to right, then top to bottom, then by sheet
- collate the stacks into individual sets of booklets
- perfect bind and trim (if desired)

### Multifolio or Signatures

Multifolio or signatures, is a technique used to produce longer booklets.
This technique makes the most sense when your book has at least 128 pages.
For example, you can bind your paper in eight sheet folios (also known as signatures), with each folio containing 32 pages of your book.
For such a multi folio booklet set `multifolio:on` and play around with `foliosize` which defaults to 8.

### Description

A configuration string to specify the details of the booklet layout.

| parameter              | values                                                                                  | default |
| :--------------------- | :-------------------------------------------------------------------------------------- | :------ |
| dimensions             | (width, height) in user units eg. '400 200'                                             | 595 842 |
| formsize, paper size   | [paper size](../paper.md) to be used. Append L or P to enforce landscape/portrait mode  | A4      |
| btype                  | The method for arranging pages into a booklet. (booklet, bookletadvanced, perfectbound) | booklet |
| binding                | The edge of the paper which has the binding. (long, short)                              | long    |
| multifolio             | on/off true/false, for PDF input only                                                   | off     |
| foliosize              | for multi folio booklets only                                                           | 8       |
| guides                 | on/off true/false                                                                       | off     |
| border                 | on/off true/false                                                                       | off     |
| margin                 | float >= 0                                                                              | 0       |
| backgroundcolor, bgcol | [color](../getting_started/color.md)                                                    | none    |

<br>

#### Default description

```sh
'formsize:A4, dimensions:595 842, guides:off, border:off, margin:0'
```

- You only have to specify any parameter diverging from the default.

- Only one of dimensions or formsize is allowed.

- You may use parameter prefixes as long as the parameter can be identified.

## Examples

### 2-up booklet

Let's make a booklet where two of your pages fit on one side of a sheet of paper.
We'll be using A4 so the format of the booklet pages will be A5.
This command generates a PDF file representing a sequence of page pairs (front and back side of a sheet of paper).
Once generated we need to print the file two-sided and then assemble our booklet by stacking the printed sheets and folding them down the middle:

```sh
$ pdfcpu booklet -- "p:A4, border:on" booklet.pdf 2 pageSequence.pdf
```

Here is the front and back side of the first printed sheet of paper for an input file with eight pages.
<br>
This also explains that four booklet pages fit on one sheet of paper:

<p align="center">
  <img style="border-color:silver" border="1" src="resources/book2A4p1r.png" height="300">
  <img style="border-color:silver" border="1" src="resources/book2A4p2r.png" height="300">
</p>

<br>

You can also set margins, the sheet background color and you can even render the guidelines for folding:

```sh
$ pdfcpu booklet -- "formsize:A4, border:off, guide:on, margin:10, bgcol:#beded9" booklet.pdf 2 pageSequence.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/book2A4p1.png" height="300">
  <img style="border-color:silver" border="1" src="resources/book2A4p2.png" height="300">
</p>

<br>

Since A4 happens to be the default form size the following command is identical to the one above:

```sh
$ pdfcpu booklet -- "g:on, ma:10, bgcol:#beded9" booklet.pdf 2 pageSequence.pdf
```

<br>

### 4-up booklet

Now let's make a little zine also using A4 but now four zine pages shall fit on one sheet of A4 which results
in a zine page size of A6.

The assembly stage looks a little bit different here since we will also need to do some cutting.
After pdfcpu has generated the PDF which holds our zine first we need to print it using two-sided printing.
Then we take the printed stack and cut the sheets horizontally. After cutting, we place the bottom
set of pages after the top set of pages in the zine. Then fold the half sheets:

```sh
$ pdfcpu booklet -- "p:A4, border:on" zine.pdf 4 pageSequence.pdf
```

Here are the first two pages representing the front and back side of the first paper sheet:

<p align="center">
  <img style="border-color:silver" border="1" src="resources/book4A4p1r.png" height="300">
  <img style="border-color:silver" border="1" src="resources/book4A4p2r.png" height="300">
</p>

<br>

Using guidelines for cutting and folding and a nice combination of margin and background color the assembly steps
may be easier to understand:

```sh
$ pdfcpu booklet -- "p:A4, bo:off, g:on, ma:10, bgcol:#beded9" zine.pdf 4 pageSequence.pdf
```

<p align="center">
  <img style="border-color:silver" border="1" src="resources/book4A4p1.png" height="300">
  <img style="border-color:silver" border="1" src="resources/book4A4p2.png" height="300">
</p>

<br>

### Booklet from image files

Similar to _nup_ and _grid_ the _booklet_ command also accepts a sequence of image files instead of a PDF input file.
In this case pdfcpu applies the same logic as above treating each image as a booklet page:

```sh
$ pdfcpu booklet -- "p:A4, g:on, ma:25, bgcol:#beded9" bookletFromImages.pdf 4 *.png
```

In the following example we have five input image files resulting in a booklet with five pages of content.<br>
Since we want to produce a 4-up booklet we can fit eight booklet pages on one sheet of paper.<br>
As a result of this pages 6 thru 8 are rendered as empty pages.<br>
Here are all pages (1 and 2) of the output file _bookletFromImages.pdf_:

<p align="center">
  <img style="border-color:silver" border="1" src="resources/book4A4Imp1.png" height="300">
  <img style="border-color:silver" border="1" src="resources/book4A4Imp2.png" height="300">
</p>
<br>

### Booklet with folios/signatures

- You bind your paper in eight sheet folios each making up 32 pages of your book.<br>
- Each sheet is going to make four pages of your book, gets printed on both sides and folded in half.<br>
- For such a multi folio booklet set 'multifolio:on' and play around with 'foliosize' which defaults to 8.

```sh
$ pdfcpu booklet -- "p:A4, multifolio:on, foliosize:8" hardbackbook.pdf 2 in.pdf
```
