---
layout: default
---

# Reset Configuration

The configuration file (config.yml) holds carefully selected default values for various aspects of pdfcpu's operations.

This command resets the configuration file to the current major version.

Please also check out the [config dir](../getting_started/config_dir.md) docs.


## Usage

```
pdfcpu config reset
```

Warning: Do not forget to backup your config.yml before you execute this command in case you have some customization going on.

## Background

Sometimes a new pdfcpu version introduces a new command that also extends the configuration eg. by a new parameter. 

Upgrading to a new version without upgrading the pdfcpu config file is not recommended.
It may or may not lead to side effects and in worse case to a hard landing.

Although the release notes will always include a reminder whenever a config file upgrade is necessary 
pdfcpu will output a warning whenever the config file version does not match the version of the current release.

Consider this config file version:
```
# version (Do not edit!)
version: v0.9.0 dev
```

and the current pdfcpu version:
```
$ pdfcpu version
pdfcpu: v0.9.1 dev
commit: 22ebeff8 (2024-10-18T19:51:48Z)
base  : go1.23.0
config: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml
```

The major version is 9 and match, so no upgrade is needed.

In contrast the config file version:
```
# version (Do not edit!)
version: v0.9.0 dev
```

and the pdfcpu version:
```
$ pdfcpu version
pdfcpu: v0.10.0 dev
commit: 22ebeff8 (2024-10-18T19:51:48Z)
base  : go1.23.0
config: /Users/horstrutter/Library/Application Support/pdfcpu/config.yml
```

have different major versions 9 and 10 respectively making a config file upgrade necessary.

## Output

The following is the output from upgrading v0.8.x to v0.9.0:

```
$ pdfcpu config reset
Did you make a backup of /Users/horstrutter/Library/Application Support/pdfcpu/config.yml ?
(yes/no): yes
Are you ready to reset your config.yml to v0.9.0 dev ?
(yes/no): yes
resetting..
Ready - Don't forget to update config.yml with your modifications.
```