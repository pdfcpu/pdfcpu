---
layout: default
title: API Installation
---

# API Installation

Use pdfcpu as a Go library.

---

## Install

    go get github.com/pdfcpu/pdfcpu@latest

---

## Usage

Import the API package:

    import "github.com/pdfcpu/pdfcpu/pkg/api"

Example:

    package main

    import (
        "log"

        "github.com/pdfcpu/pdfcpu/pkg/api"
    )

    func main() {
        if err := api.ValidateFile("input.pdf", nil); err != nil {
            log.Fatal(err)
        }
    }

---

## Documentation

* API: [pkg.go.dev docs](https://pkg.go.dev/github.com/pdfcpu/pdfcpu/pkg/api)
* Examples:
  * [API tests](https://github.com/pdfcpu/pdfcpu/tree/master/pkg/api/test)
  * [Sample files](https://github.com/pdfcpu/pdfcpu/tree/master/pkg/samples)

---

<img referrerpolicy="no-referrer-when-downgrade" src="https://static.scarf.sh/a.png?x-pxid=0b675754-cb2d-4989-bdb9-814aba0ea888" />