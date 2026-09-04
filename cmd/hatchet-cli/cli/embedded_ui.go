package cli

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"html/template"
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
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

var uiCmd = &cobra.Command{
	Use:   "embedded-ui",
	Short: "Serve the dashboard UI for an embedded Hatchet instance",
	Long: `Serve the Hatchet dashboard UI for an embedded Hatchet instance. Embedded
instances run inside your application and do not ship a frontend; this command
serves the UI bundled in the CLI binary and proxies API requests to the
instance's API server. Access is protected by a one-time token in the opened URL.`,
	Example: `  # Serve the UI for an embedded instance's API server
  hatchet embedded-ui --api-url http://localhost:8080

  # Serve the UI for a configured profile
  hatchet embedded-ui --profile local

  # Serve on a fixed port without opening a browser
  hatchet embedded-ui --api-url http://localhost:8080 --port 9000 --no-open`,
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

	if err := checkEmbedded(target, insecureSkipVerify); err != nil {
		configcli.Logger.Fatalf("%v", err)
	}

	handler, err := newUIHandler(target, insecureSkipVerify)
	if err != nil {
		configcli.Logger.Fatalf("could not build UI server: %v", err)
	}

	listener, err := listenUI(host, port)
	if err != nil {
		configcli.Logger.Fatalf("%v", err)
	}

	token, err := randomToken()
	if err != nil {
		configcli.Logger.Fatalf("could not generate access token: %v", err)
	}

	localURL := fmt.Sprintf("http://%s:%d", browserHost(host), listener.Addr().(*net.TCPAddr).Port)
	tokenURL := localURL + "/?ui_token=" + token

	server := &http.Server{
		Handler:           tokenGate(token, handler),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	fmt.Println(uiStartedView(tokenURL, target.String(), profileName))

	if !noOpen {
		openBrowser(tokenURL)
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
		parsed, err := parseTargetURL(apiURLFlag)
		if err != nil {
			configcli.Logger.Fatalf("invalid --api-url '%s': %v", apiURLFlag, err)
		}

		return parsed, false, ""
	}

	if profileFlag == "" {
		parsed, ok := defaultEmbeddedTarget()
		if !ok {
			configcli.Logger.Fatalf("no embedded instance found at %s. Pass --api-url (printed on the engine's ready line) or --profile.", defaultEmbeddedAPIURL)
		}

		return parsed, false, ""
	}

	selectedProfile := profileFlag

	profile, err := configcli.GetProfile(selectedProfile)
	if err != nil {
		configcli.Logger.Fatalf("could not get profile '%s': %v", selectedProfile, err)
	}

	if profile.ApiServerURL == "" {
		configcli.Logger.Fatalf("profile '%s' has no API server URL configured", selectedProfile)
	}

	parsed, err := parseTargetURL(profile.ApiServerURL)
	if err != nil {
		configcli.Logger.Fatalf("profile '%s' has an invalid API server URL '%s': %v", selectedProfile, profile.ApiServerURL, err)
	}

	return parsed, profile.TLSStrategy == "none", selectedProfile
}

// defaultEmbeddedAPIURL is where the embedded engine binds its API when the
// port is free (see hatchet-embedded's DefaultAPIPort).
const defaultEmbeddedAPIURL = "http://localhost:28243"

func defaultEmbeddedTarget() (*url.URL, bool) {
	parsed, err := parseTargetURL(defaultEmbeddedAPIURL)
	if err != nil {
		return nil, false
	}

	if err := checkEmbedded(parsed, false); err != nil {
		return nil, false
	}

	fmt.Printf("Using the embedded instance at %s\n", defaultEmbeddedAPIURL)

	return parsed, true
}

const uiTokenCookie = "hatchet-ui-token"

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func tokenGate(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("ui_token"); q != "" {
			if subtle.ConstantTimeCompare([]byte(q), []byte(token)) != 1 {
				http.Error(w, "invalid token", http.StatusForbidden)
				return
			}

			http.SetCookie(w, &http.Cookie{ // nolint:gosec // served over plain HTTP on loopback; Secure would drop the cookie
				Name:     uiTokenCookie,
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})

			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		c, err := r.Cookie(uiTokenCookie)
		if err != nil || subtle.ConstantTimeCompare([]byte(c.Value), []byte(token)) != 1 {
			http.Error(w, "access denied: open the URL printed by 'hatchet embedded-ui'", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func checkEmbedded(target *url.URL, insecureSkipVerify bool) error {
	httpClient := &http.Client{Timeout: 5 * time.Second}

	if insecureSkipVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // nolint:gosec
		}
	}

	client, err := rest.NewClientWithResponses(target.String(), rest.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("could not build an API client for %s: %w", target, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.MetadataGetWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("could not reach %s: %w", target, err)
	}

	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return fmt.Errorf("%s does not look like a Hatchet API server (GET /api/v1/meta returned %d)", target, resp.StatusCode())
	}

	if resp.JSON200.Embedded == nil || !*resp.JSON200.Embedded {
		return fmt.Errorf("%s is not an embedded Hatchet instance; 'hatchet embedded-ui' only serves the UI for embedded instances", target)
	}

	return nil
}

func parseTargetURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("must be an http:// or https:// URL")
	}

	if u.Host == "" {
		return nil, fmt.Errorf("missing host")
	}

	return u, nil
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
	mux.Handle("/api", proxy)
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

	// index.html is a template rendered by the server that hosts it (see
	// cmd/hatchet-staticfileserver); the UI is always served at the root here
	index, err := renderIndex(assets)
	if err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.FS(assets))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")

		reqPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if reqPath == "" || reqPath == "index.html" {
			serveIndex(w, index)
			return
		}

		if _, err := fs.Stat(assets, reqPath); err != nil {
			serveIndex(w, index)
			return
		}

		if base := path.Base(r.URL.Path); strings.HasSuffix(base, ".html") || strings.HasSuffix(base, ".js") || base == "." || base == "/" {
			w.Header().Set("Cache-Control", "no-cache")
		}

		fileServer.ServeHTTP(w, r)
	}), nil
}

