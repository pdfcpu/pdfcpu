---
layout: default
title: "List Certificates"
---

# List Certificates

List the certificates currently managed by pdfcpu.

```
pdfcpu certificates list
pdfcpu certificates list --json
```

## Flags

| name | description | default | required |
|:-----|:------------|:--------|:---------|
| j(son) | output JSON | false | no |

## Examples

```sh
$ pdfcpu certificates list
trustedCertDir: /Users/horstrutter/Library/Application Support/pdfcpu/certs

...
```

Use JSON for machine-readable certificate metadata, including subjects, issuers, serial numbers, validity periods, CA status,
per-file errors, and the total installed certificate count:

```sh
$ pdfcpu certificates list --json
{
	"header": {
		"version": "pdfcpu vX.Y.Z",
		"creation": "YYYY-MM-DD HH:MM:SS TZ"
	},
	"trustedCertDir": "/Users/horstrutter/Library/Application Support/pdfcpu/certs",
	"totalInstalledCerts": 1,
	"files": [
		{
			"name": "/root-ca.pem",
			"certificates": [
				{
					"subject": {
						"commonName": "Example Root CA"
					},
					"issuer": {
						"commonName": "Example Root CA"
					},
					"serialNumber": "1",
					"notBefore": "2025-01-01",
					"notAfter": "2035-01-01",
					"isCA": true
				}
			]
		}
	]
}
```
