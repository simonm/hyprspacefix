package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"

	"github.com/joshuarubin/go-sway"
)

// WindowManager defines the interface for window manager operations
type WindowManager interface {
	Init() error
	AssignWorkspaceToMonitor(workspace int, monitor string) error
	Close() error
}

// HyprlandManager implements WindowManager for Hyprland
type HyprlandManager struct {
	conn       net.Conn
	socketPath string
}

func NewHyprlandManager() *HyprlandManager {
	return &HyprlandManager{}
}

func (h *HyprlandManager) Init() error {
	signature := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")
	if signature == "" {
		return fmt.Errorf("HYPRLAND_INSTANCE_SIGNATURE not set - are you running Hyprland?")
	}

	// Get current user ID for socket path
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

	// Construct socket path in /run/user/UID/
	h.socketPath = fmt.Sprintf("/run/user/%s/hypr/%s/.socket.sock", currentUser.Uid, signature)

	conn, err := net.Dial("unix", h.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to Hyprland socket at %s: %v", h.socketPath, err)
	}

	h.conn = conn
	return nil
}

func (h *HyprlandManager) sendMessage(message string) error {
	if h.conn == nil {
		return fmt.Errorf("no connection to Hyprland")
	}

	_, err := h.conn.Write([]byte(message + "\n"))
	return err
}

func (h *HyprlandManager) AssignWorkspaceToMonitor(workspace int, monitor string) error {
	// Send both commands in sequence
	cmds := []string{
		fmt.Sprintf("keyword workspace %d,monitor:%s", workspace, monitor),
		fmt.Sprintf("dispatch moveworkspacetomonitor %d %s", workspace, monitor),
	}

	for _, cmd := range cmds {
		if err := h.sendMessage(cmd); err != nil {
			return fmt.Errorf("failed to send command '%s': %v", cmd, err)
		}
	}

	return nil
}

func (h *HyprlandManager) Close() error {
	if h.conn != nil {
		return h.conn.Close()
	}
	return nil
}

// SwayManager implements WindowManager for Sway
type SwayManager struct {
	client sway.Client
	ctx    context.Context
}

func NewSwayManager() *SwayManager {
	return &SwayManager{
		ctx: context.Background(),
	}
}

func (s *SwayManager) Init() error {
	client, err := sway.New(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Sway: %v", err)
	}
	s.client = client
	return nil
}

func (s *SwayManager) AssignWorkspaceToMonitor(workspace int, monitor string) error {
	cmd := fmt.Sprintf("workspace number %d, move workspace to output %s",
		workspace, monitor)

	replies, err := s.client.RunCommand(s.ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to run Sway command: %v", err)
	}

	for _, reply := range replies {
		if !reply.Success {
			return fmt.Errorf("Sway command failed: %s", reply.Error)
		}
	}

	return nil
}

func (s *SwayManager) Close() error {
	return nil
}

func main() {
	// Command line flags
	wmType := flag.String("wm", "", "Window manager type (hyprland or sway)")
	monitorName := flag.String("name", "", "Monitor name")
	workspaceRange := flag.String("range", "", "Workspace range (e.g., '1-5')")
	dryRun := flag.Bool("dry-run", false, "Print commands without executing them")
	verbose := flag.Bool("verbose", false, "Enable verbose output")

	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage: %s --wm=[hyprland|sway] --name=monitor-name --range=start-end [options]\n\n",
			os.Args[0],
		)
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s --wm=hyprland --name=DP-1 --range=1-5\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --wm=sway --name=DP-1 --range=1-5\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Validate flags
	if *wmType == "" || *monitorName == "" || *workspaceRange == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Setup logging
	log.SetFlags(0)
	if *verbose {
		log.SetFlags(log.Ltime | log.Lmicroseconds)
	}

	// Create window manager instance
	var wm WindowManager
	switch strings.ToLower(*wmType) {
	case "hyprland":
		wm = NewHyprlandManager()
	case "sway":
		wm = NewSwayManager()
	default:
		log.Fatalf("Unsupported window manager: %s", *wmType)
	}

	// Initialize window manager
	if err := wm.Init(); err != nil {
		log.Fatalf("Failed to initialize %s: %v", *wmType, err)
	}
	defer wm.Close()

	// Parse workspace range
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

	// Process workspaces
	for ws := start; ws <= end; ws++ {
		if *dryRun {
			log.Printf("Would assign workspace %d to monitor %s", ws, *monitorName)
			continue
		}

		if *verbose {
			log.Printf("Assigning workspace %d to monitor %s", ws, *monitorName)
		}

		if err := wm.AssignWorkspaceToMonitor(ws, *monitorName); err != nil {
			log.Printf("Error assigning workspace %d: %v", ws, err)
		}
	}

	if *verbose {
		log.Printf("Successfully configured %d workspaces", end-start+1)
	}
}
