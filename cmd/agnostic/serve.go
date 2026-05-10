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
	serveListen  string
	serveOpen    bool
	serveToken   string
	serveTLSCert string
	serveTLSKey  string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web UI server",
	Long: `Start a local HTTP(S) server with a web interface for managing packages.

The server binds to localhost by default for security. Use --listen to change
the address and port. Use --open to open the browser automatically.

Examples:
  agnostic serve
  agnostic serve --listen 127.0.0.1:9090
  agnostic serve --listen 0.0.0.0:8080 --token mytoken
  agnostic serve --tls-cert cert.pem --tls-key key.pem`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := manager.NewAgnosticManager()

		var opts []server.ServerOption
		if serveToken != "" {
			opts = append(opts, server.WithToken(serveToken))
		}
		if serveTLSCert != "" && serveTLSKey != "" {
			opts = append(opts, server.WithTLS(serveTLSCert, serveTLSKey))
		}

		srv := server.New(mgr, opts...)

		scheme := "http"
		if serveTLSCert != "" && serveTLSKey != "" {
			scheme = "https"
		}

		fmt.Printf("🌐 Web UI starting at %s://%s\n", scheme, serveListen)
		fmt.Println("Press Ctrl+C to stop")

		if serveOpen {
			fmt.Println("🔓 Opening browser...")
			openBrowser(scheme + "://" + serveListen)
		}

		return srv.Listen(serveListen)
	},
}

func init() {
	serveCmd.Flags().StringVarP(&serveListen, "listen", "l", "127.0.0.1:8080", "Address and port to listen on")
	serveCmd.Flags().BoolVarP(&serveOpen, "open", "o", false, "Open browser automatically")
	serveCmd.Flags().StringVar(&serveToken, "token", "", "Fixed auth token (auto-generated if empty)")
	serveCmd.Flags().StringVar(&serveTLSCert, "tls-cert", "", "TLS certificate file path")
	serveCmd.Flags().StringVar(&serveTLSKey, "tls-key", "", "TLS private key file path")
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
