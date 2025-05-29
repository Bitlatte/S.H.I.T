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
	"sort" // Added import
	"strings"
	"time" // Added import for time package

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
		// siteData is the package-level variable from cmd/root.go
		return runBuildProcess(appConfig, siteData)
	},
}

func runBuildProcess(cfg config.Config, site *model.SiteData) error { // Modified signature
	fmt.Println("Starting SHIT SSG build process...")
	fmt.Printf("Using OutputDir: '%s', BaseURL: '%s', SiteTitle: '%s'\n", cfg.OutputDir, cfg.BaseURL, cfg.SiteTitle)

	// Ensure ContentItems is initialized
	if site.ContentItems == nil {
		site.ContentItems = []*model.ContentItem{}
	}

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
		// This might be fine if we are not rendering HTML files directly in this phase
		fmt.Printf("Warning: no .html layout files found in '%s'. This might be an issue if rendering is expected.\n", layoutsDir)
	}

	// templates, err := template.ParseFiles(layoutFiles...) // We might not need to parse all templates here if only collecting content
	// if err != nil {
	// 	return fmt.Errorf("failed to parse layout files: %w", err)
	// }
	// fmt.Printf("Successfully parsed %d layout file(s).\n", len(layoutFiles))

	// Parse all template files from the layouts directory.
	// templates, err := template.ParseFiles(layoutFiles...) // Old way
	// if err != nil {
	// 	return fmt.Errorf("failed to parse layout files: %w", err)
	// }

	// New parsing strategy:
	var baseHTMLPath string
	var otherLayoutFiles []string
	var partialLayoutFiles []string // For files in partials directory

	for _, f := range layoutFiles {
		if filepath.Base(f) == "base.html" && filepath.Dir(f) == layoutsDir { // Ensure it's the root base.html
			baseHTMLPath = f
		} else if strings.HasPrefix(filepath.Dir(f), filepath.Join(layoutsDir, "partials")) {
			partialLayoutFiles = append(partialLayoutFiles, f)
		} else {
			otherLayoutFiles = append(otherLayoutFiles, f)
		}
	}

	if baseHTMLPath == "" {
		return fmt.Errorf("base.html not found directly in layouts directory '%s'", layoutsDir)
	}

	// 1. Parse base.html first.
	// Also include all partials with base.html as they are often globally needed.
	filesToParseInitially := append([]string{baseHTMLPath}, partialLayoutFiles...)
	templates, err := template.ParseFiles(filesToParseInitially...)
	if err != nil {
		return fmt.Errorf("failed to parse base.html and partials: %w", err)
	}

	// 2. Then, parse the other layout files (single, list-posts, etc., EXCLUDING home.html for now)
	var finalOtherLayoutFiles []string
	var homeHTMLPathForFinalParse string
	for _, f := range otherLayoutFiles {
		if filepath.Base(f) == "home.html" && filepath.Dir(f) == layoutsDir {
			homeHTMLPathForFinalParse = f
		} else {
			finalOtherLayoutFiles = append(finalOtherLayoutFiles, f)
		}
	}

	if len(finalOtherLayoutFiles) > 0 {
		templates, err = templates.ParseFiles(finalOtherLayoutFiles...)
		if err != nil {
			return fmt.Errorf("failed to parse other page layout files (excluding home.html): %w", err)
		}
	}

	// 3. Parse home.html last (if found)
	if homeHTMLPathForFinalParse != "" {
		templates, err = templates.ParseFiles(homeHTMLPathForFinalParse)
		if err != nil {
			return fmt.Errorf("failed to parse home.html: %w", err)
		}
		fmt.Println("Parsed home.html specifically.")
	}

	fmt.Printf("Successfully parsed %d layout file(s) with new strategy.\n", len(layoutFiles))


	fmt.Printf("Processing content from source directory: '%s'\n", sourceDir)
	walkErr := filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("error accessing path '%s' during walk: %w", path, walkErr)
		}

		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}

		fmt.Printf("Processing file: %s\n", path)

		fileBytes, readErr := os.ReadFile(path)
		if readErr != nil {
			// Log error and continue with next file? Or fail build? For now, fail.
			return fmt.Errorf("failed to read file '%s': %w", path, readErr)
		}

		var fmData map[string]interface{} // Frontmatter data
		markdownBodyContent, frontmatterErr := frontmatter.Parse(bytes.NewReader(fileBytes), &fmData)
		if frontmatterErr != nil {
			fmt.Printf("Warning: Could not parse frontmatter for %s (or no frontmatter found): %v. Treating as pure markdown.\n", path, frontmatterErr)
			markdownBodyContent = fileBytes // Use full content if no frontmatter
			fmData = make(map[string]interface{}) // Ensure fmData is not nil
		}

		var htmlBuffer bytes.Buffer
		if convertErr := mdParser.Convert(markdownBodyContent, &htmlBuffer); convertErr != nil {
			return fmt.Errorf("failed to convert markdown to HTML for file '%s': %w", path, convertErr)
		}

		// Determine Page Title
		pageTitle := ""
		if titleFromFM, ok := fmData["title"].(string); ok && titleFromFM != "" {
			pageTitle = titleFromFM
		} else {
			pageBaseName := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			tempTitle := strings.ReplaceAll(strings.ReplaceAll(pageBaseName, "-", " "), "_", " ")
			titleCaser := cases.Title(language.English)
			pageTitle = titleCaser.String(tempTitle)
		}

		// Determine Content Type
		relPath, _ := filepath.Rel(sourceDir, path)
		dir := filepath.Dir(relPath)
		parts := strings.Split(dir, string(filepath.Separator))
		itemType := "page" // Default
		if len(parts) > 0 && parts[0] != "." && parts[0] != "" {
			itemType = parts[0]
		}
		if fmType, ok := fmData["type"].(string); ok && fmType != "" { // Frontmatter overrides directory
			itemType = fmType
		}

		// Parse Date
		var itemDate time.Time
		if dateStr, ok := fmData["date"].(string); ok {
			// Try common date formats
			formats := []string{"2006-01-02T15:04:05Z07:00", "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"}
			parsed := false
			for _, format := range formats {
				parsedDate, err := time.Parse(format, dateStr)
				if err == nil {
					itemDate = parsedDate
					parsed = true
					break
				}
			}
			if !parsed {
				fmt.Printf("Warning: Could not parse date string '%s' for %s with any common format. Please use YYYY-MM-DD or RFC3339 format.\n", dateStr, path)
			}
		}

		// Determine Permalink
		// relPath already calculated above
		itemPermalink := "/" + strings.TrimSuffix(relPath, filepath.Ext(relPath)) + "/"
		itemPermalink = strings.Replace(itemPermalink, ".md/", "/", 1) // Clean for .md extensions if present
		itemPermalink = filepath.Clean(itemPermalink)                 // General path cleaning
		if !strings.HasPrefix(itemPermalink, "/") { // Ensure leading slash
			itemPermalink = "/" + itemPermalink
		}
		if !strings.HasSuffix(itemPermalink, "/") { // Ensure trailing slash for directory-like URLs
			itemPermalink += "/"
		}


		// Get Summary
		itemSummary := ""
		if summary, ok := fmData["summary"].(string); ok {
			itemSummary = summary
		}

		// Get Layout
		itemLayout := ""
		if layout, ok := fmData["layout"].(string); ok { // Check if layout is specified in frontmatter
			itemLayout = layout
		}

		// Construct ContentItem
		contentItem := &model.ContentItem{
			Title:       pageTitle,
			Date:        itemDate,
			Type:        itemType,
			SourcePath:  path,
			Permalink:   itemPermalink,
			ContentHTML: template.HTML(htmlBuffer.String()),
			Frontmatter: fmData,
			Summary:     itemSummary,
			Layout:      itemLayout,
		}

		// Store ContentItem
		site.ContentItems = append(site.ContentItems, contentItem)

		// Old file writing logic is now removed from here.
		// Generation will happen in a new loop after all content is processed.
		return nil
	})

	if walkErr != nil { // Changed from err to walkErr to avoid conflict
		return fmt.Errorf("error during content collection walk: %w", walkErr)
	}

	// Debugging/verification for collected items
	fmt.Printf("Collected %d content items:\n", len(site.ContentItems))
	for _, item := range site.ContentItems {
		fmt.Printf("  - Title: %s, Type: %s, Path: %s, Permalink: %s, Date: %s, Layout: %s\n", item.Title, item.Type, item.SourcePath, item.Permalink, item.Date.Format("2006-01-02"), item.Layout)
	}

	// Sort ContentItems by Date (descending)
	sort.Slice(site.ContentItems, func(i, j int) bool {
		if site.ContentItems[i].Date.IsZero() {
			return false // i is "greater" (comes after) if j is not zero or i is zero
		}
		if site.ContentItems[j].Date.IsZero() {
			return true // j is "greater" (comes after) if i is not zero or j is zero
		}
		return site.ContentItems[i].Date.After(site.ContentItems[j].Date)
	})
	fmt.Println("Content items sorted by date (descending).")

	// Initialize and populate filtered slices and map
	site.Posts = []*model.ContentItem{}
	site.Projects = []*model.ContentItem{}
	site.ContentByType = make(map[string][]*model.ContentItem)

	for _, item := range site.ContentItems {
		// Populate ContentByType
		site.ContentByType[item.Type] = append(site.ContentByType[item.Type], item)

		// Populate Posts
		if item.Type == "posts" { // Corrected: "posts" (plural) to match directory-derived type
			site.Posts = append(site.Posts, item)
		}

		// Populate Projects
		if item.Type == "project" { // Correct: "project" (singular) matches explicit frontmatter type
			site.Projects = append(site.Projects, item)
		}
	}

	// Debugging/verification for filtered collections
	fmt.Printf("Found %d posts, %d projects.\n", len(site.Posts), len(site.Projects))
	for contentType, items := range site.ContentByType {
		fmt.Printf("Type '%s': %d items\n", contentType, len(items))
	}
	fmt.Println("Content collection, sorting, and filtering completed.")

	// ** Step 2: Page Generation Loop **
	fmt.Println("Starting page generation...")
	defaultSingleLayout := "single.html" // Default layout for single items

	for _, item := range site.ContentItems {
		layoutToExecute := defaultSingleLayout // Start with default

		// Check for single-post.html for "post" type
		if item.Type == "post" {
			if templates.Lookup("single-post.html") != nil {
				layoutToExecute = "single-post.html"
			} else {
				fmt.Printf("Warning: Layout 'single-post.html' not found for post '%s', using '%s'\n", item.Title, layoutToExecute)
			}
		}

		// Check for layout specified in frontmatter
		if item.Layout != "" {
			if templates.Lookup(item.Layout) != nil {
				layoutToExecute = item.Layout
			} else {
				fmt.Printf("Warning: Frontmatter layout '%s' for item '%s' not found, using '%s'\n", item.Layout, item.Title, layoutToExecute)
			}
		}
		
		// Final check: if the determined layoutToExecute is not found, fall back or error
		if templates.Lookup(layoutToExecute) == nil {
			fmt.Printf("Warning: Layout '%s' for item '%s' not found. Attempting to use conventional base layout '%s'.\n", layoutToExecute, item.Title, conventionalBaseLayout)
			layoutToExecute = conventionalBaseLayout
			if templates.Lookup(layoutToExecute) == nil {
				// This is a critical error, means even base.html is missing or not parsed.
				return fmt.Errorf("critical error: Neither layout '%s' nor conventional base layout '%s' could be found for item '%s'. Halting build.", item.Layout, conventionalBaseLayout, item.Title)
			}
		}


		// Prepare Output Path
		// outputDir is cfg.OutputDir
		outputPath := filepath.Join(outputDir, item.Permalink, "index.html")
		outputSubDir := filepath.Dir(outputPath)

		if err := os.MkdirAll(outputSubDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory '%s' for item '%s': %w", outputSubDir, item.Title, err)
		}

		// Prepare Template Data Context
		dataForTemplate := struct {
			Site *model.SiteData
			Item *model.ContentItem
		}{
			Site: site,
			Item: item,
		}

		// Create Output File
		outFile, createErr := os.Create(outputPath)
		if createErr != nil {
			return fmt.Errorf("failed to create output file '%s' for item '%s': %w", outputPath, item.Title, createErr)
		}
		defer outFile.Close() // Ensure file is closed

		// Execute Template
		// The 'name' in ExecuteTemplate is the template definition name.
		// For files parsed with ParseFiles, this is typically the filename.
		if err := templates.ExecuteTemplate(outFile, layoutToExecute, dataForTemplate); err != nil {
			return fmt.Errorf("failed to execute template '%s' for item '%s' (outputting to '%s'): %w", layoutToExecute, item.Title, outputPath, err)
		}
		fmt.Printf("Successfully generated: %s using layout %s\n", outputPath, layoutToExecute)
	}

	// ** Step 3: Generate Homepage **
	fmt.Println("Generating homepage...")
	homeLayoutName := "home.html"
	homeTpl := templates.Lookup(homeLayoutName)
	if homeTpl == nil {
		// If home.html is optional, this could be a warning or log instead of an error.
		// For now, assume it's required if present in the logic.
		return fmt.Errorf("homepage layout '%s' not found. Please create it in the layouts directory", homeLayoutName)
	}

	homeOutputPath := filepath.Join(outputDir, "index.html") // outputDir is cfg.OutputDir

	dataForHomepage := struct {
		Site *model.SiteData
	}{
		Site: site,
	}

	homeOutFile, err := os.Create(homeOutputPath)
	if err != nil {
		return fmt.Errorf("failed to create homepage file '%s': %w", homeOutputPath, err)
	}
	defer homeOutFile.Close()

	err = templates.ExecuteTemplate(homeOutFile, homeLayoutName, dataForHomepage)
	if err != nil {
		return fmt.Errorf("failed to execute homepage template '%s': %w", homeLayoutName, err)
	}
	fmt.Printf("Successfully generated homepage: %s using layout %s\n", homeOutputPath, homeLayoutName)

	// ** Step 4: Generate Posts Listing Page **
	fmt.Println("Generating posts list page...")
	postListLayoutName := "list-posts.html"
	postListTpl := templates.Lookup(postListLayoutName)

	if postListTpl == nil {
		fmt.Printf("Warning: Post list layout '%s' not found. Skipping generation of post list page.\n", postListLayoutName)
	} else {
		postListOutputDir := filepath.Join(outputDir, "posts") // outputDir is cfg.OutputDir
		if err := os.MkdirAll(postListOutputDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory '%s' for posts list page: %w", postListOutputDir, err)
		}
		postListOutputPath := filepath.Join(postListOutputDir, "index.html")

		dataForPostList := struct {
			Site *model.SiteData
		}{
			Site: site,
		}

		postListOutFile, err := os.Create(postListOutputPath)
		if err != nil {
			return fmt.Errorf("failed to create post list page file '%s': %w", postListOutputPath, err)
		}
		defer postListOutFile.Close()

		err = templates.ExecuteTemplate(postListOutFile, postListLayoutName, dataForPostList)
		if err != nil {
			return fmt.Errorf("failed to execute post list page template '%s': %w", postListLayoutName, err)
		}
		fmt.Printf("Successfully generated post list page: %s using layout %s\n", postListOutputPath, postListLayoutName)
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
