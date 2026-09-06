---
layout: default
title: "Environment Variables"
---

# Environment Variables

These variables control configuration location and temporary storage. Normally, the operating system's defaults are
sufficient; you do not need to set them yourself.

## Configuration location

pdfcpu selects its configuration root in this order:

1. `--conf PATH` (or an explicit root supplied by a Go API caller).
2. A non-empty `PDFCPU_CONFIG_ROOT`.
3. The operating system's default configuration location.

The root is the **parent** of the `pdfcpu` directory. For example, `/srv/config` selects
`/srv/config/pdfcpu/config.yml`. `--conf disable` bypasses configuration-location selection.

| Variable | Platform | Effect |
|:--|:--|:--|
| `PDFCPU_CONFIG_ROOT` | All | Overrides the default configuration root unless an explicit root is supplied. |
| `XDG_CONFIG_HOME` | Linux and other Unix systems except macOS | Default root when non-empty; must be an absolute path. Otherwise uses `$HOME/.config`. |
| `HOME` | Unix | Supplies `$HOME/.config`, or `$HOME/Library/Application Support` on macOS. |
| `APPDATA` | Windows | Default configuration root, normally set by Windows. |

Set a root for one command in a Unix shell:

```sh
PDFCPU_CONFIG_ROOT=/srv/config pdfcpu config inspect
```

In PowerShell, set it for the current session:

```powershell
$env:PDFCPU_CONFIG_ROOT = 'C:\pdfcpu-config'
pdfcpu config inspect
```

An unset or empty `PDFCPU_CONFIG_ROOT` uses the OS default. If that default cannot be determined, pdfcpu reports an error.

## Temporary storage

PDF input from `stdin` and merged form multi-fill output to `stdout` use the OS temporary directory. The directory must
already exist, be writable and have enough free space.

| Variable | Platform | Effect |
|:--|:--|:--|
| `TMPDIR` | Unix | Selects temporary storage. If unset or empty, Go uses `/tmp`; macOS normally sets a per-user value. |
| `TMP`, `TEMP`, `USERPROFILE` | Windows | For ordinary user processes, Windows checks these in order, then falls back to the Windows directory. |
| `SystemTemp` | Windows SYSTEM processes | Overrides the system temporary directory when Windows provides `GetTempPath2`. |

Windows selection follows its [temporary-path API](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-gettemppath2w).

Choose temporary storage for one command in a Unix shell:

```sh
mkdir -p /path/to/scratch
TMPDIR=/path/to/scratch pdfcpu optimize - output.pdf < input.pdf
```

For a PowerShell session, set `$env:TMP` to an existing writable directory.

These variables do not move replacement staging: temporary replacement files stay beside their destinations, and font
installation stages inside the selected font tree.
