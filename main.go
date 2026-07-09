// hyprspacefix: bulk-assign Hyprland workspaces to a monitor.
//
// Emits either modern Lua-config dispatchers (Hyprland 0.55+) or the legacy
// hyprlang syntax, auto-selected from `hyprctl version` (override with --syntax).
//
// Usage: hyprspacefix --name=DP-1 --range=1-5 [--syntax=auto|lua|hyprlang]

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// luaConfigMinMinor is the first 0.x minor release where Hyprland's Lua config
// (and the new dispatcher syntax) became the default.
const luaConfigMinMinor = 55

// checkHyprctl verifies that hyprctl is available.
func checkHyprctl() error {
	if _, err := exec.LookPath("hyprctl"); err != nil {
		return fmt.Errorf("hyprctl not found in PATH: %v", err)
	}
	return nil
}

var versionRe = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

// detectLuaSyntax reports whether the running Hyprland expects the Lua-config
// dispatcher syntax, by parsing `hyprctl version`. Also returns the parsed
// version string for logging.
func detectLuaSyntax() (useLua bool, version string, err error) {
	out, err := exec.Command("hyprctl", "version").CombinedOutput()
	if err != nil {
		return false, "", fmt.Errorf("running `hyprctl version`: %v", err)
	}
	m := versionRe.FindStringSubmatch(string(out))
	if m == nil {
		return false, "", fmt.Errorf("no version number found in `hyprctl version` output")
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	version = m[1] + "." + m[2] + "." + m[3]
	useLua = major > 0 || (major == 0 && minor >= luaConfigMinMinor)
	return useLua, version, nil
}

// workspaceCommands returns the batch sub-commands that bind workspace ws to
// monitor name, in either Lua (0.55+) or legacy hyprlang syntax.
func workspaceCommands(ws int, name string, useLua, persistent bool) []string {
	if useLua {
		// A persistent rule both binds the workspace to the monitor and
		// creates/places it there (the move dispatchers are gone/no-op on
		// 0.55). Without persistence the rule only affects future creation;
		// with it, existing workspaces get relocated too — at the cost of the
		// workspace always existing, so some bars show it even when empty.
		rule := fmt.Sprintf("dispatch hl.dsp.workspace.move({ workspace = '%d', monitor = '%s'", ws, name)
		if persistent {
			rule += ", persistent = true"
		}
		rule += " })"
		return []string{rule}
	}
	kw := fmt.Sprintf("keyword workspace %d,monitor:%s", ws, name)
	if persistent {
		kw += ",persistent:true"
	}
	return []string{
		kw,
		fmt.Sprintf("dispatch moveworkspacetomonitor %d %s", ws, name),
	}
}

func main() {
	monitorName := flag.String("name", "", "Monitor name")
	workspaceRange := flag.String("range", "", "Workspace range (e.g., '1-5')")
	syntax := flag.String("syntax", "auto", "Dispatcher syntax: auto|lua|hyprlang")
	persistent := flag.Bool("persistent", true, "Make workspaces persistent (always exist). Required on 0.55+ to relocate already-existing workspaces; set =false for a cleaner bar if your real layout switches re-home without it.")
	dryRun := flag.Bool("dry-run", false, "Print commands without executing them")
	verbose := flag.Bool("verbose", false, "Enable verbose output")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --name=monitor-name --range=start-end [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example:\n")
		fmt.Fprintf(os.Stderr, "  %s --name=DP-1 --range=1-5\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	log.SetFlags(0)
	if *verbose {
		log.SetFlags(log.Ltime | log.Lmicroseconds)
	}

	if *monitorName == "" || *workspaceRange == "" {
		flag.Usage()
		os.Exit(1)
	}

	if err := checkHyprctl(); err != nil {
		log.Fatal(err)
	}

	// Parse range.
	rangeParts := strings.Split(*workspaceRange, "-")
	if len(rangeParts) != 2 {
		log.Fatal("Range must be in format 'n-m'")
	}
	start, err := strconv.Atoi(rangeParts[0])
	if err != nil {
		log.Fatal("Invalid start range:", err)
	}
	end, err := strconv.Atoi(rangeParts[1])
	if err != nil {
		log.Fatal("Invalid end range:", err)
	}

	// Select dispatcher syntax.
	var useLua bool
	switch strings.ToLower(*syntax) {
	case "lua":
		useLua = true
	case "hyprlang", "legacy":
		useLua = false
	case "auto", "":
		l, ver, derr := detectLuaSyntax()
		if derr != nil {
			log.Fatalf("could not auto-detect Hyprland version (use --syntax=lua|hyprlang to override): %v", derr)
		}
		useLua = l
		if *verbose {
			mode := "hyprlang"
			if useLua {
				mode = "lua"
			}
			log.Printf("detected Hyprland %s -> %s syntax", ver, mode)
		}
	default:
		log.Fatalf("invalid --syntax %q (want auto|lua|hyprlang)", *syntax)
	}

	// Build batch commands.
	var batchCmds []string
	for ws := start; ws <= end; ws++ {
		batchCmds = append(batchCmds, workspaceCommands(ws, *monitorName, useLua, *persistent)...)
	}
	batchCmd := strings.Join(batchCmds, " ; ")

	if *dryRun {
		fmt.Println("Would execute:")
		// Double-quote the shell wrapper so the single-quoted Lua strings
		// inside survive a copy-paste into the shell.
		fmt.Printf("hyprctl --batch \"%s\"\n", batchCmd)
		return
	}

	if *verbose {
		log.Printf("Configuring workspaces %d-%d on monitor %s", start, end, *monitorName)
	}

	// No shell is involved here, so quotes in batchCmd are passed literally.
	out, err := exec.Command("hyprctl", "--batch", batchCmd).CombinedOutput()
	if err != nil {
		log.Fatalf("Error executing batch commands: %v\nOutput: %s", err, out)
	}
	if *verbose {
		log.Printf("Successfully configured %d workspaces", end-start+1)
	}
}
