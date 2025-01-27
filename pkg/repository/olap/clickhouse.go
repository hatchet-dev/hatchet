package olap

import (
	"crypto/tls"
	"fmt"
	"os"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func CreateClickhouseConnection() (clickhouse.Conn, error) {
	return clickhouse.Open(&clickhouse.Options{
		Addr: []string{os.Getenv("CLICKHOUSE_SECURE_NATIVE_HOSTNAME") + ":9440"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: os.Getenv("CLICKHOUSE_PASSWORD"),
		},
		Debugf: func(format string, v ...interface{}) {
			fmt.Printf(format, v)
		},
		TLS: &tls.Config{
			InsecureSkipVerify: true,
		},
		Protocol: clickhouse.Native,
		// See docs on connection pooling with the Clickhouse Go client
		// https://clickhouse.com/docs/en/integrations/go#connection-pooling
		MaxIdleConns: 20,
		MaxOpenConns: 40,
	})
}
