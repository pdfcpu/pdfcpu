---
layout: default
---

# About

`pdfcpu` is a PDF processor written in Go supporting encryption.<br><br>
It is available on all major platforms and can process files based on all versions up to PDF V1.7.<br><br>
The parser which has been carefully crafted is able to handle many files violating the PDF specification and also repairs corrupt files on the fly.<br><br>
Support for PDF 2.0 is ongoing work and coming up.<br>
pdfcpu v0.6.0 introduced basic support for validation and as of v0.7.0 pdfcpu also supports writing back validated PDF 2.0 files.<br><br>
Support for new PDF 2.0 features will be implemented on an as-needed basis since PDF 2.0 files are hard to find.<br><br>
Please open an issue if you want to have a specific feature addressed and can share a test file.




## Use the Command Line Interface

Use shell scripts and the CLI to build your PDF processing pipelines for batch processing. `pdfcpu's` rich command line also allows the processing of encrypted files. You can use `pdfcpu` to manipulate your PDF files on the command line of all major platforms.  

## Use as Library
Use the `pdfcpu` API to integrate PDF processing into your Go based backend systems.

Each operation is available file based (also used by pdfcpu's CLI):
```
func OptimizeFile(inFile, outFile string, conf *pdf.Configuration) (err error)
```

and interface based (typically using io.ReadSeeker/io.Writer):
```
func Optimize(rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error
```

Learn more about the API including examples for many operations at [pkg.go.dev](https://pkg.go.dev/github.com/pdfcpu/pdfcpu/pkg/api).
