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
	Compress bool
	Quality     int
	Concurrency int
	Scale       int

	// Callbacks
	CreateWriter func(chunkIndex int, maxDigits int) (io.WriteCloser, error)
	OnProgress   func(current, total int)
	OnLog        func(msg string)
}

// CompressConfig holds the configuration for standalone compression
type CompressConfig struct {
	Input    io.ReadSeeker
	Password string
	Quality     int
	Concurrency int
	Scale       int

	// Callbacks
	CreateWriter func() (io.WriteCloser, error)
	OnProgress   func(current, total int)
	OnLog        func(msg string)
}

// ExtractConfig holds the configuration for extraction
type ExtractConfig struct {
	Input    io.ReadSeeker
	From     int
	To       int
	Password    string
	Concurrency int
	Scale       int

	// Callbacks
	Output     io.Writer
	OnProgress func(current, total int)
	OnLog      func(msg string)
}

// RemoveConfig holds the configuration for page removal
type RemoveConfig struct {
	Input    io.ReadSeeker
	Pages    []int
	Password string

	// Callbacks
	Output     io.Writer
	OnProgress func(current, total int)
	OnLog      func(msg string)
}

// prepareContext opens the PDF and returns a validated context
func prepareContext(input io.ReadSeeker, password string, compress bool, log func(string)) (*model.Context, error) {
	conf := model.NewDefaultConfiguration()
	conf.UserPW = password
	conf.OwnerPW = password
	conf.Optimize = compress // Enable structural optimization if compression is requested
	conf.ValidationMode = model.ValidationRelaxed
	conf.Reader15 = true
	conf.PostProcessValidate = false

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
	ctx, err := prepareContext(c.Input, c.Password, c.Compress, c.OnLog)
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

		if c.Compress {
			if err := compressContextImages(ctxDest, c.Quality, c.Concurrency, c.Scale, nil); err != nil && c.OnLog != nil {
				c.OnLog(fmt.Sprintf("warning: image compression failed on chunk %d: %v", i, err))
			}
			// Important: Ensure the context actually triggers structural optimization natively if desired
			if ctxDest.Configuration != nil {
				ctxDest.Configuration.Optimize = true
			}
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

// CompressPDF executes the standalone PDF compression process without splitting
func CompressPDF(c CompressConfig) error {
	ctx, err := prepareContext(c.Input, c.Password, true, c.OnLog)
	if err != nil {
		return err
	}

	if c.OnLog != nil {
		c.OnLog("Compressing image streams (parallel)...")
	}

	// 1. Process image objects with granular progress and concurrency
	if err := compressContextImages(ctx, c.Quality, c.Concurrency, c.Scale, c.OnProgress); err != nil && c.OnLog != nil {
		c.OnLog(fmt.Sprintf("warning: encountered issue compressing images: %v", err))
	}

	// 2. Pre-optimization pass explicitly trigger pdfcpu cleanup if Optimize is true
	if err := api.OptimizeContext(ctx); err != nil && c.OnLog != nil {
		c.OnLog(fmt.Sprintf("warning: structural optimization had issues: %v", err))
	}

	// 3. Write
	writer, err := c.CreateWriter()
	if err != nil {
		return fmt.Errorf("failed to create writer: %w", err)
	}
	defer writer.Close()

	if err := api.WriteContext(ctx, writer); err != nil {
		return fmt.Errorf("failed to write compressed PDF: %w", err)
	}

	if c.OnProgress != nil {
		c.OnProgress(1, 1)
	}

	if c.OnLog != nil {
		c.OnLog("Compression successful.")
	}

	return nil
}

// Extract executes the PDF extraction process (Range Mode)
func Extract(c ExtractConfig) error {
	ctx, err := prepareContext(c.Input, c.Password, false, c.OnLog)
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

// Remove executes the PDF page removal process
func Remove(c RemoveConfig) error {
	ctx, err := prepareContext(c.Input, c.Password, false, c.OnLog)
	if err != nil {
		return err
	}

	if len(c.Pages) == 0 {
		return fmt.Errorf("no pages specified for removal")
	}

	if c.OnLog != nil {
		c.OnLog(fmt.Sprintf("Removing %d pages...", len(c.Pages)))
	}

	// Convert []int to []string for pdfcpu
	pageStrings := make([]string, 0, len(c.Pages))
	for _, p := range c.Pages {
		if p < 1 || p > ctx.PageCount {
			return fmt.Errorf("invalid page number: %d (total pages: %d)", p, ctx.PageCount)
		}
		pageStrings = append(pageStrings, fmt.Sprintf("%d", p))
	}

	// Use api.RemovePages which accepts []string
	// We need to re-open the input because api.RemovePages takes an io.ReadSeeker
	// and we already read it in prepareContext. But wait, we can just use the
	// context and call RemovePages on it?
	// api.RemovePages doesn't take a context.
	
	// Since we are already at the end of the file, let's just use api.RemovePages
	// and reset the seeker.
	if _, err := c.Input.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to reset input seeker: %w", err)
	}

	conf := model.NewDefaultConfiguration()
	conf.UserPW = c.Password
	conf.OwnerPW = c.Password
	conf.ValidationMode = model.ValidationRelaxed
	conf.Reader15 = true

	if err := api.RemovePages(c.Input, c.Output, pageStrings, conf); err != nil {
		return fmt.Errorf("failed to remove pages: %v", err)
	}

	if c.OnProgress != nil {
		c.OnProgress(1, 1)
	}

	if c.OnLog != nil {
		c.OnLog("Removal successful.")
	}

	return nil
}
