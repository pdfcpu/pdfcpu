---
layout: default
title: "About"
---

# About

<style>
.about-grid {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  margin: 20px 0 26px;
}

.about-item {
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  padding: 14px 16px;
}

.about-item h3 {
  font-size: 1rem;
  margin: 0 0 6px;
}

.about-item p {
  margin: 0;
}

.about-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin: 16px 0 24px;
}

.about-actions a {
  border: 1px solid #d1d5db;
  border-radius: 6px;
  display: inline-block;
  font-weight: 700;
  padding: 8px 12px;
}
</style>

pdfcpu is a PDF processor written in Go. It is built for command-line workflows,
server-side automation, and Go applications that need direct control over PDF
files without pulling in a heavyweight runtime.

<div class="about-grid">
  <div class="about-item">
    <h3>Real-world PDFs</h3>
    <p>The parser handles many files that violate the specification and repairs common issues while processing.</p>
  </div>
  <div class="about-item">
    <h3>CLI and API</h3>
    <p>Use pdfcpu from scripts and terminals, or embed the same capabilities in Go applications.</p>
  </div>
  <div class="about-item">
    <h3>Broad Processing</h3>
    <p>Validate, optimize, split, merge, watermark, encrypt, inspect, extract, and manipulate PDF files.</p>
  </div>
  <div class="about-item">
    <h3>Modern PDF Work</h3>
    <p>pdfcpu supports PDF 1.7 and includes evolving support for PDF 2.0 validation and writing.</p>
  </div>
</div>

## PDF 2.0

PDF 2.0 support is ongoing and grows from practical use cases.

New features are implemented when there is enough real-world material to test against. If you need support for a specific feature, please open an issue and include a sample file where possible.

## Usage

pdfcpu has two primary entry points:

<div class="about-actions">
  <a href="/getting_started/install_cli/?src=docs">Install the CLI</a>
  <a href="/getting_started/install_api/?src=docs">Use the Go API</a>
</div>

### Command Line

Use the CLI to build PDF processing pipelines, including workflows for encrypted files.

### Go Library

Use the API to integrate PDF processing into Go applications. Most operations are available in file-based and stream-based forms:

```go
func OptimizeFile(inFile, outFile string, conf *pdf.Configuration) error

func Optimize(rs io.ReadSeeker, w io.Writer, conf *pdf.Configuration) error
```

More examples are available at [pkg.go.dev](https://pkg.go.dev/github.com/pdfcpu/pdfcpu/pkg/api).

For complete, real-world usage scenarios, see the [API tests](https://github.com/pdfcpu/pdfcpu/tree/master/pkg/api/test).
