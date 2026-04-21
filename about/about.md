---
layout: default
---

# About

pdfcpu is a PDF processor written in Go.

It supports all PDF versions up to 1.7 and includes evolving support for PDF 2.0, including validation and writing of validated files.

The parser is designed for real-world PDFs and can handle many files that violate the specification, repairing common issues on the fly.

## PDF 2.0

Support for PDF 2.0 is ongoing.

New features are implemented on demand due to the limited availability of real-world test files.  
If you need support for a specific feature, please open an issue and provide a sample file if possible.

## Usage

### Command Line

Use the CLI to build PDF processing pipelines, including support for encrypted files.

### Go Library

Use the API to integrate PDF processing into Go applications.

Most operations are available in two forms:

File-based:

    func OptimizeFile(inFile, outFile string, conf *pdf.Configuration) error

Stream-based:

    func Optimize(rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error

More examples are available at [pkg.go.dev](https://pkg.go.dev/github.com/pdfcpu/pdfcpu/pkg/api).

For complete, real-world usage scenarios, see the [API tests](https://github.com/pdfcpu/pdfcpu/tree/master/pkg/api/test).
