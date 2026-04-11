package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/bastianrob/skyhold-pdf/processor"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var compressCmd = &cobra.Command{
	Use:   "compress",
	Short: "Standalone PDF compression",
	Long:  `Shrink PDF size by applying aggressive image downsampling and structural optimization without splitting into chunks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if inputPath == "" || outDir == "" {
			return fmt.Errorf("invalid arguments: --input and --out are required for compression")
		}

		if password == "" {
			password = os.Getenv("PDF_PASSWORD")
		}

		// 1. Open Input
		inputFile, err := os.Open(inputPath)
		if err != nil {
			return fmt.Errorf("failed to open source PDF: %w", err)
		}
		defer inputFile.Close()

		// 2. Setup UI (Progress & Logs)
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
						progressbar.OptionSetDescription("[green]Compressing[reset]"),
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

		// 3. Setup Factory
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

		config := processor.CompressConfig{
			Input:        inputFile,
			Password:     password,
			Quality:      quality,
			Concurrency:  concurrency,
			Scale:        scale,
			CreateWriter: createWriter,
			OnLog:        logFn,
			OnProgress:   progressFn,
		}

		err = processor.CompressPDF(config)
		if err == nil {
			if verbose {
				fmt.Printf("Successfully compressed PDF into %s\n", outDir)
			} else {
				fmt.Printf("Compression successful.\n")
			}
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(compressCmd)
}
