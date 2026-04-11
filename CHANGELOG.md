and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2026-04-11

### Added
- New `combine` command to merge multiple PDF files into a single document.
- Support for on-the-fly structural and image compression during the merging process.
- Library-level support for combining PDFs via `processor.CombinePDFs` with granular progress reporting.

### Changed
- **Major Rebrand**: Renamed the project to **Skyhold PDF** and updated the CLI binary name to `pdf`.
- Migrated Go module path to `github.com/bastianrob/skyhold-pdf`.
- Refreshed all documentation and usage examples to reflect the new identity.
- Improved progress tracking during complex multi-phase operations.

### Fixed
- Resolved a `nil pointer dereference` panic during `combine` operations by transitioning from low-level `MergeXRefTables` to the robust `api.MergeRaw` interface.

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
