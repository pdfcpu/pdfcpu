---
layout: default
---

# About

`pdfcpu` is a PDF processor written in Go supporting encryption.

## `pdfcpu` the command line tool

Use shell scripts and the `pdfcpu` CLI to build your PDF processing pipelines for batch processing. 
`pdfcpu's` rich command line also allows the processing of encrypted files. You can use `pdfcpu` to manipulate your PDF files on the command line of all major platforms.  

## `pdfcpu` the Go library
Use the `pdfcpu` API to integrate PDF processing into your Go based backend systems.

Each operation is available as a file based version (also used by pdfcpu's CLI):
```
func OptimizeFile(inFile, outFile string, conf *pdf.Configuration) (err error)
```

and as an interface based version (typically using io.ReadSeeker/io.Writer):

```
func Optimize(rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error
```
Learn more about the API including examples for all operations at [GoDoc](https://godoc.org/github.com/hhrutter/pdfcpu/pkg/api).
