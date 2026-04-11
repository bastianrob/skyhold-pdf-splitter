package processor

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// CombineConfig holds the configuration for merging PDFs
type CombineConfig struct {
	Inputs   []io.ReadSeeker
	Password string
	Compress bool
	Quality     int
	Concurrency int
	Scale       int

	// Callbacks
	CreateWriter func() (io.WriteCloser, error)
	OnProgress   func(current, total int)
	OnLog        func(msg string)
}

// CombinePDFs merges multiple PDFs into one
func CombinePDFs(c CombineConfig) error {
	if len(c.Inputs) == 0 {
		return fmt.Errorf("no input files provided")
	}

	conf := model.NewDefaultConfiguration()
	conf.UserPW = c.Password
	conf.OwnerPW = c.Password
	conf.Optimize = c.Compress

	if c.OnLog != nil {
		c.OnLog(fmt.Sprintf("Merging %d files...", len(c.Inputs)))
	}

	// 1. Merge logic using higher-level MergeRaw function
	// We use MergeRaw because it accepts []io.ReadSeeker directly and handles internal structure integration safely.
	
	if c.OnProgress != nil {
		c.OnProgress(10, 100)
	}

	if c.Compress {
		// Merge into an intermediate buffer if we need to compress afterward
		var buf bytes.Buffer
		if err := api.MergeRaw(c.Inputs, &buf, false, conf); err != nil {
			return fmt.Errorf("merging failed: %v", err)
		}

		if c.OnProgress != nil {
			c.OnProgress(50, 100)
		}

		if c.OnLog != nil {
			c.OnLog("Applying image compression to merged document...")
		}

		// Use the existing compression logic on the merged result
		compressConf := CompressConfig{
			Input:        bytes.NewReader(buf.Bytes()),
			Password:     c.Password,
			Quality:      c.Quality,
			Concurrency:  c.Concurrency,
			Scale:        c.Scale,
			CreateWriter: c.CreateWriter,
			OnLog:        c.OnLog,
			OnProgress:   c.OnProgress,
		}
		return CompressPDF(compressConf)
	}

	// Direct merge to output if no compression needed
	writer, err := c.CreateWriter()
	if err != nil {
		return fmt.Errorf("failed to create writer: %w", err)
	}
	defer writer.Close()

	if err := api.MergeRaw(c.Inputs, writer, false, conf); err != nil {
		return fmt.Errorf("merging failed: %v", err)
	}

	if c.OnProgress != nil {
		c.OnProgress(100, 100)
	}

	return nil
}
