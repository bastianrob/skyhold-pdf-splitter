package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/bastianrob/skyhold-pdf/processor"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var combineCmd = &cobra.Command{
	Use:   "combine [pdf1] [pdf2] ... [pdfN]",
	Short: "Combine multiple PDFs into one",
	Long:  `Merges multiple PDF files in the order they are provided. Supports optional compression and high-performance processing.`,
	Args:  cobra.MinimumNArgs(2),
	Example: `  pdf combine part1.pdf part2.pdf -o merged.pdf
  pdf combine *.pdf -o result.pdf --compress --quality 60`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if outDir == "" {
			return fmt.Errorf("invalid arguments: --out is required")
		}

		if password == "" {
			password = os.Getenv("PDF_PASSWORD")
		}

		// 1. Open all inputs
		// We'll close them manually at the end or use defers if we can guarantee they don't leak.
		// Since this is a CLI, defer is fine as the process exits.
		inputs := make([]io.ReadSeeker, 0, len(args))
		for _, path := range args {
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open %s: %w", path, err)
			}
			defer f.Close()
			inputs = append(inputs, f)
		}

		// 2. Setup UI
		var bar *progressbar.ProgressBar
		var logFn func(string)
		var progressFn func(int, int)

		if verbose {
			logFn = func(msg string) {
				fmt.Println(msg)
			}
			progressFn = func(current, total int) {
				if bar == nil {
					bar = progressbar.NewOptions(total,
						progressbar.OptionSetDescription("[green]Combining[reset]"),
						progressbar.OptionSetWriter(os.Stderr),
						progressbar.OptionShowCount(),
						progressbar.OptionSetWidth(40),
						progressbar.OptionEnableColorCodes(true),
						progressbar.OptionSetPredictTime(true),
						progressbar.OptionSetElapsedTime(true),
					)
				}
				bar.Set(current)
				if current == total {
					fmt.Println()
				}
			}
		}

		// 3. Setup Writer Factory
		createWriter := func() (io.WriteCloser, error) {
			flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
			if !force {
				flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
			}

			outFile, err := os.OpenFile(outDir, flags, 0644)
			if err != nil {
				if os.IsExist(err) {
					return nil, fmt.Errorf("file already exists: %s (use --force to overwrite)", outDir)
				}
				return nil, fmt.Errorf("failed to create file %s: %w", outDir, err)
			}
			return outFile, nil
		}

		config := processor.CombineConfig{
			Inputs:      inputs,
			Password:    password,
			Compress:    compress,
			Quality:     quality,
			Concurrency: concurrency,
			Scale:       scale,
			CreateWriter: createWriter,
			OnLog:        logFn,
			OnProgress:   progressFn,
		}

		err := processor.CombinePDFs(config)
		if err == nil {
			if verbose {
				fmt.Printf("Successfully combined PDFs into %s\n", outDir)
			} else {
				fmt.Printf("Combined PDFs.\n")
			}
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(combineCmd)
}
