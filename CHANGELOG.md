# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.5] - 2024-03-08

- Automatically generates root cert, all domain-specific certificates will be signed by the root cert.
- Fix issue where system or default proxy will be used even in proxy bypass mode

## [0.1.4] - 2024-03-07

- Support traffic inspection and logging

## [0.1.3] - 2024-02-25

- Switched to use outbound IP querying services to validate backend proxies, instead of configured, single website+keyword checking
- Enhanced probe efficiency, performance, and accuracy

## [0.1.2] - 2024-02-23

- Establish basic changelog and version tagging
- Re-architect roprox to function as a standalone process, decoupled from other projects in agux

## [0.1.1] - yyyy-MM-dd

- Undocumented works... (apologies)

## [0.1.0] - 2018-07-12

- First commit to github
