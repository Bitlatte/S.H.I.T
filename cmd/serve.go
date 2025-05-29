// cmd/serve.go
package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time" // For potential debouncing in the future

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var serverPort int // For the --port flag

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serves the site locally and watches for changes",
	Long: `The serve command performs an initial build of your site, then starts a local
web server to serve your output directory. It also watches your content, layouts,
and static directories for changes and automatically rebuilds the site.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initial build
		log.Println("Performing initial build...")
		// siteData is the package-level variable from cmd/root.go
		if err := runBuildProcess(appConfig, siteData); err != nil {
			// Log fatal if initial build fails, as there's nothing to serve.
			log.Fatalf("Initial build failed: %v. Please fix issues and try again.", err)
			return err // Should be unreachable due to log.Fatalf
		}
		log.Println("Initial build successful.")

		// Setup fsnotify watcher
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatalf("Failed to create file watcher: %v", err)
			return err
		}
		defer watcher.Close()

		// Goroutine for watching file changes
		go func() {
			// Simple debouncing: wait a short period after an event before rebuilding
			var buildTimer *time.Timer
			debounceDuration := 500 * time.Millisecond // 500ms debounce

			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return // Channel closed
					}
					// We're interested in events that change file content or structure
					if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
						log.Printf("Change detected: %s (%s)", event.Name, event.Op.String())

						// If a directory we were explicitly watching is removed, fsnotify might stop watching it.
						// Or if a new directory is added within a watched path, its children aren't automatically watched.
						// For simplicity, on relevant events, re-evaluate and re-add watches.
						// This is a bit heavy-handed but more robust for catching new subdirectories.
						if event.Has(fsnotify.Create) && isDir(event.Name) {
							log.Printf("New directory created: %s. Adding to watcher.", event.Name)
							if err := watcher.Add(event.Name); err != nil {
								log.Printf("Error adding new directory %s to watcher: %v", event.Name, err)
							}
						}

						// Debounce rebuilding
						if buildTimer != nil {
							buildTimer.Stop()
						}
						buildTimer = time.AfterFunc(debounceDuration, func() {
							log.Println("Rebuilding site due to changes...")
							// siteData is the package-level variable from cmd/root.go
							if err := runBuildProcess(appConfig, siteData); err != nil {
								log.Printf("Error during rebuild: %v", err)
							} else {
								log.Println("Site rebuilt successfully.")
							}
						})
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return // Channel closed
					}
					log.Printf("Watcher error: %v", err)
				}
			}
		}()

		// Add paths to watcher
		// Note: fsnotify watches specific paths. For recursive watching, add each directory.
		pathsToWatch := []string{
			conventionalContentDir,
			conventionalLayoutsDir,
			conventionalStaticDir,
		}

		for _, rootPath := range pathsToWatch {
			if _, statErr := os.Stat(rootPath); os.IsNotExist(statErr) {
				log.Printf("Directory '%s' not found, not watching.", rootPath)
				continue
			}

			log.Printf("Setting up watch for %s and its subdirectories...", rootPath)
			err = filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					// Log error but continue trying to watch other paths
					log.Printf("Error walking %s: %v", path, err)
					return nil
				}
				if d.IsDir() {
					if watchErr := watcher.Add(path); watchErr != nil {
						log.Printf("Failed to watch %s: %v", path, watchErr)
					}
				}
				return nil
			})
			if err != nil {
				log.Printf("Error during initial directory walk for watching %s: %v", rootPath, err)
			}
		}

		// Start the HTTP server
		serverAddr := fmt.Sprintf(":%d", serverPort)
		log.Printf("Serving site from '%s' on http://localhost%s", appConfig.OutputDir, serverAddr)
		log.Println("Press Ctrl+C to stop the server.")

		fs := http.FileServer(http.Dir(appConfig.OutputDir))
		// Use a custom handler to prevent directory listing and set no-cache headers
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Prevent directory listing
			if strings.HasSuffix(r.URL.Path, "/") && r.URL.Path != "/" {
				// Check if an index.html exists in the directory
				_, err := os.Stat(filepath.Join(appConfig.OutputDir, r.URL.Path, "index.html"))
				if os.IsNotExist(err) {
					http.NotFound(w, r)
					return
				}
			}
			// Set headers to prevent caching during development
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			fs.ServeHTTP(w, r)
		})

		if err := http.ListenAndServe(serverAddr, nil); err != nil {
			// This will be logged if the server fails to start (e.g., port already in use)
			log.Fatalf("Failed to start HTTP server: %v", err)
			return err // Should be unreachable
		}
		return nil // Should not be reached
	},
}

// Helper function to check if a path is a directory
func isDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}

func init() {
	serveCmd.Flags().IntVarP(&serverPort, "port", "p", 1313, "Port to serve the site on")
	rootCmd.AddCommand(serveCmd)
}
