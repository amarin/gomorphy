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

var (
	debug       = flag.Bool("d", false, "enable debug logging")
	skipDL      = flag.Bool("l", false, "use local file only, skip downloading")
	output      = flag.String("o", "", "output .dat path (default: auto)")
	showVersion = flag.Bool("version", false, "print version and exit")
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
  -version    print version and exit

Examples:
  gomorphy_build update          # download + unpack + compile
  gomorphy_build update -l       # skip download, only compile
  gomorphy_build compile         # compile existing dict.xml
  gomorphy_build compile -o /tmp/oc.dat   # custom output
`)
	}

	flag.Parse()

	if *showVersion {
		fmt.Println(morphology.Version)
		return
	}

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := flag.Arg(0)
	if o := common.FindOutFlag(flag.Args()[1:]); o != "" {
		*output = o
	}

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
func compileAndSave(xmlPath, datPath string, started time.Time) error {
	pp := newProgressPrinter(started)
	go pp.run()

	d, err := morphology.CompileFromXMLFile(xmlPath, pp.update)
	pp.stop()

	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	if err := d.SaveTo(datPath); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}

// progressPrinter prints a "[phase] processed/total (pct%), rate, elapsed,
// eta" line to stderr every 10s from a background goroutine (run), driven
// by progress reports from the compiler (update).
type progressPrinter struct {
	started time.Time

	processed int64
	total     int64
	phase     int64 // 0=insert, 1=compile

	stopCh chan struct{}

	// touched only by run's own goroutine, never concurrently.
	lastProcessed int64
	lastReport    time.Time
}

func newProgressPrinter(started time.Time) *progressPrinter {
	return &progressPrinter{started: started, stopCh: make(chan struct{}), lastReport: started}
}

// update is the progress callback passed to the compiler: records the
// latest counters and flips to the "compile" phase once insertion is
// nearly done. Safe for concurrent use with run.
func (p *progressPrinter) update(processed, total int) {
	atomic.StoreInt64(&p.processed, int64(processed))
	atomic.StoreInt64(&p.total, int64(total))
	if total > 0 && int64(processed) >= int64(total)-100 {
		atomic.StoreInt64(&p.phase, 1)
	}
}

// run prints a progress line every 10s until stop is called. Meant to run
// in its own goroutine.
func (p *progressPrinter) run() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.printLine()
		}
	}
}

func (p *progressPrinter) printLine() {
	elapsed := time.Since(p.started)
	elapsedRounded := elapsed.Round(time.Second)
	processed := atomic.LoadInt64(&p.processed)
	total := atomic.LoadInt64(&p.total)
	phase := atomic.LoadInt64(&p.phase)

	rate := float64(0)
	if processed != p.lastProcessed {
		dt := time.Since(p.lastReport).Seconds()
		if dt > 0 {
			rate = float64(processed-p.lastProcessed) / dt
		}
	}
	p.lastProcessed = processed
	p.lastReport = time.Now()

	var msg, eta string
	if total > 0 {
		pct := int(processed * 100 / total)
		remainingKeys := total - processed
		var remaining time.Duration
		if rate > 0 && remainingKeys > 0 {
			remaining = time.Duration(float64(remainingKeys)/rate) * time.Second
		}

		if phase == 0 {
			msg = fmt.Sprintf("  [build] %d/%d (%d%%), %d keys/s, elapsed %s",
				processed, total, pct, int64(rate), elapsedRounded)
		} else {
			msg = fmt.Sprintf("  [compile] %d/%d (%d%%), %d nodes/s, elapsed %s",
				processed, total, pct, int64(rate), elapsedRounded)
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
	_ = os.Stderr.Sync()
}

// stop halts the background goroutine and prints a trailing newline to
// finish the progress line.
func (p *progressPrinter) stop() {
	close(p.stopCh)
	fmt.Fprint(os.Stderr, "\n")
}

func logDone(started time.Time, msg string) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Fprintf(os.Stderr, "\ndone in %s, peak RSS heap %d MiB: %s\n",
		time.Since(started).Round(time.Millisecond), mem.Sys>>20, msg)
}
