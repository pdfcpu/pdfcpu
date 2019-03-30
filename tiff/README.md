 # Note

This package is an improved version of golang.org/x/image/tiff. It uses a consolidated version of `compress/lzw` (hosted at: [github.com/hhrutter/pdfcpu/lzw](https://github.com/hhrutter/pdfcpu/tree/master/lzw)) for compression and also adds support for CMYK.
 CCITT Group3/4 compression is supported for reading only.

## Background

As stated in this [golang proposal](https://github.com/golang/go/issues/25409) right now Go lzw implementations are spread out over the standard library(`compress/lzw`) and golang.org/x/image/tiff/lzw. As of go1.11 `compress/lzw` works reliably for GIF only. This is also the reason the TIFF package at golang.org/x/image/tiff provides its own lzw implementation for compression.

In addition with PDF there is a third variant of lzw needed.

`pdfcpu` supports lzw compression for PDF files and hosts a consolidated implementation of lzw at [github.com/hhrutter/pdfcpu/lzw](https://github.com/hhrutter/pdfcpu/lzw) which works for GIF, PDF and TIFF. It not only supports the PDF LZWFilter but also processing PDFs with embedded TIFF images. Therefore it also provides a variant of golang.org/x/image/tiff already leveraging the new consolidated lzw implementation([github.com/hhrutter/pdfcpu/lzw](https://github.com/hhrutter/pdfcpu/lzw)).

This implementation provides

* both lzw Reader and Writer as opposed to the original golang.org/x/image/tiff/lzw
* support for CMYK color models.

## Goal

A `compress/lzw` that works for a maximum number of components with a specific need for `lzw` support (as of now GIF, TIFF and PDF) and as a side effect of this an improved TIFF package that may or may not will make it into standard library one day.