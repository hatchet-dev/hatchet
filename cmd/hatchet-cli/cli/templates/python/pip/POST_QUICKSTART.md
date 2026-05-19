Trigger a workflow (in another terminal):
```sh
hatchet trigger simple
```

**Notes:**
- The virtual environment (`.venv`) will be automatically created by `hatchet worker dev`
- If running commands manually (not via `hatchet worker`), make sure to activate the virtual environment first:
  ```sh
  source .venv/bin/activate  # On Windows: .venv\Scripts\activate
  ```
