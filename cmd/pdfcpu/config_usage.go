/*
Copyright 2026 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

const (
	paperSizes = `This is a list of predefined paper sizes:

   ISO 216:1975 A:
      4A0, 2A0, A0, A1, A2, A3, A4, A5, A6, A7, A8, A9, A10

   ISO 216:1975 B:
      B0+, B0, B1+, B1, B2+, B2, B3, B4, B5, B6, B7, B8, B9, B10

   ISO 269:1985 C:
      C0, C1, C2, C3, C4, C5, C6, C7, C8, C9, C10

   ISO 217:2013 untrimmed:
      RA0, RA1, RA2, RA3, RA4, SRA0, SRA1, SRA2, SRA3, SRA4, SRA1+, SRA2+, SRA3+, SRA3++

   American:
      SuperB(=B+),
      Tabloid (=ANSIB, DobleCarta), Ledger(=ANSIB, DobleCarta),
      Legal, GovLegal(=Oficio, Folio),
      Letter (=ANSIA, Carta, AmericanQuarto), GovLetter, Executive,
      HalfLetter (=Memo, Statement, Stationary),
      JuniorLegal (=IndexCard),
      Photo

   ANSI/ASME Y14.1:
      ANSIA (=Letter, Carta, AmericanQuarto),
      ANSIB (=Ledger, Tabloid, DobleCarta),
      ANSIC, ANSID, ANSIE, ANSIF

   ANSI/ASME Y14.1 Architectural series:
      ARCHA (=ARCH1),
      ARCHB (=ARCH2, ExtraTabloide),
      ARCHC (=ARCH3),
      ARCHD (=ARCH4),
      ARCHE (=ARCH6),
      ARCHE1 (=ARCH5),
      ARCHE2,
      ARCHE3

   American uncut:
      Bond, Book, Cover, Index, NewsPrint (=Tissue), Offset (=Text)

   English uncut:
      Crown, DoubleCrown, Quad, Demy, DoubleDemy, Medium, Royal, SuperRoyal,
      DoublePott, DoublePost, Foolscap, DoubleFoolscap

   F4

   China GB/T 148-1997 D Series:
      D0, D1, D2, D3, D4, D5, D6,
      RD0, RD1, RD2, RD3, RD4, RD5, RD6

   Japan:

   B-series variant:
      JIS-B0, JIS-B1, JIS-B2, JIS-B3, JIS-B4, JIS-B5, JIS-B6,
      JIS-B7, JIS-B8, JIS-B9, JIS-B10, JIS-B11, JIS-B12

   Shirokuban4, Shirokuban5, Shirokuban6
   Kiku4, Kiku5
   AB, B40, Shikisen`

	usageLongVersion = "Print the pdfcpu version & build info."

	usageLongPaper = "Print a list of supported paper sizes."

	usageLongSelectedPages = "Print definition of the -pages flag."

	usageLongConfig = `Manage pdfcpu configuration.

Configuration modes control how pdfcpu obtains settings and resources:

   auto
      Normal CLI mode. Uses config.yml, user fonts and trusted certificates
      from the selected configuration root. Missing resources may be created
      when a command needs them.

   stateless
      Selected with --conf disable. Uses built-in settings and the 14 core PDF
      fonts without accessing configuration files, user fonts or trusted
      certificates.

   read-only
      Loads an existing file-backed configuration without creating or changing
      anything. This mode is available to Go API applications. The CLI uses
      read-only loading internally for config inspect and config validate.
      There is no CLI flag that selects read-only mode for normal PDF commands.

auto and read-only use the same configuration-root selection:
--conf PATH, then PDFCPU_CONFIG_ROOT, then the operating system default.
--conf PATH changes the location; it does not change the mode.

config inspect reports the mode normal PDF commands would use for the selected
configuration. It may therefore report "mode: auto" even though inspection
itself is read-only and never writes configuration.

Typical workflows:

   Local CLI using automatic configuration at the default location:
      pdfcpu validate document.pdf

   Service or container using a managed configuration root:
      pdfcpu --conf /srv/app-config config init
      pdfcpu --conf /srv/app-config config validate
      pdfcpu --conf /srv/app-config validate document.pdf

   Diagnose a selected configuration without modifying it:
      pdfcpu --conf /srv/app-config config inspect
      pdfcpu --conf /srv/app-config config inspect --json

   Immutable container using only built-in configuration:
      pdfcpu --conf disable validate document.pdf`

	usageLongConfigInit = `Create missing configuration resources without replacing existing files.

The selected root contains a pdfcpu directory with config.yml, user fonts and trusted certificates.
Existing compatible files are preserved while missing resources are created.
If config.yml already exists but is incompatible or malformed, initialization stops and leaves it unchanged.
config init never upgrades or repairs an existing config.yml.
Stateless mode cannot be initialized.

Typical examples:
   Initialize the operating system default root:
      pdfcpu config init

   Initialize an explicit root:
      pdfcpu --conf /srv/app-config config init`

	usageLongConfigList = `Print the selected config.yml path followed by the file exactly as stored.

The command loads and validates the selected configuration before printing it.
Automatic mode may initialize a missing default tree; an existing compatible config.yml is not modified.
An incompatible schema produces upgrade or reset guidance instead of file content.
Command-line overrides are not reflected; use config inspect for selected paths and configured policy.

Typical examples:
   Print the default configuration:
      pdfcpu config list

   Print configuration from an explicit root:
      pdfcpu --conf /srv/app-config config list`

	usageLongConfigInspect = `Inspect selected configuration paths, schema, policy and limits without modifying configuration.

            mode ... selected configuration mode: auto | stateless
          source ... root selection source: flag | environment | os-default
         default ... true when source is os-default
       stateless ... true when built-in configuration is used without filesystem resources
   write capable ... true when the selected mode permits configuration writes

"write capable" does not mean every reported path is writable. See the writable field for each path.
The reported mode describes how normal operations use the selected configuration; inspection itself never writes.
The schema version section reports the detected and supported config.yml schema versions.
Configured values do not include per-operation command-line overrides such as --offline.

Typical examples:
   Inspect the selected configuration:
      pdfcpu config inspect

   Emit formatted JSON for automation:
      pdfcpu config inspect --json

   Confirm stateless operation:
      pdfcpu --conf disable config inspect`

	usageLongConfigValidate = `Validate the complete selected configuration without modifying or initializing it.

The command checks schema compatibility, unknown and duplicate keys, and configuration values.
An older schema produces reset guidance; a newer schema requires a newer pdfcpu version.
Stateless mode validates the built-in configuration without accessing filesystem resources.

Typical examples:
   Validate the default configuration:
      pdfcpu config validate

   Validate configuration from an explicit root:
      pdfcpu --conf /srv/app-config config validate

   Validate stateless defaults:
      pdfcpu --conf disable config validate`

	usageLongConfigReset = `Replace the selected config.yml with built-in defaults for the current configuration schema.

The command preserves user fonts and trusted certificates stored beside config.yml.
It can replace an incompatible or malformed config.yml because it does not load that file first.
Back up custom settings before resetting. Without --force, pdfcpu asks for confirmation.
Stateless mode cannot be reset.

Typical examples:
   Reset interactively:
      pdfcpu config reset

   Reset an explicit root without prompting:
      pdfcpu --conf /srv/app-config config reset --force`
)
