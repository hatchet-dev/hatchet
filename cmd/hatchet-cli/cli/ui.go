package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/spf13/cobra"

	configcli "github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/config/cli"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/ui"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

var uiCmd = &cobra.Command{
	Use:     "ui",
	Aliases: []string{"web", "dashboard"},
	Short:   "Serve the Hatchet dashboard UI locally",
	Long: `Serve the Hatchet dashboard UI from the CLI, pointed at an existing Hatchet
instance. The UI is bundled into the CLI binary and proxied to the API server of
your selected profile, so you can browse a self-hosted deployment without
deploying the frontend separately.`,
	Example: `  # Serve the UI for your default profile
  hatchet ui

  # Serve the UI for a specific profile
  hatchet ui --profile production

  # Point the UI at an explicit API server (skips profile resolution)
  hatchet ui --api-url http://localhost:8080

  # Serve on a fixed port without opening a browser
  hatchet ui --port 9000 --no-open`,
	Run: func(cmd *cobra.Command, args []string) {
		runUI(cmd)
	},
}

func runUI(cmd *cobra.Command) {
	apiURLFlag, _ := cmd.Flags().GetString("api-url")
	profileFlag, _ := cmd.Flags().GetString("profile")
	port, _ := cmd.Flags().GetInt("port")
	host, _ := cmd.Flags().GetString("host")
	noOpen, _ := cmd.Flags().GetBool("no-open")

	if !ui.Bundled() {
		configcli.Logger.Fatal("This CLI build does not include the dashboard UI.")
	}

	target, insecureSkipVerify, profileName := resolveUITarget(apiURLFlag, profileFlag)

	handler, err := newUIHandler(target, insecureSkipVerify)
	if err != nil {
		configcli.Logger.Fatalf("could not build UI server: %v", err)
	}

	listener, err := listenUI(host, port)
	if err != nil {
		configcli.Logger.Fatalf("%v", err)
	}

	localURL := fmt.Sprintf("http://%s:%d", browserHost(host), listener.Addr().(*net.TCPAddr).Port)

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	fmt.Println(uiStartedView(localURL, target.String(), profileName))

	if !noOpen {
		openBrowser(localURL)
	}

	interruptCh := cmdutils.InterruptChan()

	select {
	case err := <-errCh:
		configcli.Logger.Fatalf("UI server failed: %v", err)
	case <-interruptCh:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}
}

func resolveUITarget(apiURLFlag, profileFlag string) (target *url.URL, insecureSkipVerify bool, profileName string) {
	if apiURLFlag != "" {
		parsed, err := url.Parse(apiURLFlag)
		if err != nil {
			configcli.Logger.Fatalf("invalid --api-url '%s': %v", apiURLFlag, err)
		}

		return parsed, false, ""
	}

	selectedProfile := profileFlag
	if selectedProfile == "" {
		selectedProfile = selectProfileForm(true)
	}

	if selectedProfile == "" {
		configcli.Logger.Fatal("no profile selected. Configure a profile with 'hatchet profile' or pass --api-url.")
	}

	profile, err := configcli.GetProfile(selectedProfile)
	if err != nil {
		configcli.Logger.Fatalf("could not get profile '%s': %v", selectedProfile, err)
	}

	if profile.ApiServerURL == "" {
		configcli.Logger.Fatalf("profile '%s' has no API server URL configured", selectedProfile)
	}

	parsed, err := url.Parse(profile.ApiServerURL)
	if err != nil {
		configcli.Logger.Fatalf("profile '%s' has an invalid API server URL '%s': %v", selectedProfile, profile.ApiServerURL, err)
	}

	return parsed, profile.TLSStrategy == "none", selectedProfile
}

func newUIHandler(target *url.URL, insecureSkipVerify bool) (http.Handler, error) {
	origin := target.Scheme + "://" + target.Host

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)

			pr.Out.Host = target.Host
			if pr.Out.Header.Get("Origin") != "" {
				pr.Out.Header.Set("Origin", origin)
			}
			if pr.Out.Header.Get("Referer") != "" {
				pr.Out.Header.Set("Referer", origin+pr.Out.URL.Path)
			}
		},
		ModifyResponse: func(resp *http.Response) error {
			if cookies := resp.Header["Set-Cookie"]; len(cookies) > 0 {
				for i, c := range cookies {
					cookies[i] = rewriteSetCookie(c)
				}
			}
			return nil
		},
	}

	if insecureSkipVerify {
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // nolint:gosec
		}
	}

	spa, err := newSPAHandler()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", proxy)
	mux.Handle("/", spa)

	return mux, nil
}

func rewriteSetCookie(cookie string) string {
	parts := strings.Split(cookie, ";")
	out := parts[:0]

	for i, p := range parts {
		trimmed := strings.TrimSpace(p)

		if i > 0 {
			lower := strings.ToLower(trimmed)
			if strings.HasPrefix(lower, "domain=") || lower == "secure" {
				continue
			}
		}

		out = append(out, trimmed)
	}

	return strings.Join(out, "; ")
}

func newSPAHandler() (http.Handler, error) {
	assets, err := ui.Assets()
	if err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.FS(assets))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")

		reqPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if reqPath == "" {
			reqPath = "index.html"
		}

		if _, err := fs.Stat(assets, reqPath); err != nil {
			serveIndex(w, assets)
			return
		}

		if base := path.Base(r.URL.Path); strings.Contains(base, "html") || strings.Contains(base, "js") || base == "." || base == "/" {
			w.Header().Set("Cache-Control", "no-cache")
		}

		fileServer.ServeHTTP(w, r)
	}), nil
}

func serveIndex(w http.ResponseWriter, assets fs.FS) {
	index, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(index)
}

func listenUI(host string, port int) (net.Listener, error) {
	if port != 0 {
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			return nil, fmt.Errorf("could not bind to %s:%d: %w", host, port, err)
		}

		return ln, nil
	}

	const base = 8080

	for p := base; p < base+100; p++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, p))
		if err == nil {
			return ln, nil
		}
	}

	return nil, fmt.Errorf("could not find a free port starting at %d; specify one with --port", base)
}

func browserHost(host string) string {
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		return "localhost"
	default:
		return host
	}
}

func uiStartedView(localURL, targetURL, profileName string) string {
	var lines []string

	lines = append(lines, styles.SuccessMessage("Hatchet dashboard is running!"))
	lines = append(lines, "")
	lines = append(lines, styles.KeyValue("Dashboard", localURL))
	if profileName != "" {
		lines = append(lines, styles.KeyValue("Profile", profileName))
	}
	lines = append(lines, styles.KeyValue("API server", targetURL))
	lines = append(lines, "")
	lines = append(lines, styles.Muted.Render("Press Ctrl+C to stop."))

	return styles.SuccessBox.Render(strings.Join(lines, "\n"))
}

func init() {
	rootCmd.AddCommand(uiCmd)

	uiCmd.Flags().StringP("profile", "n", "", "Profile whose API server the UI targets (default: default profile)")
	uiCmd.Flags().String("api-url", "", "API server URL to proxy to (overrides the profile's API server URL)")
	uiCmd.Flags().IntP("port", "p", 0, "Port to serve the UI on (default: auto-detect starting at 8080)")
	uiCmd.Flags().String("host", "localhost", "Host interface to bind the UI server to")
	uiCmd.Flags().Bool("no-open", false, "Do not automatically open a browser")
}
