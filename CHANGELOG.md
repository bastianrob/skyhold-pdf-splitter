# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-04-10

### Added
- Multi-core parallel image compression using a worker pool pattern for significantly faster processing (up to 10x).
- Percentage-based image scaling (downsampling) for aggressive file size reduction while maintaining legibility.
- New global CLI flags: `--scale` (`-m`) and `--concurrency` (`-j`).
- Standalone `compress` subcommand for optimizing PDFs without entry into the chunking pipeline.
- High-quality bi-linear interpolation for native pure-Go image resizing.

### Changed
- Refactored `processor` logic to decouple image processing from PDF context manipulation for thread safety.
- Updated progress reporting to provide granular, object-level status updates.
- Integrated `pdfcpu` structural optimization as a post-compression pass.

### Fixed
- Resolved a race condition during PDF object identification in multi-core mode.
- Corrected `/Length` metadata and `StreamLength` byte tracking for re-encoded image streams.
- Fixed an issue where the progress bar would prematurely stop at 10% for large documents.

## [1.0.0] - 2026-03-15
- Initial release with core PDF chunking and extraction capabilities.
- Support for S3-compatible streaming and RAM-efficient indexing.
