package embeddedpg

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

type Postgres struct {
	ConnStr string
	Port    int
	stop    func() error
}

func (p *Postgres) Stop() error { return p.stop() }

func Start(user, password, database string, port int, version string) (*Postgres, error) {
	if port == 0 {
		p, err := freePort()
		if err != nil {
			return nil, err
		}
		port = p
	}
	if port < 0 || port > 65535 {
		return nil, fmt.Errorf("port %d out of range", port)
	}

	pgVersion := parseVersion(version)

	runtimeDir, err := os.MkdirTemp("", "hatchet-epg-")
	if err != nil {
		return nil, err
	}

	binPath := binariesPath(pgVersion)

	unlock, err := lockBinaries(binPath)
	if err != nil {
		_ = os.RemoveAll(runtimeDir)
		return nil, err
	}

	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username(user).
		Password(password).
		Database(database).
		Port(uint32(port)).
		Version(pgVersion).
		CachePath(cachePath()).
		BinariesPath(binPath).
		RuntimePath(runtimeDir).
		DataPath(filepath.Join(runtimeDir, "data")).
		StartParameters(map[string]string{"timezone": "UTC"}).
		Logger(io.Discard))

	err = pg.Start()
	unlock()
	if err != nil {
		_ = os.RemoveAll(runtimeDir)
		return nil, fmt.Errorf("start embedded postgres: %w", err)
	}

	return &Postgres{
		ConnStr: fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable", user, password, port, database),
		Port:    port,
		stop: func() error {
			stopErr := pg.Stop()
			_ = os.RemoveAll(runtimeDir)
			return stopErr
		},
	}, nil
}

func cacheRoot() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "hatchet-embedded-postgres")
}

func cachePath() string { return filepath.Join(cacheRoot(), "download") }

func binariesPath(v embeddedpostgres.PostgresVersion) string {
	return filepath.Join(cacheRoot(), "bin", string(v))
}

func binariesReady(binPath string) bool {
	_, err := os.Stat(filepath.Join(binPath, "bin", "pg_ctl"))
	return err == nil
}

func parseVersion(v string) embeddedpostgres.PostgresVersion {
	major := ""
	for _, r := range v {
		if r < '0' || r > '9' {
			break
		}
		major += string(r)
	}
	switch major {
	case "18":
		return embeddedpostgres.V18
	case "17":
		return embeddedpostgres.V17
	case "15":
		return embeddedpostgres.V15
	case "14":
		return embeddedpostgres.V14
	default:
		return embeddedpostgres.V16
	}
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
