## Note
This package is an improved version of golang.org/x/image/tiff.

It uses a consolidated version of `compress/lzw` (hosted at: [github.com/hhrutter/pdfcpu/lzw](https://github.com/hhrutter/pdfcpu/lzw)) for compression and also adds support for CMYK.

## Background

As stated in this [golang proposal](https://github.com/golang/go/issues/25409) right now lzw implementations are spread out over the standard library(`compress/lzw`) and golang.org/x/image/tiff/lzw. 

As of go1.11 `compress/lzw` works reliably for GIF only. This is the reason the TIFF package at golang.org/x/image/tiff provides its own lzw implementation for compression needs. Additionally with PDF there is a third variant of lzw needed.

`pdfcpu` supports lzw compression for pdf and hosts a consolidated implementation of lzw at [github.com/hhrutter/pdfcpu/lzw](https://github.com/hhrutter/pdfcpu/lzw) which works for GIF,PDF and also TIFF.

`pdfcpu` not only supports the PDF LZWFilter but also processing pdfs with embedded tiff images.
Therefore it also provides a variant of golang.org/x/image/tiff already leveraging the new consolidated lzw implementation([github.com/hhrutter/pdfcpu/lzw](https://github.com/hhrutter/pdfcpu/lzw)). 

This implementation provides
* both lzw Reader and Writer as opposed to the original golang.org/x/image/tiff/lzw
* support for CMYK color models.

## Goal
The overall intention is to provide a future `compress/lwz` that works for a maximum number of components with specific needs to support lzw compression - right now including GIF, TIFF and PDF processing. A side effect of this goal is this improved TIFF package.