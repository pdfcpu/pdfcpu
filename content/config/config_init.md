---
layout: default
title: "Initialize Configuration"
---

# Initialize Configuration

Create missing resources in the selected configuration tree without replacing existing files.

`config init` creates `config.yml`, the user-font directory and the trusted-certificate directory below the selected
configuration root. Existing compatible files are preserved while missing resources are created.


## Usage

```
pdfcpu config init
```

Use `--conf PATH` to select a configuration root. `--conf disable` selects stateless mode and cannot be initialized.

If `config.yml` already exists but is older, newer or malformed, initialization stops and leaves it unchanged. `config
init` never upgrades or repairs an existing `config.yml`.


## Output

First initialization using the macOS default root:

```
$ pdfcpu config init
configuration initialized
root: /Users/horstrutter/Library/Application Support
config: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml
fonts: /Users/horstrutter/Library/Application Support/pdfcpu/fonts
certificates: /Users/horstrutter/Library/Application Support/pdfcpu/certs
schema version: 1
```

Running the command again on a complete compatible tree reports `configuration already initialized` and leaves it
unchanged.

Paths vary by operating system and selected configuration root.
