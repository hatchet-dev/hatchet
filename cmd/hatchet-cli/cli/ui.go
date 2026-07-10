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
		configcli.Logger.Fatal("This CLI build does not include the dashboard UI. " +
			"Use an official release binary, or run 'task build-cli' to build one with the UI embedded.")
	}

	target, insecureSkipVerify, profileName := resolveUITarget(apiURLFlag, profileFlag)

	handler, err := newUIHandler(target, insecureSkipVerify)
	if err != nil {
		configcli.Logger.Fatalf("could not build UI server: %v", err)
	}

	listenPort, err := resolveUIPort(host, port)
	if err != nil {
		configcli.Logger.Fatalf("%v", err)
	}

	addr := fmt.Sprintf("%s:%d", host, listenPort)
	localURL := fmt.Sprintf("http://%s:%d", host, listenPort)

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

// resolveUITarget determines the API server the UI should be proxied to. An
// explicit --api-url wins; otherwise the selected profile's API server URL is
// used.
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

// newUIHandler builds the HTTP handler that serves the embedded UI and proxies
// /api requests to the target Hatchet API server.
func newUIHandler(target *url.URL, insecureSkipVerify bool) (http.Handler, error) {
	origin := target.Scheme + "://" + target.Host

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)

			// Present the request to the API server as same-origin so cookie/CSRF
			// checks pass, since the browser only ever talks to our local origin.
			pr.Out.Host = target.Host
			if pr.Out.Header.Get("Origin") != "" {
				pr.Out.Header.Set("Origin", origin)
			}
			if pr.Out.Header.Get("Referer") != "" {
				pr.Out.Header.Set("Referer", origin+pr.Out.URL.Path)
			}
		},
		// Rewrite Set-Cookie so session cookies from the (possibly remote, https)
		// API server are accepted by the browser against our local http origin.
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
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // nolint:gosec // opt-in via profile tlsStrategy=none
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

// rewriteSetCookie strips the Domain attribute and the Secure flag so a cookie
// scoped to a remote https host is stored against our local http origin.
func rewriteSetCookie(cookie string) string {
	parts := strings.Split(cookie, ";")
	out := parts[:0]

	for i, p := range parts {
		trimmed := strings.TrimSpace(p)

		// index 0 is the name=value pair, which we always keep.
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

// newSPAHandler serves the embedded UI bundle with single-page-app fallback to
// index.html, or a built-in notice when no UI build is bundled.
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
			// Unknown path: fall back to index.html so client-side routing works.
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

// resolveUIPort returns the port to listen on. A non-zero port is used as-is;
// port 0 auto-detects a free port starting at the default base.
func resolveUIPort(host string, port int) (int, error) {
	if port != 0 {
		return port, nil
	}

	const base = 8080

	for p := base; p < base+100; p++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, p))
		if err == nil {
			_ = ln.Close()
			return p, nil
		}
	}

	return 0, fmt.Errorf("could not find a free port starting at %d; specify one with --port", base)
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
