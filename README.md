# Skyhold PDF Splitter

**Skyhold PDF Splitter** is a Go-based CLI and library used to split large PDF files into smaller chunks. It’s designed to handle multi-gigabyte files efficiently without crashing or using up all your system RAM.

It is built on top of [pdfcpu](https://github.com/pdfcpu/pdfcpu) and is optimized for cloud-native pipelines, allowing you to read and write directly to S3-compatible storage or any standard Go `io` stream.

## Key Capabilities
- **Memory Efficient**: Instead of loading the whole file into RAM, it indexes the PDF and streams individual pages.
- **Pure-Go Image Compression**: Supports aggressive JPEG re-sampling and structural optimization for scanned documents without requiring Ghostscript or CGO.
- **S3 & Stream Ready**: Uses `io.ReadSeeker` and `io.Writer` interfaces, so you can plug it into AWS S3, MinIO, or web servers easily.
- **Password Support**: Handles encrypted PDFs via CLI flags or the `PDF_PASSWORD` environment variable.
- **Safe Overwrites**: Uses standard OS flags to prevent accidental file corruption or race conditions.

## Installation

### For CLI usage:
Install the `pdf-chunker` binary directly into your `$GOPATH/bin`:

```bash
go install github.com/bastianrob/skyhold-pdf-splitter/cmd/pdf-chunker@latest
```

### For Library usage:
Add the `processor` package as a dependency to your Go project:

```bash
go get github.com/bastianrob/skyhold-pdf-splitter
```

*(Ensure your `$(go env GOPATH)/bin` directory is in your system `$PATH` for the CLI)*

## CLI Usage

### Chunking (Splitting) a PDF
Split an entire document into chunks of a designated size. You can optionally compress the output chunks to save space.

```bash
pdf-chunker -i ./massive_report.pdf -s 100 -o ./output_dir/ -v -c -q 50
```

### Standalone PDF Compression
Shrink a PDF without splitting it into chunks. This is ideal for reducing the size of image-heavy scans.

```bash
pdf-chunker compress -i ./large_scan.pdf -o ./compressed.pdf -q 60
# OR you can use the shorthand
pdf-chunker -i ./large_scan.pdf -c -o ./compressed.pdf -q 60
```

### Extracting a Specific Page Range
Extract a targeted slice of a document (e.g., Pages 10 to 50) and save it directly to a file.

```bash
pdf-chunker extract -i ./massive_report.pdf -f 10 -t 50 -o ./snippet.pdf -v
```
*(If you pass a directory to `-o` instead of a file name, the output will automatically be named `<base>-p10-p50.pdf`)*

### Global Flags
- `-i, --input`: Source PDF file (Required)
- `-o, --out`: Destination directory or file (Required)
- `-s, --size`: Number of pages per chunk (Required for splitting)
- `-p, --password`: Password for encrypted files (Optional, or use `PDF_PASSWORD` env var)
- `-c, --compress`: Enables structural optimization and image re-sampling (JPEG quality reduction)
- `-q, --quality`: Sets the JPEG compression quality (1-100, default: 60)
- `-f, --force`: Overwrite existing target files (Optional)
- `-v, --verbose`: Enables the CLI Progress Bar (`█░░░`) and detailed activity logs (Optional)

## Library Usage (Go Package)

You can import `processor` to chunk or extract PDFs securely from memory streams, cloud storage, or anywhere else that supports standard Go `io` interfaces.

### Example: Extracting via `io.Reader/Writer`

```go
package main

import (
    "os"
    "log"
    "github.com/bastianrob/skyhold-pdf-splitter/processor"
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

### Example: Standalone Compression

```go
config := processor.CompressConfig{
    Input:    file,
    Quality:  45, // Set aggressive JPEG compression
    CreateWriter: func() (io.WriteCloser, error) {
        return os.Create("optimized.pdf")
    },
}

if err := processor.CompressPDF(config); err != nil {
    log.Fatal(err)
}
```

### Example: Processing to/from S3 (via gocloud.dev)

Since PDF parsing requires an `io.ReadSeeker` (random access) to read the internal document index, remote S3 streams cannot be parsed on-the-fly. The standard approach is to download the source blob to a local temporary file first, but **stream the extracted chunks directly back to S3**.

```go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/bastianrob/skyhold-pdf-splitter/processor"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/s3blob"
)

func main() {
	ctx := context.Background()

	// 1. Connect to S3 compatible storage
	bucket, err := blob.OpenBucket(ctx, "s3://my-bucket?region=us-east-1")
	if err != nil {
		log.Fatalf("Failed to open bucket: %v", err)
	}
	defer bucket.Close()

	// 2. Download source PDF to a local temp file for io.ReadSeeker support
	s3Reader, _ := bucket.NewReader(ctx, "massive_report.pdf", nil)
	tempFile, _ := os.CreateTemp("", "source-*.pdf")
	io.Copy(tempFile, s3Reader)
	s3Reader.Close()
	
	// Reset pointer to the beginning of the file to prepare for parsing
	tempFile.Seek(0, io.SeekStart)
	defer os.Remove(tempFile.Name()) // Clean up locally
	defer tempFile.Close()

	// 3. Chunk PDF and stream outputs back to S3 directly
	config := processor.ChunkConfig{
		Input:    tempFile,
		PageSize: 100,
		CreateWriter: func(chunkIndex int, maxDigits int) (io.WriteCloser, error) {
			// Stream the output chunk directly to S3
			outKey := fmt.Sprintf("output/chunk-%0*d.pdf", maxDigits, chunkIndex)
			return bucket.NewWriter(ctx, outKey, nil)
		},
		OnLog: func(msg string) {
			log.Println(msg)
		},
	}

	if err := processor.Chunk(config); err != nil {
		log.Fatalf("Chunking failed: %v", err)
	}
}
```

## License

Skyhold PDF Splitter is licensed under the [Apache License, Version 2.0](LICENSE). 
Copyright © 2026 Robin Bastian / SkyHold.
