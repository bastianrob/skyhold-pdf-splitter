package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/bastianrob/skyhold-pdf-splitter/processor"
)

var (
	inputPath string
	size      int
	outDir    string
	password  string
	force     bool
	verbose   bool
)

var rootCmd = &cobra.Command{
	Use:   "pdf-chunker",
	Short: "A high-performance PDF chunker CLI",
	Long:  `A command-line interface designed to split large PDF files into smaller chunks or extract specific page ranges.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default behavior: Split into chunks
		if inputPath == "" || size <= 0 || outDir == "" {
			return fmt.Errorf("invalid arguments: --input, --size (must be > 0), and --out are required for splitting. Use 'extract' subcommand for extraction")
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

		// 2. Prepare directories
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))

		// 3. Setup UI (Progress & Logs)
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
						progressbar.OptionSetDescription("[green]Chunking[reset]"),
						progressbar.OptionSetWriter(os.Stderr),
						progressbar.OptionShowCount(),
						progressbar.OptionSetWidth(40),
						progressbar.OptionEnableColorCodes(true),
						progressbar.OptionSetPredictTime(true),
						progressbar.OptionSetElapsedTime(true),
						progressbar.OptionSetTheme(progressbar.Theme{
							Saucer:        "[green]█[reset]",
							SaucerHead:    "[green]█[reset]",
							SaucerPadding: "░",
							BarStart:      "|",
							BarEnd:        "|",
						}),
					)
				}
				bar.Add(1)
				if current == total {
					fmt.Println()
				}
			}
		}

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

	// Local flags (only for the root command = split)
	rootCmd.Flags().IntVarP(&size, "size", "s", 0, "The number of pages per chunk (Required for split)")

	rootCmd.MarkPersistentFlagRequired("input")
	rootCmd.MarkPersistentFlagRequired("out")
}
