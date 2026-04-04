# PDF Chunker CLI Specification

## 1. Overview
A command-line interface (CLI) application designed to take a source PDF file and split it into multiple smaller PDF files based on a user-defined chunk size.

## 2. Tech Stack
* **Language:** Go (Golang)
* **Rationale:** Go provides immediate startup times, high performance for file I/O, and compiles down to a single, statically-linked binary. This ensures zero friction for end-users, as they do not need to install language runtimes or package managers to use the tool.

## 3. Architecture
The application is structured into four primary modules:

### 3.1. CLI Interface (Entrypoint)
* Handles POSIX-compliant flag parsing.
* Automatically generates and displays the `--help` menu.
* Passes parsed arguments to the Validator module.

### 3.2. Validator
* **Input File:** Confirms the source PDF exists, has correct read permissions, and is a valid PDF format.
* **Massive File Support:** The application must utilize **random access seeking** via file handles (`os.Open`) rather than loading the full content into memory. This ensures support for multi-GB files.
* **Encryption Check:** Detects if the PDF is password-protected. If no password is provided for an encrypted file, it prompts the user or exits with a dedicated error code.
* **Chunk Size:** Ensures the provided size fits within standard bounds (integer greater than `0`).
* **Output Directory:** Checks if the directory exists. If it does not, it recursively creates the required directory tree.
* **Logic Check:** Retrieves the total page count of the source PDF. If the chunk size is equal to or greater than the total page count, the application gracefully creates a single output file and alerts the user.

### 3.3. Core Processor (The Splitter)
* Opens and reads the PDF file graph.
* Calculates the number of required output chunks: `ceil(Total Pages / Chunk Size)`.
* Iterates through the document, calculating the precise `start_page` and `end_page` for each iteration.

### 3.4. File Writer
* **Calculates the target filename:**
  * Extracts the "base name" (everything before the final `.pdf` extension).
  * Appends the chunk number using zero-padding logic based on the total number of chunks to maintain alphabetical sorting: `{base-name}-{chunk number, zero padded}.pdf`.
  * *Example:* If source is `report.v1.pdf` (12 chunks), files will be `report.v1-01.pdf` through `report.v1-12.pdf`.
* **Resource Optimization:** Uses `pdfcpu` to extract specific page ranges efficiently.
* **Metadata Integrity:** Clones original document metadata (Author, Producer, etc.) to each output chunk where possible.
* **Write Operation:** Writes the resulting PDF chunk to the verified output directory. If a file already exists, it errors out unless the `--force` flag is used.

## 4. Dependencies
* `github.com/spf13/cobra`: Industry-standard Go library for building modern CLI applications, handling routing, and flag parsing.
* `github.com/pdfcpu/pdfcpu`: A robust, pure Go PDF processing library used for extracting page ranges and writing new PDFs without requiring C bindings (CGO).
* `github.com/schollz/progressbar/v3`: For thread-safe, robust terminal progress bar rendering during large file operations.

## 5. Input Parameters
The CLI must accept the following arguments:
* `--input` (short: `-i`): **Required**. The file path to the source PDF.
* `--size` (short: `-s`): **Required**. The number of pages per chunk.
* `--out` (short: `-o`): **Required**. The target directory for the chunked files.
* `--password` (short: `-p`): **Optional**. Password for encrypted PDFs. Can also be provided via `PDF_PASSWORD` environment variable.
* `--force` (short: `-f`): **Optional**. Overwrites existing files in the output directory.
* `--verbose` (short: `-v`): **Optional**. Enables detailed logging and displays a progress bar for large files.

**Usage Example:**
```bash
pdf-chunker --input ./docs/financial-report.pdf --size 5 --out ./processed-reports/
```

## 6. Success Criteria
* **Absolute Accuracy:** Every page from the source PDF must be present in the output directory exactly once. No pages can be duplicated, skipped, or corrupted.
* **Strict Naming Formatting:** Output files must perfectly match the `{base-name}-{chunk number, zero padded}.pdf` schema.
* **Safety & Idempotency:** By default, the tool must not overwrite existing files. If `--force` is provided, it must perform a clean overwrite.
* **Memory & Performance:** * **Optimized Memory Footprint:** The application must utilize stream-based reading to prevent loading the entire file payload into RAM, keeping overhead as low as the PDF structure allows.
    * **Streaming Output:** Chunks should be written as they are generated to minimize cumulative RAM consumption.
    * **Safe Concurrency:** If implementing concurrent chunking to reduce execution time, the application must use a worker pool where each worker holds its own isolated, read-only file handle (`os.File`) to prevent race conditions during read operations, or securely leverage `pdfcpu`'s native optimizations.
* **Encryption Handling:** Correctly handles password-protected files via flag, environment variable, or interactive prompt.
* **Graceful Error Handling:**
    * Fatal errors (e.g., file not found, permission denied) must print clear, actionable messages to `stderr` and exit with a non-zero status code (e.g., `exit status 1`).
    * Successful executions must print a brief summary to `stdout` and exit with status `0`.