package common

import "strings"

// FindOutFlag scans positional args for -o <path> or -o=<path>.
//
// The stdlib flag package stops parsing at the first non-flag argument, so
// in a CLI shaped as "<subcommand> [flags]" any -o that follows the
// subcommand is left in flag.Args() instead of being parsed automatically.
func FindOutFlag(args []string) string {
	for i := range args {
		if args[i] == "-o" && i+1 < len(args) {
			return args[i+1]
		}
		if v, ok := strings.CutPrefix(args[i], "-o="); ok {
			return v
		}
	}
	return ""
}

// StripOutFlag removes -o <path> / -o=<path> from args.
func StripOutFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "-o" {
			i++
			continue
		}
		if strings.HasPrefix(args[i], "-o=") {
			continue
		}
		out = append(out, args[i])
	}
	return out
}
