---
layout: default
title: "Piping"
---

# Piping

Many pdfcpu commands accept `-` as a conventional Unix placeholder for standard input or standard output.

Use `-` in an input PDF position to read a PDF from `stdin`.

Use `-` in an output PDF position to write a PDF to `stdout`.

This makes pdfcpu fit naturally into shell pipelines, object storage workflows and container jobs.

## Notes

The two `-` arguments in a command like this have different meanings:

```sh
pdfcpu optimize - -
```

The first `-` is the input PDF read from `stdin`.

The second `-` is the output PDF written to `stdout`.

For commands with non-PDF arguments, `-` support applies to PDF input and output positions only. JSON files, image files, CSV files, font files and other auxiliary inputs remain regular file arguments unless documented otherwise.

Avoid adding a separate `optimize` step after commands that already write an optimized PDF. For example, `merge`, `stamp`, `watermark`, `trim` and `rotate` produce processed PDF output directly. Use `optimize` as its own pipeline step when optimization is the operation you want to perform.

## Support Matrix

The matrix describes PDF piping support for command input and output positions.
Auxiliary files such as JSON, CSV, images, fonts and stamp source PDFs are regular file arguments unless noted otherwise.

<details>
<summary>Core Commands</summary>

| Command | stdin PDF | stdout PDF | Notes |
|:--|:--|:--|:--|
| `validate` | yes | no | Reads one or more input PDFs; `-` may appear as an input file. |
| `optimize` | yes | yes | Use `pdfcpu optimize - -` for a full PDF pipe. |
| `info` | yes | no | Supports text and JSON info output for streamed input. |
| `merge` | partial | yes | `merge` create mode accepts one stdin input; stdout is supported for create and zip mode, not append mode. |
| `split` | yes | no | Writes split PDFs into `outDir`. |
| `collect` | yes | yes | Creates a selected page sequence. |
| `trim` | yes | yes | Writes a processed PDF. |
| `rotate` | yes | yes | Writes a processed PDF. |
| `crop` | yes | yes | Writes a processed PDF. |
| `resize` | yes | yes | Writes a processed PDF. |
| `zoom` | yes | yes | Writes a processed PDF. |
| `stamp add/update/remove` | yes | yes | Stamp source files remain regular file arguments. |
| `watermark add/update/remove` | yes | yes | Watermark source files remain regular file arguments. |
| `signatures validate` | yes | no | Validation output is text. |
| `signatures remove` | yes | yes | Writes a processed PDF. |

</details>

<details>
<summary>Security Commands</summary>

| Command | stdin PDF | stdout PDF | Notes |
|:--|:--|:--|:--|
| `encrypt` | yes | yes | Writes encrypted PDF output. |
| `decrypt` | yes | yes | Writes decrypted PDF output. |
| `changeupw` | yes | yes | Writes a processed PDF. |
| `changeopw` | yes | yes | Writes a processed PDF. |
| `permissions list` | yes | no | Output is text. |
| `permissions set` | yes | yes | Writes a processed PDF. |

</details>

<details>
<summary>Page And Document Metadata Commands</summary>

| Command | stdin PDF | stdout PDF | Notes |
|:--|:--|:--|:--|
| `pages insert` | yes | yes | Writes a processed PDF. |
| `pages remove` | yes | yes | Writes a processed PDF. |
| `boxes list` | yes | no | Output is text. |
| `boxes add/remove` | yes | yes | Writes a processed PDF. |
| `annotations list` | yes | no | Output is text. |
| `annotations remove` | yes | yes | Writes a processed PDF. |
| `bookmarks list` | yes | no | Output is text. |
| `bookmarks export` | yes | no | Writes JSON to `outFileJSON`, not stdout. |
| `bookmarks import/remove` | yes | yes | JSON import file remains a regular file argument. |
| `keywords list` | yes | no | Output is text. |
| `keywords add/remove` | yes | yes | Writes a processed PDF. |
| `properties list` | yes | no | Output is text. |
| `properties add/remove` | yes | yes | Writes a processed PDF. |
| `pagelayout list` | yes | no | Output is text. |
| `pagelayout set/reset` | yes | yes | Writes a processed PDF. |
| `pagemode list` | yes | no | Output is text. |
| `pagemode set/reset` | yes | yes | Writes a processed PDF. |
| `viewerpref list` | yes | no | Supports text and JSON output for streamed input. |
| `viewerpref set/reset` | yes | yes | JSON input files remain regular file arguments. |

