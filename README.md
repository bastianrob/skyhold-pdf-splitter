# Skyhold PDF

**Skyhold PDF** is a high-performance Go-based CLI and library designed to split, merge, compress, and extract PDF files with extreme memory efficiency. It handles multi-gigabyte files with ease by utilizing stream-based indexing rather than loading full documents into RAM.

Built on top of [pdfcpu](https://github.com/pdfcpu/pdfcpu), it is optimized for cloud-native pipelines, supporting direct integration with Go `io` streams (S3, MinIO, or local storage).

## Key Capabilities
- **Memory Efficient**: Streams individual pages via indexing, perfect for massive PDFs.
- **Advanced Re-sampling**: Pure-Go image compression with worker-pool parallelization for aggressive size reduction.
- **Combined Merge & Compress**: Merge multiple PDFs and optimize the resulting file in a single processing pass.
- **Cloud Ready**: Plugs directly into any `io.ReadSeeker` or `io.Writer`.
- **Zero Dependencies**: Compiles to a single static binary without requiring Ghostscript or CGO.

## Installation

### For CLI usage:
Install the `pdf` binary directly into your `$GOPATH/bin`:

```bash
go install github.com/bastianrob/skyhold-pdf/cmd/pdf@latest
```

### For Library usage:
Add the `processor` package to your Go project:

```bash
go get github.com/bastianrob/skyhold-pdf
```

---

## CLI Usage

### 1. Splitting (Chunking)
Split a massive PDF into smaller batches of a specific page size.
```bash
pdf -i report.pdf -s 50 -o ./output/ -v
```

### 2. Merging (Combining)
Merge multiple PDF files into one. You can optionally compress the result.
```bash
pdf combine doc1.pdf doc2.pdf -o merged.pdf -c -q 60
```

### 3. Compression
Optimize an existing PDF's structure and downsample images.
```bash
# Aggressive optimization
pdf compress -i scan.pdf -o small.pdf -q 40 -m 50 -v
```

### 4. Extraction
Extract a targeted page range.
```bash
pdf extract -i source.pdf -f 10 -t 20 -o snippet.pdf
```

## Global Flags
- `-i, --input`: Source PDF file (Required for root, compress, extract)
- `-o, --out`: Destination file or directory (Required)
- `-s, --size`: Pages per chunk (Root command)
- `-c, --compress`: Enable structural and image optimization
- `-q, --quality`: JPEG compression quality (1-100, default: 60)
- `-m, --scale`: Image scaling factor (1-100, default: 100)
- `-j, --concurrency`: Parallel workers for optimization (Default: NumCPU)
- `-p, --password`: PDF password
- `-f, --force`: Overwrite existing files
- `-v, --verbose`: Enable progress bars and detailed logs

---

## Library Usage

### Example: Merging PDFs Programmatically
```go
import "github.com/bastianrob/skyhold-pdf/processor"

func main() {
    config := processor.CombineConfig{
        Inputs:   []io.ReadSeeker{file1, file2},
        Compress: true,
        Quality:  50,
        CreateWriter: func() (io.WriteCloser, error) {
            return os.Create("merged.pdf")
        },
        OnProgress: func(curr, total int) {
            fmt.Printf("Merging: %d/%d\n", curr, total)
        },
    }
    processor.CombinePDFs(config)
}
```

## License
Skyhold PDF is licensed under the [Apache License, Version 2.0](LICENSE).  
Copyright © 2026 Robin Bastian / SkyHold.
