// cmd/build.go
package cmd

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Bitlatte/S.H.I.T/internal/config"
	"github.com/Bitlatte/S.H.I.T/internal/model"

	"github.com/adrg/frontmatter"
	"github.com/spf13/cobra"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	conventionalContentDir = "content"
	conventionalLayoutsDir = "layouts"
	conventionalBaseLayout = "base.html" // The main layout file to execute for pages
	conventionalStaticDir  = "static"    // For static assets
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Builds the static site from content, layouts, and static assets",
	Long: `The build command processes Markdown files from './content/',
extracts frontmatter, applies templates from './layouts/' (including partials),
copies static assets from './static/', and generates the site in the
configured output directory (default './public/').`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBuildProcess(appConfig)
	},
}

func runBuildProcess(cfg config.Config) error {
	fmt.Println("Starting SHIT SSG build process...") // This log might be duplicated if serve calls it too. Consider passing a logger or a quiet flag.
	fmt.Printf("Using OutputDir: '%s', BaseURL: '%s', SiteTitle: '%s'\n", cfg.OutputDir, cfg.BaseURL, cfg.SiteTitle)

	mdParser := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			gmhtml.WithHardWraps(),
		),
	)

	sourceDir := conventionalContentDir
	layoutsDir := conventionalLayoutsDir
	staticDir := conventionalStaticDir
	outputDir := cfg.OutputDir

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("conventional source directory '%s' not found. Please create it and add your Markdown files", sourceDir)
	}
	if _, err := os.Stat(layoutsDir); os.IsNotExist(err) {
		return fmt.Errorf("conventional layouts directory '%s' not found. Please create it and add your .html layout files", layoutsDir)
	}

	fmt.Printf("Cleaning output directory: %s\n", outputDir)
	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("failed to remove output directory '%s': %w", outputDir, err)
	}
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory '%s': %w", outputDir, err)
	}
	fmt.Printf("Output directory '%s' prepared.\n", outputDir)

	if _, err := os.Stat(staticDir); !os.IsNotExist(err) {
		fmt.Printf("Copying static assets from '%s' to '%s'\n", staticDir, outputDir)
		if err := copyDirContents(staticDir, outputDir); err != nil {
			return fmt.Errorf("failed to copy static assets: %w", err)
		}
		fmt.Println("Static assets copied successfully.")
	} else {
		fmt.Printf("Static assets directory '%s' not found, skipping copy.\n", staticDir)
	}

	fmt.Printf("Loading layouts from: %s\n", layoutsDir)
	var layoutFiles []string
	err := filepath.WalkDir(layoutsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".html") {
			layoutFiles = append(layoutFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to find layout files in '%s': %w", layoutsDir, err)
	}
	if len(layoutFiles) == 0 {
		return fmt.Errorf("no .html layout files found in '%s'. At least '%s' is expected", layoutsDir, conventionalBaseLayout)
	}

	templates, err := template.ParseFiles(layoutFiles...)
	if err != nil {
		return fmt.Errorf("failed to parse layout files: %w", err)
	}
	fmt.Printf("Successfully parsed %d layout file(s).\n", len(layoutFiles))

	fmt.Printf("Processing content from source directory: '%s'\n", sourceDir)
	err = filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("error accessing path '%s' during walk: %w", path, walkErr)
		}

		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}

		fmt.Printf("Processing file: %s\n", path)

		fileBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("failed to read file '%s': %w", path, readErr)
		}

		var fmData map[string]interface{}
		markdownBodyContent, frontmatterErr := frontmatter.Parse(bytes.NewReader(fileBytes), &fmData)
		if frontmatterErr != nil {
			fmt.Printf("Warning: Could not parse frontmatter for %s (or no frontmatter found): %v. Treating as pure markdown.\n", path, frontmatterErr)
			markdownBodyContent = fileBytes
			fmData = make(map[string]interface{})
		}

		var htmlBuffer bytes.Buffer
		if convertErr := mdParser.Convert(markdownBodyContent, &htmlBuffer); convertErr != nil {
			return fmt.Errorf("failed to convert markdown to HTML for file '%s': %w", path, convertErr)
		}

		pageTitle := ""
		if titleFromFM, ok := fmData["title"].(string); ok && titleFromFM != "" {
			pageTitle = titleFromFM
		} else {
			pageBaseName := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			tempTitle := strings.ReplaceAll(strings.ReplaceAll(pageBaseName, "-", " "), "_", " ")
			titleCaser := cases.Title(language.English)
			pageTitle = titleCaser.String(tempTitle)
		}

		pageDate := ""
		if dateFromFM, ok := fmData["date"].(string); ok {
			pageDate = dateFromFM
		}

		relPath, relErr := filepath.Rel(sourceDir, path)
		if relErr != nil {
			return fmt.Errorf("failed to get relative path for '%s': %w", path, relErr)
		}
		htmlFilename := strings.TrimSuffix(relPath, filepath.Ext(relPath)) + ".html"
		outputPath := filepath.Join(outputDir, htmlFilename)
		outputSubDir := filepath.Dir(outputPath)

		if mkdirErr := os.MkdirAll(outputSubDir, os.ModePerm); mkdirErr != nil {
			return fmt.Errorf("failed to create directory '%s': %w", outputSubDir, mkdirErr)
		}

		pageData := model.PageData{
			SiteTitle: cfg.SiteTitle,
			PageTitle: pageTitle,
			Content:   template.HTML(htmlBuffer.String()),
			BaseURL:   cfg.BaseURL,
			Date:      pageDate,
			Params:    fmData,
		}

		outFile, createErr := os.Create(outputPath)
		if createErr != nil {
			return fmt.Errorf("failed to create output file '%s': %w", outputPath, createErr)
		}
		defer outFile.Close()

		if execErr := templates.ExecuteTemplate(outFile, conventionalBaseLayout, pageData); execErr != nil {
			if strings.Contains(execErr.Error(), "template not defined") {
				return fmt.Errorf("failed to execute template for '%s': base layout '%s' not found or not parsed correctly. Original error: %w", outputPath, conventionalBaseLayout, execErr)
			}
			return fmt.Errorf("failed to execute template for '%s': %w", outputPath, execErr)
		}
		fmt.Printf("Successfully generated: %s\n", outputPath)
		return nil
	})

	if err != nil {
		return fmt.Errorf("error during build process: %w", err)
	}

	fmt.Println("SHIT SSG build completed successfully!")
	return nil
}

