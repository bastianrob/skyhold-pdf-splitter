package processor

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ChunkConfig holds the application configuration for splitting
type ChunkConfig struct {
	Input    io.ReadSeeker
	PageSize int
	Password string

	// Callbacks
	CreateWriter func(chunkIndex int, maxDigits int) (io.WriteCloser, error)
	OnProgress   func(current, total int)
	OnLog        func(msg string)
}

// ExtractConfig holds the configuration for extraction
type ExtractConfig struct {
	Input    io.ReadSeeker
	From     int
	To       int
	Password string

	// Callbacks
	Output     io.Writer
	OnProgress func(current, total int)
	OnLog      func(msg string)
}

// prepareContext opens the PDF and returns a validated context
func prepareContext(input io.ReadSeeker, password string, log func(string)) (*model.Context, error) {
	conf := model.NewDefaultConfiguration()
	conf.UserPW = password
	conf.OwnerPW = password
	conf.Optimize = false // Disable optimization for memory safety

	if log != nil {
		log("Initializing PDF context (reading indexing)...")
	}

	ctx, err := api.ReadAndValidate(input, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF (check password if encrypted): %v", err)
	}

	if ctx.PageCount == 0 {
		return nil, fmt.Errorf("the source PDF has 0 pages")
	}

	return ctx, nil
}

// Chunk executes the PDF splitting process
func Chunk(c ChunkConfig) error {
	ctx, err := prepareContext(c.Input, c.Password, c.OnLog)
	if err != nil {
		return err
	}

	totalPages := ctx.PageCount
	numChunks := int(math.Ceil(float64(totalPages) / float64(c.PageSize)))

	if c.OnLog != nil {
		c.OnLog(fmt.Sprintf("Strategy: Split %d pages into %d chunks of %d pages each", totalPages, numChunks, c.PageSize))
	}

	maxDigits := len(fmt.Sprintf("%d", numChunks))
	if maxDigits < 2 {
		maxDigits = 2
	}

	var errorMsgs []string

	for i := 1; i <= numChunks; i++ {
		start := (i - 1) * c.PageSize + 1
		end := i * c.PageSize
		if end > totalPages {
			end = totalPages
		}

		selectedPages := make([]int, 0, end-start+1)
		for p := start; p <= end; p++ {
			selectedPages = append(selectedPages, p)
		}

		ctxDest, err := pdfcpu.ExtractPages(ctx, selectedPages, false)
		if err != nil {
			errorMsgs = append(errorMsgs, fmt.Sprintf("failed to extract chunk %d (%d-%d): %v", i, start, end, err))
			continue
		}

		writer, err := c.CreateWriter(i, maxDigits)
		if err != nil {
			errorMsgs = append(errorMsgs, fmt.Sprintf("failed to create writer for chunk %d: %v", i, err))
			continue
		}

		if err := api.WriteContext(ctxDest, writer); err != nil {
			writer.Close()
			errorMsgs = append(errorMsgs, fmt.Sprintf("failed to write chunk %d: %v", i, err))
			continue
		}
		writer.Close()

		if c.OnProgress != nil {
			c.OnProgress(i, numChunks)
		}
	}

	if len(errorMsgs) > 0 {
		return fmt.Errorf("errors during chunking:\n- %s", strings.Join(errorMsgs, "\n- "))
	}

	return nil
}

// Extract executes the PDF extraction process (Range Mode)
func Extract(c ExtractConfig) error {
	ctx, err := prepareContext(c.Input, c.Password, c.OnLog)
	if err != nil {
		return err
	}

	// 1. Validation of range
	if c.From < 1 || c.To > ctx.PageCount || c.From > c.To {
		return fmt.Errorf("invalid range: %d-%d (total pages: %d)", c.From, c.To, ctx.PageCount)
	}

	if c.OnLog != nil {
		c.OnLog(fmt.Sprintf("Extracting pages %d to %d...", c.From, c.To))
	}

	// 2. Extraction
	pageNrs := []int{}
	for p := c.From; p <= c.To; p++ {
		pageNrs = append(pageNrs, p)
	}

	ctxDest, err := pdfcpu.ExtractPages(ctx, pageNrs, false)
	if err != nil {
		return fmt.Errorf("failed to extract range: %v", err)
	}

	// 3. Write
	if err := api.WriteContext(ctxDest, c.Output); err != nil {
		return err
	}

	if c.OnProgress != nil {
		c.OnProgress(1, 1)
	}

	if c.OnLog != nil {
		c.OnLog("Extraction successful.")
	}

	return nil
}
