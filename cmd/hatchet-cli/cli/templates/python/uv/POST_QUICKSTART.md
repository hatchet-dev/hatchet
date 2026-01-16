Trigger a workflow (in another terminal):
```sh
hatchet trigger simple
```

**Notes:**
- The virtual environment will be automatically created by `hatchet worker dev`
- `uv run` automatically uses the virtual environment, no need to activate it manually
- The `uv.lock` file will be automatically generated when dependencies are installed