// copyDirContents recursively copies contents from src to dst.
// It copies files and directories.
func copyDirContents(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}
		dstPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			// *** THE FIX IS HERE ***
			// Use os.ModePerm for the new directory, not the source directory's permissions.
			// os.ModePerm (0777) will be adjusted by the system's umask (e.g., to 0755).
			if err := os.MkdirAll(dstPath, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dstPath, err)
			}
		} else {
			if err := copyFile(path, dstPath); err != nil {
				return fmt.Errorf("failed to copy file from %s to %s: %w", path, dstPath, err)
			}
		}
		return nil
	})
}

// copyFile copies a single file from srcFile to dstFile.
func copyFile(srcFile, dstFile string) error {
	srcF, err := os.Open(srcFile)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", srcFile, err)
	}
	defer srcF.Close()

	// Ensure the destination directory exists
	dstDir := filepath.Dir(dstFile)
	if err := os.MkdirAll(dstDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", dstDir, err)
	}

	dstF, err := os.Create(dstFile)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dstFile, err)
	}
	defer dstF.Close()

	if _, err := io.Copy(dstF, srcF); err != nil {
		return fmt.Errorf("failed to copy data from %s to %s: %w", srcFile, dstFile, err)
	}

	// Preserve file mode (permissions)
	srcInfo, err := os.Stat(srcFile)
	if err == nil { // If we can get source info
		if err := os.Chmod(dstFile, srcInfo.Mode()); err != nil {
			// Non-fatal, but log it? For now, just continue.
			fmt.Printf("Warning: could not set permissions on %s: %v\n", dstFile, err)
		}
	} else {
		fmt.Printf("Warning: could not stat source file %s to preserve permissions: %v\n", srcFile, err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
