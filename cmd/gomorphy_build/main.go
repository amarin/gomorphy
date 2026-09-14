// gomorphy_build — build compiled dictionaries from various sources.
//
// Usage:
//   gomorphy_build update [-l]   download + unpack + compile OpenCorpora
//   gomorphy_build compile [-d]  compile dict.xml (existing) to GMOR
//
// Flags:
//   -l   use local file only, skip downloading
//   -d   debug logging
//   -o   output path (default: <domain>/opencorpora.dat)

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/amarin/logging"

	"github.com/amarin/gomorphy/pkg/common"
	"github.com/amarin/gomorphy/pkg/morphology"
	"github.com/amarin/gomorphy/pkg/opencorpora"
)

const programVersion = "0.1.0"

var (
	debug     = flag.Bool("d", false, "enable debug logging")
	skipDL    = flag.Bool("l", false, "use local file only, skip downloading")
	output    = flag.String("o", "", "output .dat path (default: auto)")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `gomorphy_build — build GMOR dictionaries from various sources.

Usage:
  gomorphy_build <command> [flags]

Commands:
  update    Download, unpack and compile OpenCorpora dictionary (full cycle)
  compile   Compile existing dict.xml to GMOR format

Flags:
  -d          debug logging
  -l          skip downloading (local only)
  -o <path>   output file path (default depends on command)

Examples:
  gomorphy_build update          # download + unpack + compile
  gomorphy_build update -l       # skip download, only compile
  gomorphy_build compile         # compile existing dict.xml
  gomorphy_build compile -o /tmp/oc.dat   # custom output
`)
	}

	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := flag.Arg(0)

	switch command {
	case "update":
		if err := runUpdate(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "compile":
		if err := runCompile(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n\n", command)
		flag.Usage()
		os.Exit(1)
	}
}

func initLogging() {
	opts := []logging.Option{
		logging.WithFormat(logging.FormatText),
		logging.WithTarget(logging.StdErr),
	}
	if *debug {
		opts = append(opts, logging.WithLevel(logging.LevelDebug))
	} else {
		opts = append(opts, logging.WithLevel(logging.LevelInfo))
	}
	if err := logging.Init(opts...); err != nil {
		fmt.Fprintf(os.Stderr, "logging: init: %v\n", err)
		os.Exit(1)
	}
}

func runUpdate() error {
	initLogging()
	started := time.Now()

	fmt.Println("=== gomorphy_build update ===")

	dataPath := common.DomainDataPath(opencorpora.DomainName)
	loader := opencorpora.NewLoader(dataPath)

	// Phase 1: download + unpack.
	fmt.Println("\n[1/3] Download / unpack...")
	if err := loader.Sync(*skipDL); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	xmlPath := loader.UnpackedFilePath()
	datPath := resolveOutputPath(dataPath)

	// Phase 2: compile + save.
	fmt.Printf("\n[2/3] Compile %s → %s (GMOR)\n", xmlPath, datPath)
	if err := compileAndSave(xmlPath, datPath, started); err != nil {
		return err
	}

	logDone(started, "dictionary ready")
	return nil
}

func runCompile() error {
	initLogging()
	started := time.Now()

	fmt.Println("=== gomorphy_build compile ===")

	xmlPath := findDictXML()
	datPath := resolveOutputPath(opencorpora.DomainName)

	fmt.Printf("\n[1/2] Compile %s → %s (GMOR)\n", xmlPath, datPath)
	if err := compileAndSave(xmlPath, datPath, started); err != nil {
		return err
	}

	logDone(started, "dictionary ready")
	return nil
}

// findDictXML looks for dict.xml in the opencorpora data directory.
func findDictXML() string {
	dataPath := common.DomainDataPath(opencorpora.DomainName)
	xmlPath := filepath.Join(dataPath, opencorpora.LocalUnpackedFilename)
	if _, err := os.Stat(xmlPath); err == nil {
		return xmlPath
	}
	// Fallback: try current directory.
	xmlPath = "dict.xml"
	if _, err := os.Stat(xmlPath); err == nil {
		return xmlPath
	}
	fmt.Fprintf(os.Stderr, "error: dict.xml not found (tried %s and .)\n", dataPath)
	os.Exit(1)
	return ""
}

// resolveOutputPath returns the output .dat path.
func resolveOutputPath(dataPath string) string {
	if *output != "" {
		return *output
	}
	return filepath.Join(dataPath, "opencorpora.dat")
}

// compileAndSave runs the GMOR compilation with a background progress printer.
// The progress callback updates atomic values; a goroutine prints every 10s.
func compileAndSave(xmlPath, datPath string, started time.Time) error {
	var (
		processed      int64
		total          int64
		phase          int64 // 0=insert, 1=compile
	)

	progress := func(a, b int) {
		atomic.StoreInt64(&processed, int64(a))
		atomic.StoreInt64(&total, int64(b))
		if b > 0 && int64(a) >= int64(b)-100 {
			atomic.StoreInt64(&phase, 1)
		}
	}

	// Start background printer goroutine.
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		lastProcessed := int64(0)
		lastReport := time.Now()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				elapsed := time.Since(started)
				elapsedRounded := elapsed.Round(time.Second)
				p := atomic.LoadInt64(&processed)
				t := atomic.LoadInt64(&total)
				currentPhase := atomic.LoadInt64(&phase)

				rate := float64(0)
				if p != lastProcessed {
					dt := time.Since(lastReport).Seconds()
					if dt > 0 {
						rate = float64(p-lastProcessed) / dt
					}
				}
				lastProcessed = p
				lastReport = time.Now()

				var msg string
				var eta string
				if t > 0 {
					pct := int(p * 100 / t)
					remainingKeys := t - p
					var remaining time.Duration
					if rate > 0 && remainingKeys > 0 {
						remaining = time.Duration(float64(remainingKeys)/rate) * time.Second
					}

					if currentPhase == 0 {
						msg = fmt.Sprintf("  [build] %d/%d (%d%%), %d keys/s, elapsed %s",
							p, t, pct, int64(rate), elapsedRounded)
					} else {
						msg = fmt.Sprintf("  [compile] %d/%d (%d%%), %d nodes/s, elapsed %s",
							p, t, pct, int64(rate), elapsedRounded)
					}

					if remaining > 0 && remaining < elapsed*3 {
						eta = fmt.Sprintf(" eta %s", remaining.Round(time.Second))
					} else if rate > 0 {
						eta = fmt.Sprintf(" eta ~%s", remaining.Round(time.Second))
					}
					msg += eta
				} else {
					msg = fmt.Sprintf("  [build] processing, elapsed %s", elapsedRounded)
				}
				fmt.Fprintf(os.Stderr, "\r%s\r", msg)
				os.Stderr.Sync()
			}
		}
	}()

	// Run compile (blocks until done).
	d, err := morphology.CompileFromXMLFile(xmlPath, progress)
	close(stop)
	fmt.Fprint(os.Stderr, "\n") // finish the progress line

	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	if err := d.SaveTo(datPath); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}

func logDone(started time.Time, msg string) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Fprintf(os.Stderr, "\ndone in %s, peak RSS heap %d MiB: %s\n",
		time.Since(started).Round(time.Millisecond), mem.Sys>>20, msg)
}
