package watcher

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher ...
type Watcher struct {
	FSNotifyWatcher *fsnotify.Watcher
	Directories     map[string]bool
	Excludes        []string
	Done            chan bool
	Events          chan string
}

// NewWatcher ...
func NewWatcher(excludes []string) (*Watcher, error) {
	fsNotifyWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if excludes == nil {
		excludes = []string{}
	}

	result := &Watcher{
		FSNotifyWatcher: fsNotifyWatcher,
		Directories:     map[string]bool{},
		Excludes:        excludes,
		Done:            make(chan bool),
		Events:          make(chan string),
	}
	return result, nil
}

// Add ...
func (w *Watcher) Add(dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			for _, exclude := range w.Excludes {
				if path == exclude || strings.HasPrefix(path, exclude+"/") {
					return nil
				}
			}

			w.FSNotifyWatcher.Add(path)
		}

		return nil
	})
}

// Run ...
func (w *Watcher) Run() {
	ticker := time.Tick(100 * time.Millisecond)
	changes := map[string]bool{}

out:
	for {
		select {
		case event := <-w.FSNotifyWatcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				changes[event.Name] = true
			} else if event.Op&fsnotify.Create == fsnotify.Create {
				if info, err := os.Stat(event.Name); err == nil {
					if info.IsDir() {
						w.Add(event.Name)
					} else {
						changes[event.Name] = true
					}
				}
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				w.FSNotifyWatcher.Remove(event.Name)
				changes[event.Name] = true
			} else if event.Op&fsnotify.Rename == fsnotify.Rename {
				changes[event.Name] = true
			}
		case <-ticker:
			for filename := range changes {
				w.Events <- filename
			}
			changes = map[string]bool{}
		case <-w.Done:
			break out
		}
	}

	close(w.Done)
	close(w.Events)
}

// Close ...
func (w *Watcher) Close() {
	w.Done <- true
	w.FSNotifyWatcher.Close()
}
