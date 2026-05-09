package agnostic

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/ElioNeto/agnostikos/internal/manager"
	"github.com/ElioNeto/agnostikos/internal/server"
	"github.com/spf13/cobra"
)

var (
	servePort string
	serveOpen bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web UI server",
	Long: `Start a local HTTP server with a web interface for managing packages.

The server binds to localhost by default for security. Use --port to change
the listening port. Use --open to open the browser automatically.

Examples:
  agnostic serve
  agnostic serve --port 9090
  agnostic serve --open`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := manager.NewAgnosticManager()
		srv := server.New(mgr)

		addr := "127.0.0.1:" + servePort
		fmt.Printf("🌐 Web UI starting at http://%s\n", addr)
		fmt.Println("Press Ctrl+C to stop")

		if serveOpen {
			fmt.Println("🔓 Opening browser...")
			openBrowser("http://" + addr)
		}

		return srv.Listen(addr)
	},
}

func init() {
	serveCmd.Flags().StringVarP(&servePort, "port", "p", "8080", "Port to listen on")
	serveCmd.Flags().BoolVarP(&serveOpen, "open", "o", false, "Open browser automatically")
	rootCmd.AddCommand(serveCmd)
}

// openBrowser tries to open a URL in the default browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.CommandContext(context.Background(), "xdg-open", url)
	case "darwin":
		cmd = exec.CommandContext(context.Background(), "open", url)
	case "windows":
		cmd = exec.CommandContext(context.Background(), "rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		if err := cmd.Start(); err != nil {
			fmt.Printf("Failed to open browser: %v\n", err)
		}
	}
}
