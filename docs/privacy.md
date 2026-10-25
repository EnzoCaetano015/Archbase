# Privacy policy

Last updated: October 25, 2026.

Archbase is a local command-line tool. It does not provide user accounts and does not collect, transmit, or store personal data, telemetry, analytics, usage metrics, crash reports, or advertising identifiers.

## Network access

Archbase accesses the network only as a direct result of an operation requested by the user. Examples include downloading a release through an official installer, querying GitHub release metadata with `arc update --check`, downloading an update with `arc update`, or cloning and refreshing a public remote Git registry configured by the user. Archbase does not make background update, analytics, or telemetry requests.

Downloads and repository hosting are provided by third parties such as GitHub and are subject to those services' own privacy policies and server logs.

## Local data

Archbase reads and writes local files only to perform commands requested by the user. Depending on the command, this can include project-scoped `.archbase` configuration and pattern files, generated rule exports, and a local cache for a configured registry. The official installers place the executable on the local machine and may update the user's `PATH` as documented in the [installation guide](installation.md).

Archbase does not send the contents of local projects to the Archbase maintainers.

Self-update reads the running installation and replaces its executable after validation, using a temporary sibling directory and lock. It does not read or modify project content or PATH. A Windows backup or a failed-rollback recovery copy may remain at the path reported by the updater.

## Contact

Questions or privacy reports can be submitted through the [Archbase GitHub repository](https://github.com/EnzoCaetano015/Archbase/issues). Security-sensitive reports should use [GitHub private vulnerability reporting](https://github.com/EnzoCaetano015/Archbase/security/advisories/new) when available.
