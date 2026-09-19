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

The example below uses `api.ValidateFile`:

```go
func ValidateFile(c context.Context, inFile string, conf *model.Configuration, options *ProgressOptions) error
```

| Parameter | Purpose | Argument in the example |
| --- | --- | --- |
| `c` | Context for cancellation; must not be `nil`. | `context.Background()` |
| `inFile` | Path to the PDF to validate. | `"input.pdf"` |
| `conf` | Configuration; `nil` uses the defaults. | First `nil` |
| `options` | Optional progress reporting; `nil` disables it. | Second `nil` |

`model.Configuration` is defined in `github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model`.<br><br>
`ProgressOptions` belongs to package `api`, so callers refer to it as `api.ProgressOptions`.
The function returns `nil` on success or an error if validation fails or the operation is canceled.

Example:

    package main

    import (
        "context"
        "log"

        "github.com/pdfcpu/pdfcpu/pkg/api"
    )

    func main() {
        if err := api.ValidateFile(context.Background(), "input.pdf", nil, nil); err != nil {
            log.Fatal(err)
        }
    }

---

## Context and cancellation

Long-running pdfcpu operations take a Go context as their first argument. Pass `context.Background()` when no cancellation
is needed, as in the example above. In a server or job runner, pass the request or job context instead. Canceling it asks
pdfcpu to stop and return an error. For file-producing operations, cancellation before final replacement preserves an
existing destination.

Do not pass a nil context.

---

## Configuration

Pass `nil` to use the defaults, or load an explicit configuration when your application needs custom settings, fonts or
trusted certificates. A configuration supplied by your application remains yours and can be reused after an operation.
Clone it before applying different settings for another job.

See [Configuration Modes](/config/config_modes) for loading options and
[Configuration Reset Required in v0.16](/getting_started/configuration_v016) when upgrading an existing application.

---

## Upgrading to v0.16

Long-running operations now use their regular API names with a required context. Validation and optimization operations
also accept an optional progress argument; pass `nil` when progress events are not needed.

The [v0.16 upgrade guide](/getting_started/configuration_v016) contains the configuration and progress examples.

---

## Documentation

* API: [pkg.go.dev docs](https://pkg.go.dev/github.com/pdfcpu/pdfcpu/pkg/api)
* Examples:
  * [API tests](https://github.com/pdfcpu/pdfcpu/tree/master/pkg/api/test)
  * [Sample files](https://github.com/pdfcpu/pdfcpu/tree/master/pkg/samples)

---

<img referrerpolicy="no-referrer-when-downgrade" src="https://static.scarf.sh/a.png?x-pxid=0b675754-cb2d-4989-bdb9-814aba0ea888" width="1" height="1" />
