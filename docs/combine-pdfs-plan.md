# Add Ability to Combine Multiple PDFs

This implementation plan outlines the steps to add a `combine` subcommand to the Skyhold PDF CLI. This allows users to merge multiple separate PDF files into a single continuous document, with optional on-the-fly compression.

## Problem Description
The tool now supports merging multiple PDF documents.
The solution:
1. Supports multiple input `.pdf` files via positional arguments.
2. Generates a single `.pdf` file.
3. Supports optional structural and image compression during the merge pass.
4. Provides library-level support for integration into other Go projects.

## Proposed Changes

### 1. Update CLI
#### [NEW] `internal/cli/combine.go`
* Subcommand: `pdf combine [file1] [file2] ... -o result.pdf`.
* Supports `--compress`, `--quality`, and other global optimization flags.

### 2. Implement Processor Core
#### [NEW] `processor/combine.go`
* `CombineConfig` struct for library usage.
* `CombinePDFs(c CombineConfig) error` logic.
* Integrated with the worker-pool image compressor.

## Verification Plan

### Automated/Manual Verification
1.  **Build Binary:** Run `go build -o pdf ./cmd/pdf`.
2.  **Test Merge:** Run `./pdf combine part1.pdf part2.pdf -o combined.pdf`.
3.  **Test Merge + Compress:** Run `./pdf combine part1.pdf part2.pdf -o combined.pdf --compress --quality 50`.
