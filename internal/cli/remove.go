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
	pagesToRemove []int
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove specific pages from a PDF",
	Long:  `Removes specific pages from a PDF file. Output can be a directory or a specific filename.`,
	Example: `  pdf remove -i input.pdf -o output.pdf -p 23,25,49`,
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
			defaultName := fmt.Sprintf("%s-removed.pdf", baseName)
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

		config := processor.RemoveConfig{
			Input:    inputFile,
			Pages:    pagesToRemove,
			Password: password,
			Output:   outFile,
			OnLog:    logFn,
		}

		err = processor.Remove(config)
		if err == nil {
			if verbose {
				fmt.Printf("Successfully removed pages to: %s\n", finalOutPath)
			} else {
				fmt.Printf("Removed pages %v to %s\n", pagesToRemove, finalOutPath)
			}
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)

	removeCmd.Flags().IntSliceVarP(&pagesToRemove, "pages", "P", nil, "List of pages to remove (comma separated)")
	removeCmd.MarkFlagRequired("pages")
}