</details>

<details>
<summary>Generate And Extract Commands</summary>

| Command | stdin PDF | stdout PDF | Notes |
|:--|:--|:--|:--|
| `create` | yes | yes | JSON input file remains a regular file argument. |
| `import` | n/a | yes | Accepts one image from stdin as `imageFile`; writes PDF output to stdout. |
| `booklet` | yes | yes | PDF input supports stdin; image inputs remain regular file arguments. |
| `nup` | yes | yes | PDF input supports stdin; image inputs remain regular file arguments. |
| `grid` | yes | yes | PDF input supports stdin; image inputs remain regular file arguments. |
| `poster` | yes | no | Writes tile PDFs into `outDir`. |
| `ndown` | yes | no | Writes tile PDFs into `outDir`. |
| `cut` | yes | no | Writes tile PDFs into `outDir`. |
| `extract image/font/content/meta` | yes | no | Writes extracted files into `outDir`. |
| `extract page` | yes | partial | Use `outDir` as `-` only with one selected page. |
| `images list` | yes | no | Output is text. |
| `images extract` | yes | no | Writes extracted images into `outDir`. |
| `images update` | yes | yes | Replacement image file remains a regular file argument. |

</details>

<details>
<summary>Forms And Embedded Files</summary>

| Command | stdin PDF | stdout PDF | Notes |
|:--|:--|:--|:--|
| `form list` | yes | no | Output is text. |
| `form export` | yes | no | Writes JSON to `outFileJSON`, not stdout. |
| `form fill` | yes | yes | JSON data file remains a regular file argument. |
| `form lock/unlock/reset/remove` | yes | yes | Writes a processed PDF. |
| `form multifill` | yes | partial | stdout requires `--mode merge`; `outDir` is still used for generated instances. |
| `attachments list` | yes | no | Output is text. |
| `attachments add/remove` | yes | yes | Added/removed files remain regular file arguments. |
| `attachments extract` | yes | no | Writes extracted files into `outDir`. |
| `portfolio list` | yes | no | Output is text. |
| `portfolio add/remove` | yes | yes | Added/removed files remain regular file arguments. |
| `portfolio extract` | yes | no | Writes extracted entries into `outDir`. |

</details>

<br>

## Examples

Validate a PDF streamed from S3:

```sh
aws s3 cp s3://acme-invoices/invoice.pdf - \
   | pdfcpu validate -
```

Optimize a PDF without writing an intermediate local file:

```sh
aws s3 cp s3://acme-contracts/master.pdf - \
   | pdfcpu optimize - - \
   | aws s3 cp - s3://acme-contracts/optimized/master.pdf
```

Merge local PDFs and upload the merged PDF:

```sh
pdfcpu merge - quarterly/*.pdf \
   | aws s3 cp - s3://acme-reports/quarterly/merged.pdf
```

Read one merge input from S3:

```sh
aws s3 cp s3://acme-reports/cover.pdf - \
   | pdfcpu merge - - chapter1.pdf chapter2.pdf \
   | aws s3 cp - s3://acme-reports/book.pdf
```

Extract a single page and upload it:

```sh
aws s3 cp s3://acme-archive/contract.pdf - \
   | pdfcpu extract --mode page --pages 3 - - \
   | aws s3 cp - s3://acme-archive/pages/contract-page-3.pdf
```

Run a stateless Kubernetes job that trims and rotates a PDF:

```sh
aws s3 cp "$INPUT_URI" - \
   | pdfcpu trim --pages "$PAGES" - - \
   | pdfcpu rotate - "$ROTATION" - \
   | aws s3 cp - "$OUTPUT_URI"
```
