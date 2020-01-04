---
layout: default
---

# Page Selection

The `-pages ` flag allows you to select specific pages for processing when using the following commands:

* [trim](../core/split.md)
* [collect](../core/collect.md)
* [extract](../extract/extract.md)
* [rotate](../core/rotate.md)
* [stamp/watermark](../core/stamp.md)
* [grid](../core/grid.md)
* [nup](../core/nup.md)
* [pages](../pages/pages.md)

## Syntax

The value of this flag is a string which is a comma separated list of expressions containing page numbers or page number ranges:

| expression | page selection
|:-----------|:-----------
| even       | include even pages
| odd        | include odd pages
| #          | include page #
| #-#        | include page range
| !#         | exclude page #
| !#-#       | exclude page range
| #-         | include page # - last page
| -#         | include first page - page #
| !#-        | exclude page # - last page
| !-#        | exclude first page - page #
| l          | include last page
| nl         | exclude last page
| #-l-1      | include page # - (last page-1)

You can use either `!` or `n` for negating an expression.<br>
`!` needs to be escaped with single quotes on the command line.


<br>

## Examples

Select the first three pages, page 5 and page 7 up to the end of the document:
```sh
-pages -3,5,7-
```

<br>

Select pages 4 to 7 but exclude page 6:

```sh
-pages "4-7,!6"
``` 

<br>

Select all pages other than page 5:

```sh
-pages "1-,!5" 
```

<br>

Select all odd pages and exclude page 1 and the last page:

```sh
-pages odd,n1,nl
```

<br>

Select all even pages and ensure the last page:

```sh
-pages even,l
```