func renderIndex(assets fs.FS) ([]byte, error) {
	raw, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		return nil, fmt.Errorf("index.html not found in the bundled UI: %w", err)
	}

	t, err := template.New("index.html").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("could not parse the bundled index.html: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, struct{ BasePath string }{"/"}); err != nil {
		return nil, fmt.Errorf("could not render the bundled index.html: %w", err)
	}

	return buf.Bytes(), nil
}

func serveIndex(w http.ResponseWriter, index []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(index)
}

func listenUI(host string, port int) (net.Listener, error) {
	// Bind the concrete IPv4 loopback: "localhost" may bind only [::1] while
	// another process holds 127.0.0.1 on the same port, making the probe pass
	// on a port the browser cannot reach us on.
	if host == "localhost" {
		host = "127.0.0.1"
	}

	if port != 0 {
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			return nil, fmt.Errorf("could not bind to %s:%d: %w", host, port, err)
		}

		return ln, nil
	}

	const base = 8080

	for p := base; p < base+1000; p++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, p))
		if err == nil {
			return ln, nil
		}
	}

	return nil, fmt.Errorf("could not find a free port starting at %d; specify one with --port", base)
}

func browserHost(host string) string {
	switch host {
	case "", "0.0.0.0", "::", "[::]", "localhost":
		return "127.0.0.1"
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

	uiCmd.Flags().StringP("profile", "n", "", "Profile whose API server the UI targets")
	uiCmd.Flags().String("api-url", "", "API server URL to proxy to (overrides the profile's API server URL)")
	uiCmd.Flags().IntP("port", "p", 0, "Port to serve the UI on (default: auto-detect starting at 8080)")
	uiCmd.Flags().String("host", "localhost", "Host interface to bind the UI server to")
	uiCmd.Flags().Bool("no-open", false, "Do not automatically open a browser")
}
