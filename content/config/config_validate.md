---
layout: default
title: "Validate Configuration"
---

# Validate Configuration

Validate the selected configuration without modifying it.

Validation checks schema compatibility and the complete configuration content. It rejects malformed schema metadata,
unknown or duplicate schema-1 keys and invalid values.


## Usage

```
pdfcpu config validate
```

Use `--conf PATH` to validate a specific configuration root. With `--conf disable`, the command validates the built-in
stateless defaults.

The CLI invokes the read-only loader internally; there is no separate flag to select this behavior. An older schema
produces reset guidance; a newer schema requires upgrading pdfcpu. Validation never initializes, resets or migrates the
selected tree.


## Output

Example for a compatible configuration using the macOS default root:

```
$ pdfcpu config validate
configuration valid
config: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml
schema version: 1
```

Paths vary by operating system and selected configuration root.
