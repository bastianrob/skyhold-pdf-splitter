# Add Compression Support During PDF Chunking

This implementation plan outlines the steps to add a `--compress` flag to the PDF Chunker CLI. This will allow the application to optimize and reduce the file size of the generated PDF chunks, which is especially useful for large documents.

## Problem Description
The user wants the ability to compress the size of a PDF, particularly targeting PDFs that originate from scanned images, which tend to be very large. 
The tool must support:
1. Compression *during* the splitting/chunking process (via a `--compress` flag).
2. Standalone compression *without* splitting (a dedicated `compress` workflow).
The solution must be built entirely with pure Go, using only OSS dependencies to allow the application to compile into a single static binary capable of running on mobile devices, avoiding external dependencies like Ghostscript.

## Proposed Changes

### 1. Update CLI Commands and Flags
#### [MODIFY] `internal/cli/root.go`
*   Add a new global boolean flag `--compress` (short: `-c`).
*   Add a new global integer flag `--quality` (short: `-q`) that defaults to `60` for controlling the JPEG output quality.
*   Bind these flags in the `init()` function of `root.go` so they can be used during chunking.

#### [NEW] `internal/cli/compress.go`
*   Add a dedicated `compress` subcommand (`pdf-chunker compress`).
*   This command will require `--input` and `--out` but will *not* require `--size`.
*   It will invoke a standalone `processor.CompressPDF()` function to cleanly compress a single PDF into a single output file.

### 2. Update Processor Configuration
#### [MODIFY] `processor/processor.go`
*   Add a `Compress bool` and `Quality int` fields to the `ChunkConfig` struct and an `ExtractConfig`.
*   Create a new configuration struct `CompressConfig` for standalone compression containing the `Quality` field.
*   In `prepareContext()`, pass the `Compress` configuration. If true, set `conf.Optimize = true`.
*   Write a new `CompressPDF(c CompressConfig) error` function. This function reads the entire `input`, runs the new Image Compression module (passing the `Quality` value) against the main context, and writes the single optimized PDF out without pagination logic.
*   Update the `Chunk` function to invoke the Image Compression module on `ctxDest` before writing each chunk if `Compress` is enabled, using the configured `Quality`.

### 3. Create Pure Go Image Compressor Module
#### [NEW] `processor/compressor.go`
*   Build a custom function `compressContextImages(ctx *model.Context, quality int)` that iterates through the PDF's objects (`ctx.Table`).
*   Identify entries that are Image `XObject` streams.
*   Extract the raw image bytes. 
*   Use standard Go `image`, `image/jpeg`, and `image/png` libraries (or `golang.org/x/image`) to decode the image.
*   Re-encode the image as JPEG using the passed `quality` setting (default `60`).
*   Replace the stream data in the `model.Context` object with the newly compressed bytes, updating the necessary PDF dictionaries (e.g., setting `/Filter` to `/DCTDecode` and modifying `/Length`).
*   This will drastically compress scanned documents purely in Go without external C libraries.

## Verification Plan

### Automated/Manual Verification
1.  **Build Binary:** Run `go build -o pdf-chunker ./cmd/pdf-chunker`. Make sure it builds without CGO dependencies.
2.  **Test Chunking With Compression:** Run `./pdf-chunker -c -i test/large-scan.pdf -s 5 -o output/` and verify the chunks are compressed properly.
3.  **Test Standalone Compression:** Run `./pdf-chunker compress -i test/large-scan.pdf -o output/compressed-scan.pdf`.
4.  **Validate Result:** Compare the standalone compressed PDF file size against the original file, open it to ensure readability, and verify zero data corruption.
