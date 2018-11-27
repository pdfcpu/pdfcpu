# Note

* This is a consolidated version of `compress/lzw` that supports GIF, TIFF and PDF.
* Please refer to this [golang proposal](https://github.com/golang/go/issues/25409) for details.
* `pdfcpu` also hosts an improved version of Go's TIFF package at [github.com/hhrutter/pdfcpu/tiff](https://github.com/hhrutter/pdfcpu/tree/master/tiff) leveraging the improved `compress/lzw`.

## Background

* PDF's LZWDecode filter comes with the optional parameter `EarlyChange`.
* The type of this parameter is `int` and the defined values are 0 and 1.
* The default value is 1.

This parameter implies two variants of lzw. (See the [PDF spec](https://www.adobe.com/content/dam/acom/en/devnet/pdf/pdfs/PDF32000_2008.pdf)).

`compress/lzw`:

* the algorithm implied by EarlyChange value 1
* provides both Reader and Writer.

golang.org/x/image/tiff/lzw (mirrored [at](https://github.com/golang/image)):

* the algorithm implied by EarlyChange value 0
* provides a Reader, lacks a Writer

In addition PDF expects a leading `clear_table` marker right at the beginning
which is not smth the stdlib `compress/lzw` does.

There are numerous PDF Writers out there and the following can be observed on arbitrary PDF files that use the LZWDecode filter:

* Some PDF writers do not write the EOD (end of data) marker.
* Some PDF writers do not write the final bits after the EOD marker.

## Goal

A `compress/lzw` that works for a maximum number of components with a specific need for `lzw` support (as of now GIF, TIFF and PDF) and as a side effect of this an improved TIFF package that may or may not will make it into standard library one day.