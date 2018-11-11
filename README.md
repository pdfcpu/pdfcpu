# pdfcpu: a golang pdf processor

[![Build Status](https://travis-ci.org/hhrutter/pdfcpu.svg?branch=master)](https://travis-ci.org/hhrutter/pdfcpu)
[![GoDoc](https://godoc.org/github.com/hhrutter/pdfcpu?status.svg)](https://godoc.org/github.com/hhrutter/pdfcpu)
[![Coverage Status](https://coveralls.io/repos/github/hhrutter/pdfcpu/badge.svg?branch=master)](https://coveralls.io/github/hhrutter/pdfcpu?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/hhrutter/pdfcpu)](https://goreportcard.com/report/github.com/hhrutter/pdfcpu)
[![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)](https://opensource.org/licenses/Apache-2.0)

![logo](resources/pdfchip3.png)

Package pdfcpu is a simple PDF processing library written in [Go](http://golang.org) supporting encryption.
It provides both an API and a CLI. Supported are all versions up to PDF 1.7 (ISO-32000).

## Status

Version: 0.1.18

* Extended API to support webserver scenarios using Readseeker and Writer.
* Support for watermarking/stamping with a specific page of another PDF file.
* Extended logging into horizontal (Info, Debug, Trace etc.) vs. vertical logging (Read, Validate, Write etc).
* The CLI will produce regular logging if you use -verbose, or -v.
* The CLI will produce verbose logging if you use -vv.
* More tests in `api/process_test.go`
* More examples in `api/example_test.go`
* More scripts under `_scripts/*`
* Fixed #5, #39, #44

<p align="center">
  <img src="resources/wmTextSample.png" height="254">&nbsp;&nbsp;&nbsp;
  <img src="resources/wmImageSample.jpg" width="180">&nbsp;&nbsp;&nbsp;
  <img src="resources/wmPDFSample.jpg" width="180">
</p>

## Motivation

This is an effort to build a PDF processing library from the ground up written in Go with strong support for batch processing via a rich command line. Over time `pdfcpu` aims to support the standard range of PDF processing features and also any interesting use cases that may present themselves along the way.

One example is reducing the size of large PDF files for mass mailings by optimization to the bare minimum. This can be achieved by analyzing a PDF's cross reference table, removing redundant embedded resources like font files or images and by always writing back the file maxing out PDF compression. I also wanted to have my own swiss army knife for PDFs written entirely in [Go](http://golang.org) that allows me to trim, split, stamp and merge PDF content.

## Features

* Validate (validates PDF files up to version 7.0)
* Read (builds xref table from PDF file)
* Write (writes xref table to PDF file)
* Optimize (gets rid of redundancies like duplicate fonts, images)
* Split (split a multi page PDF file into single page PDF files)
* Merge (a set of PDF files into one consolidated PDF file)
* Extract Images (extract all embedded images of a PDF file into a given dir)
* Extract Fonts (extract all embedded fonts of a PDF file into a given dir)
* Extract Pages (extract specific pages into a given dir)
* Extract Content (extract the PDF-Source into given dir)
* Extract Metadata (extract XML metadata)
* Trim (generate a custom version of a PDF file)
* Stamp/Watermark selected pages with text, image or PDF page
* Manage (add,remove,list,extract) embedded file attachments
* Encrypt (sets password protection)
* Decrypt (removes password protection)
* Change user/owner password
* Manage (add,list) user access permissions

## Demo Screencast (this is an older version with a smaller command set)

[![asciicast](resources/demo.png)](https://asciinema.org/a/P5jaAo9kgZXKj2iSA1OqIdLAU)

## Installation

Required build version: go1.9 and up

`go get github.com/hhrutter/pdfcpu/cmd/...`

## Usage

    pdfcpu validate [-verbose] [-mode strict|relaxed] [-upw userpw] [-opw ownerpw] inFile
    pdfcpu optimize [-verbose] [-stats csvFile] [-upw userpw] [-opw ownerpw] inFile [outFile]
    pdfcpu split [-verbose] [-upw userpw] [-opw ownerpw] inFile outDir
    pdfcpu merge [-verbose] outFile inFile...
    pdfcpu extract [-verbose] -mode image|font|content|page|meta [-pages pageSelection] [-upw userpw] [-opw ownerpw] inFile outDir
    pdfcpu trim [-verbose] -pages pageSelection [-upw userpw] [-opw ownerpw] inFile outFile
    pdfcpu stamp [-verbose] -pages pageSelection description inFile [outFile]
    pdfcpu watermark [-verbose] -pages pageSelection description inFile [outFile]

    pdfcpu attach list [-verbose] [-upw userpw] [-opw ownerpw] inFile
    pdfcpu attach add [-verbose] [-upw userpw] [-opw ownerpw] inFile file...
    pdfcpu attach remove [-verbose] [-upw userpw] [-opw ownerpw] inFile [file...]
    pdfcpu attach extract [-verbose] [-upw userpw] [-opw ownerpw] inFile outDir [file...]

    pdfcpu encrypt [-verbose] [-mode rc4|aes] [-key 40|128] [-perm none|all] [-upw userpw] [-opw ownerpw] inFile [outFile]
    pdfcpu decrypt [-verbose] [-upw userpw] [-opw ownerpw] inFile [outFile]
    pdfcpu changeupw [-verbose] [-opw ownerpw] inFile upwOld upwNew
    pdfcpu changeopw [-verbose] [-upw userpw] inFile opwOld opwNew

    pdfcpu perm list [-verbose] [-upw userpw] [-opw ownerpw] inFile
    pdfcpu perm add [-verbose] [-perm none|all] [-upw userpw] -opw ownerpw inFile

    pdfcpu version

 [Please read the documentation](https://godoc.org/github.com/hhrutter/pdfcpu)

## Contributing

* Please open an issue if you find a bug or want to propose a change.
* Feature requests - always welcome
* Bug fixes - always welcome
* PRs - also welcome, although I can't promise a merge-in right now since `pdfcpu` is stable but still _alpha_ and occasionally undergoing heavy changes.

## Disclaimer

Usage of `pdfcpu` assumes you know about and respect all copyrights of any PDF content you may be processing. This applies to the PDF files as such, their content and in particular all embedded resources like font files or images. Credit goes to [Renee French](https://instagram.com/reneefrench) for creating our beloved Gopher.

## License

Apache-2.0

## Powered By

<p align="center">
  <a href="https://golang.org"> <img src="resources/Go-Logo_Aqua.png" width="200"> </a>
</p>
