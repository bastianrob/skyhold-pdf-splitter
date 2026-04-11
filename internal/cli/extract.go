package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/bastianrob/skyhold-pdf/processor"
)

var (
	from int
	to   int
)

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract a specific page range from a PDF",
	Long:  `Extracts a range of pages (from X to Y) into a single PDF file. Output can be a directory or a specific filename.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if password == "" {
			password = os.Getenv("PDF_PASSWORD")
		}

		// 1. Open Input
		inputFile, err := os.Open(inputPath)
		if err != nil {
			return fmt.Errorf("failed to open source PDF: %w", err)
		}
		defer inputFile.Close()

		// 2. Determine Output Path
		finalOutPath := outDir
		if !strings.HasSuffix(strings.ToLower(outDir), ".pdf") {
			// It's a directory
			if err := os.MkdirAll(outDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
			baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
			defaultName := fmt.Sprintf("%s-p%d-p%d.pdf", baseName, from, to)
			finalOutPath = filepath.Join(outDir, defaultName)
		} else {
			dir := filepath.Dir(finalOutPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		// 3. Setup UI
		var logFn func(string)
		if verbose {
			logFn = func(msg string) {
				fmt.Println(msg)
			}
		}

		// 4. Open Output Writer
		flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		if !force {
			flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
		}
		outFile, err := os.OpenFile(finalOutPath, flags, 0644)
		if err != nil {
			if os.IsExist(err) {
				return fmt.Errorf("file already exists: %s (use --force to overwrite)", finalOutPath)
			}
			return fmt.Errorf("failed to create file %s: %w", finalOutPath, err)
		}
		defer outFile.Close()

		config := processor.ExtractConfig{
			Input:    inputFile,
			From:     from,
			To:       to,
			Password: password,
			Output:   outFile,
			OnLog:    logFn,
		}

		err = processor.Extract(config)
		if err == nil {
			if verbose {
				fmt.Printf("Successfully extracted to: %s\n", finalOutPath)
			} else {
				fmt.Printf("Extracted pages %d-%d to %s\n", from, to, finalOutPath)
			}
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(extractCmd)

	extractCmd.Flags().IntVarP(&from, "from", "f", 0, "First page to extract (1-indexed)")
	extractCmd.Flags().IntVarP(&to, "to", "t", 0, "Last page to extract (inclusive)")

	extractCmd.MarkFlagRequired("from")
	extractCmd.MarkFlagRequired("to")
}
