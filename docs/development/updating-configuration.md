# Updating Configuration

Modifications to Hatchet's configuration should be reflected in the appropriate [`pkg/config`](../pkg/config) package and wired in [`pkg/config/loader/loader.go`](../../pkg/config/loader/loader.go).
```go
type ServerConfig struct {
    RequestTimeout time.Duration `mapstructure:"request_timeout"`
}
```

To ensure configuration is loadable via environment variables, add the corresponding `BindEnv` call in `BindAllEnv()`.
```go
func BindAllEnv(v *viper.Viper) {
    v.BindEnv("request_timeout", "HATCHET_REQUEST_TIMEOUT")
}
```

Finally, document the new environment variable in [`frontend/docs/pages/self-hosting/configuration-options.mdx`](frontend/docs/pages/self-hosting/configuration-options.mdx) and any other relevant documentation.
```markdown
| Variable                  | Description                  | Default Value |
| ------------------------- | ---------------------------- | ------------- |
| `HATCHET_REQUEST_TIMEOUT` | Duration of request timeouts | `5s`          |
```
