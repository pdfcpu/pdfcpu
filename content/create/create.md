---
layout: default
title: "Create"
---

# Create

allows you to generate PDF via JSON

* Create a new PDF file based on JSON input with optional form definition/data.

* Repeatedly add pages to an existing PDF file serving an incremental PDF generation approach.

* Append to content of specific pages.

* Include page headers and footers.

* Include boxes, images, text, tables.

* Create a PDF form by including date fields, text fields, checkboxes, radio button groups, comboboxes and listboxes.

* Supports Unicode / pdfcpu user fonts (installed Open/TrueType fonts).

* Use layout guides and visible crop/content box through out the layout process.

* Choose your preferred layout coordinate system.

<br>

## Usage

```
pdfcpu create inFileJSON [ inFile ] outFile [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description             | required |
|:-------------|:------------------------|:---------|
| inFileJSON   | JSON input file         | yes
| inFile       | PDF input file, use `-` to read from stdin | no
| outFile      | PDF output file, use `-` to write to stdout      | yes

<br>

## Local image access

The following policy applies to local images in JSON `create`, JSON `form fill`,
and JSON/CSV `form multifill`, including CSV `@img(...)` columns.

### CLI configuration

The CLI allows local images beneath the working directory by default. Image paths are relative to that directory,
not to the JSON or CSV file's location. To select a different asset directory, add this setting to your `config.yml`:

```yaml
localImageRoot: /srv/pdf-assets
```

An image source such as `"src": "logos/company.png"` then reads `/srv/pdf-assets/logos/company.png`.
The configured root may be an absolute directory or a directory relative to the process's working directory.
An empty setting keeps the CLI default. See [Configuration](/getting_started/config_dir) for config file locations
and selection.

### Go API configuration

An empty `conf.LocalImageRoot` disables local images in the Go API. Enable them by selecting an asset directory
in the configuration passed to the create or form-fill API:

```go
conf := model.NewStatelessConfiguration()
conf.LocalImageRoot = "/srv/pdf-assets"
```

Image paths are relative to this root. The input JSON or CSV cannot change the caller's access policy.
`LocalImageRoot` is an additive configuration field; existing function signatures are unchanged.

### Path restrictions and errors

Image sources must be relative paths beneath the selected root.

**Rejected paths and files:**

- Absolute and drive-qualified paths.
- Parent-directory components (`..`).
- Symlinks that escape the allowed root.
- Non-regular files, including directories and FIFOs.
- Paths containing colons or NUL characters.

Backslashes are treated as path separators. These restrictions apply after resolving named images and file/directory
aliases.

**Error behavior:**

- An access denial returns an error from the API and fails the CLI command.
- Within an allowed root, ordinary missing or invalid images retain their existing placeholder behavior.

### Migrating existing input

An older input might reference an image like this:

```json
{"src": "../../testdata/resources/logo.png"}
```

1. Select the intended `testdata` directory as the allowed root. Set `localImageRoot` for the CLI or
   `conf.LocalImageRoot` for the Go API.
2. Change the image source to a path relative to that root:

   ```json
   {"src": "resources/logo.png"}
   ```

The repository's JSON/CSV samples use this layout.

For an absolute image path, follow the same steps: select its asset directory as the root, then rewrite the source
relative to that directory.

### Security boundaries

Use a dedicated asset directory when processing untrusted input, and control its contents. Directory confinement
does not prevent access through hard links or mounted filesystems inside that directory. Operating-system permissions
still apply, and this policy does not bound the duration of filesystem I/O.

HTTP(S) images retain their existing network policy. Offline mode controls network access independently and does not
disable allowed local images. JavaScript/WASM builds do not support confined local image loading.

## Further Reference

```
$ pdfcpu create --help
``` 

and

```
pdfcpu/pkg/testdata/json/*
pdfcpu/pkg/samples/create/*
```

<br>

Create a PDF and write it to stdout:

```sh
$ pdfcpu create invoice.json - \
   | aws s3 cp - s3://acme-billing/invoice.pdf
```

<br>

Overlay generated content onto a streamed input PDF:

```sh
$ aws s3 cp s3://acme-forms/template.pdf - \
   | pdfcpu create overlay.json - - \
   | aws s3 cp - s3://acme-forms/template-filled.pdf
```
