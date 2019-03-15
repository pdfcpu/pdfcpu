# pdfcpu: a golang pdf processor

[![Build Status](https://travis-ci.org/hhrutter/pdfcpu.svg?branch=master)](https://travis-ci.org/hhrutter/pdfcpu)
[![GoDoc](https://godoc.org/github.com/hhrutter/pdfcpu?status.svg)](https://godoc.org/github.com/hhrutter/pdfcpu)
[![Coverage Status](https://coveralls.io/repos/github/hhrutter/pdfcpu/badge.svg?branch=master)](https://coveralls.io/github/hhrutter/pdfcpu?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/hhrutter/pdfcpu)](https://goreportcard.com/report/github.com/hhrutter/pdfcpu)
[![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)](https://opensource.org/licenses/Apache-2.0)

![logo](resources/pdfchip3.png)

Package `pdfcpu` is a simple PDF processing library written in [Go](http://golang.org) supporting encryption.
It provides both an API and a CLI. Supported are all versions up to PDF 1.7 (ISO-32000). The current command set includes *validate*, *optimize*, *split*, *merge*, *extract*, *trim*, *stamp*, *watermark*, *import*, *nup*, *grid*, *rotate*, *attach*, *perm*, *encrypt*, *decrypt*, *changeupw*, *changeopw* and the list keeps growing.

Documentation is live at [pdfcpu.io](https://pdfcpu.io). If you go there you will also find examples for every command.

## Motivation

Document interchange is essential in the world we live in. PDF is everywhere and it is not going away anytime soon.<br>
This is an effort to build a PDF processing library from the ground up written in Go with strong support for batch processing via a rich command line. Over time `pdfcpu` aims to support the standard range of PDF processing features and also any interesting use cases that may present themselves along the way.


## Status

[Version: 0.1.21](https://github.com/hhrutter/pdfcpu/releases/tag/v0.1.21)

## Demo Screencast

(using older version with a smaller command set)

[![asciicast](resources/demo.png)](https://asciinema.org/a/P5jaAo9kgZXKj2iSA1OqIdLAU)

## Installation

Required build version: go1.9 and up

`go get github.com/hhrutter/pdfcpu/cmd/...`

## Contributing

### What

* Please open an issue if you find a bug or want to propose a change.
* Feature requests - always welcome!
* Bug fixes - always welcome!
* PRs - also welcome, although I can't promise a merge-in right now.
* `pdfcpu` is stable but still _alpha_ and occasionally undergoing heavy changes.

### How

* If you want to report a bug please attach the *very verbose* (`pdfcpu cmd -vv ...`) output and ideally a test PDF that you can share.
* Always make sure your contribution is based on the latest commit.
* Please sign your commits.
* Please sign the [CLA](https://cla-assistant.io/hhrutter/pdfcpu) before you submit a PR.

## Contributors

Thanks goes to these wonderful people:
<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
| [<img src="https://avatars1.githubusercontent.com/u/11322155?v=4" width="100px;"/><br /><sub><b>Horst Rutter</b></sub>](https://github.com/hhrutter)<br /> |[<img src="https://avatars0.githubusercontent.com/u/5140211?v=4" width="100px;"/><br /><sub><b>haldyr</b></sub>](https://github.com/haldyr)<br /> | [<img src="https://avatars3.githubusercontent.com/u/20608155?v=4" width="100px;"/><br /><sub><b>Vyacheslav</b></sub>](https://github.com/SimePel)<br /> | [<img src="https://avatars1.githubusercontent.com/u/617459?v=4" width="100px;"/><br /><sub><b>Erik Unger</b></sub>](https://github.com/ungerik)<br /> ||||
| :---: | :---: | :---: | :---: | :---: | :---: | :---: |
<!-- ALL-CONTRIBUTORS-LIST:END - Do not remove or modify this section -->

## Code of Conduct

Please note that this project is released with a Contributor [Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project you agree to abide by its terms.

## Disclaimer

Usage of `pdfcpu` assumes you know about and respect all copyrights of any PDF content you may be processing. This applies to the PDF files as such, their content and in particular all embedded resources like font files or images. Credit goes to [Renee French](https://instagram.com/reneefrench) for creating our beloved Gopher.

## License

Apache-2.0

## Powered By

<p align="center">
  <a href="https://golang.org"> <img src="resources/Go-Logo_Aqua.png" width="200"> </a>
</p>
