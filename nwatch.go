///usr/bin/env true; exec /usr/bin/env go run "$0" "$@"
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tzvetkoff-go/optparse"

	"github.com/tzvetkoff-go/fnmatch"
	"github.com/tzvetkoff-go/nwatch/pkg/watcher"
)

// usage ...
func usage(f io.Writer, name string) {
	fmt.Fprintln(f, "Usage:")
	fmt.Fprintf(f, "  %s [options]\n", name)
	fmt.Fprintln(f)

	fmt.Fprintln(f, "Options:")
	fmt.Fprintln(f, "  -h, --help                        Print help and exit")
	fmt.Fprintln(f, "  -v, --version                     Print version and exit")
	fmt.Fprintln(f, "  -V, --verbose                     Verbose output")
	fmt.Fprintln(f, "  -d DIR, --directory=DIR           Directories to watch")
	fmt.Fprintln(f, "  -e EXC, --exclude-dir=EXC         Directories to exclude")
	fmt.Fprintln(f, "  -p PAT, --pattern=PAT             File patterns to match")
	fmt.Fprintln(f, "  -i IGN, --ignore=IGN              File patterns to ignore")
	fmt.Fprintln(f, "  -b BLD, --build=BLD               Build command to execute")
	fmt.Fprintln(f, "  -s SRV, --server=SRV              Server command to run after successful build")
	fmt.Fprintln(f, "  -w ERR, --error-server=ERR        Web server address in case of an error")

	if f == os.Stderr {
		os.Exit(1)
	}

	os.Exit(0)
}

// mu ...
var mu sync.Mutex

// lastBuildOutput ...
var lastBuildOutput []byte

// runBuild ...
func runBuild(command string) error {
	mu.Lock()
	defer mu.Unlock()

	verbose("runBuild: %s", command)

	cmd := exec.Command("/bin/sh", "-c", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		lastBuildOutput = out
		fmt.Fprintln(os.Stderr, "-------- build error: --------")
		fmt.Fprintln(os.Stderr, strings.TrimSpace(string(out)))
		fmt.Fprintln(os.Stderr, "------------------------------")
	}

	return err
}

// serverCmd ...
var serverCmd *exec.Cmd

// runServer ...
func runServer(command string) {
	mu.Lock()
	defer mu.Unlock()

	verbose("runServer: %s", command)

	if serverCmd != nil && serverCmd.Process != nil {
		_ = serverCmd.Process.Kill()
		serverCmd.Process.Wait()
		serverCmd = nil
	}

	if errorServerChan != nil {
		errorServerChan <- true
		if errorServerChan != nil {
			close(errorServerChan)
		}
		errorServerChan = nil
	}

	serverCmd = exec.Command("/bin/sh", "-c", command)
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr
	err := serverCmd.Start()
	if err != nil {
		serverCmd = nil
		fmt.Fprintln(os.Stderr, "-------- server error: --------")
		fmt.Fprintln(os.Stderr, strings.TrimSpace(err.Error()))
		fmt.Fprintln(os.Stderr, "-------------------------------")
	}
}

// errorServerChan ...
var errorServerChan chan bool

// runErrorServer ...
func runErrorServer(address string) {
	mu.Lock()
	defer mu.Unlock()

	verbose("runErrorServer: %s", address)

	if serverCmd != nil && serverCmd.Process != nil {
		_ = serverCmd.Process.Kill()
		serverCmd.Process.Wait()
		serverCmd = nil
	}

	if errorServerChan != nil {
		errorServerChan <- true
		if errorServerChan != nil {
			close(errorServerChan)
		}
		errorServerChan = nil
	}

	s := &http.Server{
		Addr: address,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "text/plain")
			w.Write(lastBuildOutput)
		}),
	}
	errorServerChan = make(chan bool, 1)
	go func() {
		<-errorServerChan
		if errorServerChan != nil {
			close(errorServerChan)
		}
		errorServerChan = nil
		s.Shutdown(context.TODO())
	}()
	go func() {
		e := s.ListenAndServe()
		fmt.Println(e)
	}()
}

// matchAny ...
func matchAny(s string, patterns []string) bool {
	for _, pattern := range patterns {
		if fnmatch.Match(pattern, s) {
			return true
		}
	}

	return false
}

// pVerbose ...
var pVerbose = optparse.Bool("verbose", 'V', false)

// verbose ...
func verbose(format string, args ...interface{}) {
	if *pVerbose {
		fmt.Fprintf(os.Stderr, ">> "+format+"\n", args...)
	}
}

// main ...
func main() {
	// Options
	pHelp := optparse.Bool("help", 'h', false)
	pVersion := optparse.Bool("version", 'v', false)
	pDirectories := optparse.StringList("directory", 'd')
	pExcludes := optparse.StringList("exclude", 'e')
	pPatterns := optparse.StringList("pattern", 'p')
	pIgnores := optparse.StringList("ignore", 'i')
	pBuild := optparse.String("build", 'b', "")
	pServer := optparse.String("server", 's', "")
	pErrorServer := optparse.String("error-server", 'w', "")

	// Parse
	args, err := optparse.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n\n", os.Args[0], err.Error())
		usage(os.Stderr, os.Args[0])
	}

	// We don't accept positional arguments
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "%s: wrong number of arguments (given %d, expected %d)\n\n", os.Args[0], len(args), 0)
		usage(os.Stderr, os.Args[0])
	}

	// Help
	if *pHelp {
		usage(os.Stdout, os.Args[0])
	}

	// Version
	if *pVersion {
		fmt.Println("0.1.0")
		os.Exit(0)
	}

	// Default to current directory
	if len(*pDirectories) == 0 {
		*pDirectories = []string{"."}
	}

	// Default to *
	if len(*pPatterns) == 0 {
		*pPatterns = []string{"*"}
	}

	// Build command is required
	if *pBuild == "" {
		fmt.Fprintf(os.Stderr, "%s: --build is required\n\n", os.Args[0])
		usage(os.Stderr, os.Args[0])
	}

	// Create watcher
	watcher, err := watcher.NewWatcher(*pExcludes)
	if err != nil {
		panic(err)
	}

	// Watch in the background
	go func() {
		ticker := time.Tick(100 * time.Millisecond)
		hasChanges := false

		for {
			select {
			case evt := <-watcher.Events:
				verbose("evt: %s", evt)

				basename := path.Base(evt)

				if !matchAny(basename, *pPatterns) {
					verbose("no-match: %q does not match any of %q", basename, *pPatterns)
					continue
				}
				if matchAny(basename, *pIgnores) {
					verbose("match-ignore: %q matches at least one of %q", basename, *pIgnores)
					continue
				}

				hasChanges = true
			case <-ticker:
				if hasChanges {
					hasChanges = false

					if runBuild(*pBuild) == nil && *pServer != "" {
						runServer(*pServer)
					} else if *pErrorServer != "" {
						runErrorServer(*pErrorServer)
					}
				}
			}
		}
	}()

	// Exit strategy
	ch := make(chan os.Signal)
	go func() {
		<-ch
		fmt.Println("\nAborting...")
		watcher.Close()
	}()
	signal.Notify(ch, syscall.SIGINT)
	signal.Notify(ch, syscall.SIGTERM)

	// Run initial build if we have a server command passed
	if *pServer != "" && runBuild(*pBuild) == nil {
		runServer(*pServer)
	} else if *pErrorServer != "" {
		runErrorServer(*pErrorServer)
	}

	// Add directories to watch
	for _, directory := range *pDirectories {
		watcher.Add(directory)
	}

	// Finally, watch
	watcher.Run()
}
