package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/bastianrob/skyhold-pdf/processor"
)

var (
	inputPath string
	size      int
	outDir    string
	password  string
	force     bool
	verbose   bool
	compress    bool
	quality     int
	concurrency int
	scale       int
)

var rootCmd = &cobra.Command{
	Use:   "pdf",
	Short: "A high-performance PDF CLI tool",
	Long:  `A command-line interface designed to split, compress, extract and combine PDF files efficiently.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default behavior: Split into chunks
		if inputPath == "" || outDir == "" {
			return fmt.Errorf("invalid arguments: --input and --out are required")
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
						progressbar.OptionSetDescription("[green]Processing[reset]"),
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

		// If size is missing but compress is enabled, treat it as a standalone compression operation
		if size <= 0 && compress {
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
		}

		// Otherwise, proceed with chunking logic (requires size)
		if size <= 0 {
			return fmt.Errorf("invalid arguments: --size (must be > 0) is required for splitting. Use 'compress' subcommand or add --compress for standalone optimization")
		}

		// 3. Prepare directories
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))

		// 4. Setup Factory
		createWriter := func(chunkIndex int, maxDigits int) (io.WriteCloser, error) {
			outName := fmt.Sprintf("%s-%0*d.pdf", baseName, maxDigits, chunkIndex)
			outPath := filepath.Join(outDir, outName)

			flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
			if !force {
				flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
			}

			outFile, err := os.OpenFile(outPath, flags, 0644)
			if err != nil {
				if os.IsExist(err) {
					return nil, fmt.Errorf("file already exists: %s (use --force to overwrite)", outName)
				}
				return nil, fmt.Errorf("failed to create file %s: %w", outName, err)
			}
			return outFile, nil
		}

		// Start the processor
		config := processor.ChunkConfig{
			Input:        inputFile,
			PageSize:     size,
			Password:     password,
			Compress:     compress,
			Quality:      quality,
			Concurrency:  concurrency,
			Scale:        scale,
			CreateWriter: createWriter,
			OnLog:        logFn,
			OnProgress:   progressFn,
		}

		err = processor.Chunk(config)
		if err == nil {
			if verbose {
				fmt.Printf("Successfully created chunks in %s\n", outDir)
			} else {
				fmt.Printf("Created chunks.\n")
			}
		}
		return err
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Persistent flags (available to all subcommands)
	rootCmd.PersistentFlags().StringVarP(&inputPath, "input", "i", "", "The file path to the source PDF (Required)")
	rootCmd.PersistentFlags().StringVarP(&outDir, "out", "o", "", "The target directory or file (Required)")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "Password for encrypted PDFs (Optional)")
	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "Overwrite existing files")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enables detailed logging and progress bar")
	rootCmd.PersistentFlags().BoolVarP(&compress, "compress", "c", false, "Enable combined structural optimization and image compression")
	rootCmd.PersistentFlags().IntVarP(&quality, "quality", "q", 60, "JPEG compression quality from 1 to 100 (Default: 60)")
	rootCmd.PersistentFlags().IntVarP(&concurrency, "concurrency", "j", runtime.NumCPU(), "Number of parallel workers for image compression (Default: NumCPU)")
	rootCmd.PersistentFlags().IntVarP(&scale, "scale", "m", 100, "Image scaling factor in percentage (1-100, Default: 100)")

	// Local flags (only for the root command = split)
	rootCmd.Flags().IntVarP(&size, "size", "s", 0, "The number of pages per chunk (Required for split)")
}
