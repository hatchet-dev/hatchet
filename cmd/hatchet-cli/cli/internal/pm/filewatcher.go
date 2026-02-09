package pm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/patternmatcher"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

func WatchFiles(ctx context.Context, patterns []string, pm *ProcessManager) <-chan error {
	reloadNotifier := make(chan struct{}, 1)

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		cli.Logger.Fatalf("error creating file watcher: %v", err)
	}

	cwd, err := os.Getwd()

	if err != nil {
		cli.Logger.Fatalf("error getting cwd: %v", err)
	}

	// Create pattern matcher for file watching
	for _, pattern := range patterns {
		fmt.Println(styles.Muted.Render(fmt.Sprintf("  Watching pattern: %s", pattern)))
	}

	patternM, err := patternmatcher.New(patterns)
	if err != nil {
		cli.Logger.Fatalf("error parsing file patterns: %v", err)
	}

	// Track directories to watch for new files
	watchedDirs := make(map[string]bool)

	// Walk the directory tree and watch matching files and all directories
	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Watch all directories so we can detect new files
		if info.IsDir() {
			// Make path relative to cwd for pattern matching
			relPath, err := filepath.Rel(cwd, path)
			if err != nil {
				relPath = path
			}

			// Watch files that match any pattern (and aren't excluded)
			matched, err := patternM.DirMatches(relPath)

			if err != nil {
				return err
			}

			if !matched {
				return filepath.SkipDir
			}

			err = watcher.Add(path)

			if err != nil {
				return err
			}

			watchedDirs[path] = true
			return nil
		}

		// Make path relative to cwd for pattern matching
		relPath, err := filepath.Rel(cwd, path)
		if err != nil {
			relPath = path
		}

		// Watch files that match any pattern (and aren't excluded)
		matched, err := patternM.MatchesOrParentMatches(relPath)
		if err == nil && matched {
			err = watcher.Add(path)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		cli.Logger.Fatalf("error walking path: %v", err)
	}

	// TODO: remove debug prints
	for _, file := range watcher.WatchList() {
		fmt.Println(styles.Muted.Render(fmt.Sprintf("  Watching file: %s", file)))
	}

	watchCount := len(watcher.WatchList())
	fmt.Println(styles.Muted.Render(fmt.Sprintf("  Watching %d file(s) and directories", watchCount)))

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Check if this is a write to an existing file or a newly created file
				shouldReload := false

				// Make path relative to cwd for pattern matching
				relPath, err := filepath.Rel(cwd, event.Name)
				if err != nil {
					relPath = event.Name
				}

				if event.Has(fsnotify.Write) {
					// Check if it's a file we're already watching (matches a pattern)
					matched, err := patternM.MatchesOrParentMatches(relPath)
					if err == nil && matched {
						shouldReload = true
					}
				} else if event.Has(fsnotify.Create) {
					// New file or directory created
					info, err := os.Stat(event.Name)
					if err == nil {
						if info.IsDir() {
							// determine if the directory matches
							dirRelPath, err := filepath.Rel(cwd, event.Name)
							if err != nil {
								dirRelPath = event.Name
							}

							dirMatched, err := patternM.DirMatches(dirRelPath)
							if err == nil && dirMatched {
								// Watch new directory, but don't reload
								err = watcher.Add(event.Name)
								if err != nil {
									cli.Logger.Warnf("could not add new directory to filewatcher: %s", err.Error())
								}
								watchedDirs[event.Name] = true
							}
						} else {
							// New file created - check if it matches any pattern
							matched, err := patternM.MatchesOrParentMatches(relPath)
							if err == nil && matched {
								err = watcher.Add(event.Name)
								if err != nil {
									cli.Logger.Warnf("could not add new file to filewatcher: %s", err.Error())
								}
								shouldReload = true
							}
						}
					}
				}

				if shouldReload {
					fmt.Println(styles.InfoMessage(fmt.Sprintf("File change detected: %s", event.Name)))

					// non-blocking write to reloadNotifier
					select {
					case reloadNotifier <- struct{}{}:
					default:
					}
				}
			case <-reloadNotifier:
				fmt.Println(styles.InfoMessage("Reloading worker..."))

				err = pm.StartProcess(ctx)

				if err != nil {
					cli.Logger.Fatalf("error restarting worker: %v", err)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("Watcher error: %v\n", err)
			}
		}
	}()

	err = pm.StartProcess(ctx)
	if err != nil {
		cli.Logger.Fatalf("error starting worker: %v", err)
	}

	cleanup := make(chan error)

	go func() {
		<-ctx.Done()
		pm.KillProcess()
		watcher.Close()

		cleanup <- nil
	}()

	return cleanup

}
