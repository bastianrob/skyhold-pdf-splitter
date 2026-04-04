# Skyhold PDF Splitter

Skyhold PDF Splitter is a high-performance Go CLI and reusable library designed to chunk and extract pages from massive, multi-gigabyte PDF files securely and without memory exhaustion (OOM).

Under the hood, it uses the excellent [pdfcpu](https://github.com/pdfcpu/pdfcpu) engine, bypassing standard optimization logic to safely process files that would normally crash standard tools.

## Features
- **OOM-Safe Architecture**: Uses a single PDF index "map" and streams pages instead of loading gigabytes of data into RAM.
- **Robust IO**: Atomic file writes utilizing OS-level `O_EXCL` flags prevent accidental overwrites and race conditions.
- **Decrypted Workflows**: First-class support for password-protected PDFs via CLI flag or standard environment variables (useful for CI/CD pipelines).
- **Reusable API**: Designed with decoupled `io.ReadSeeker` and `io.Writer` interfaces, making it trivial to embed into backend web servers, AWS S3 pipelines, etc.

## Installation

Since the module is fully self-contained, you can install the CLI directly via standard Go tooling:

```bash
go install github.com/bastianrob/skyhold-pdf-splitter/cmd/pdf-chunker@latest
```

*(Ensure your `$(go env GOPATH)/bin` directory is in your system `$PATH`)*

## CLI Usage

### Chunking (Splitting) a PDF
Split an entire document into chunks of a designated size.

```bash
pdf-chunker -i ./massive_report.pdf -s 100 -o ./output_dir/ -v
```

### Extracting a Specific Page Range
Extract a targeted slice of a document (e.g., Pages 10 to 50) and save it directly to a file.

```bash
pdf-chunker extract -i ./massive_report.pdf -f 10 -t 50 -o ./snippet.pdf -v
```
*(If you pass a directory to `-o` instead of a file name, the output will automatically be named `<base>-p10-p50.pdf`)*

### Global Flags
- `--input` / `-i`: Path to the source file (Required)
- `--out` / `-o`: Path to the target directory or explicit file (Required)
- `--password` / `-p`: PDF decryption key (Optional. Will fallback to the `PDF_PASSWORD` env var).
- `--force` / `-f`: Overwrite existing target files (Optional)
- `--verbose` / `-v`: Enables the CLI Progress Bar (`█░░░`) and detailed activity logs (Optional)

## Library Usage (Go Package)

You can import `processor` to chunk or extract PDFs securely from memory streams, cloud storage, or anywhere else that supports standard Go `io` interfaces.

### Example: Extracting via `io.Reader/Writer`

```go
package main

import (
    "os"
    "log"
    "github.com/bastianrob/skyhold-pdf-splitter/internal/processor"
)

func main() {
    file, _ := os.Open("massive.pdf")
    defer file.Close()

    outFile, _ := os.Create("extracted.pdf")
    defer outFile.Close()

    config := processor.ExtractConfig{
        Input:    file,     // Any io.ReadSeeker
        Output:   outFile,  // Any io.Writer
        From:     10,
        To:       20,
        Password: "my_secret_password",
        OnLog: func(msg string) {
            log.Println(msg)
        },
    }

    if err := processor.Extract(config); err != nil {
        log.Fatalf("Extraction failed: %v", err)
    }
}
```

### Example: Splitting / Chunking

For chunking, you must provide a `CreateWriter` factory function that returns an `io.WriteCloser` for each requested chunk loop.

```go
config := processor.ChunkConfig{
    Input:    file,
    PageSize: 100,
    CreateWriter: func(chunkIndex int, maxDigits int) (io.WriteCloser, error) {
        // e.g. Return an S3 stream or a local file
        return os.Create(fmt.Sprintf("chunk-%d.pdf", chunkIndex))
    },
    OnProgress: func(current, total int) {
        fmt.Printf("Progress: %d/%d\n", current, total)
    },
}

processor.Chunk(config)
```
