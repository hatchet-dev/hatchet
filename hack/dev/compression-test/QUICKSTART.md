# Quick Start Guide

## Build and Test (Current Branch)

1. **Set your environment variables**:

   ```bash
   export HATCHET_CLIENT_TOKEN="your-token-here"
   export HATCHET_CLIENT_HOST_PORT="localhost:7070"  # Where your engine is running
   ```

2. **Build all images** (from repo root):

   ```bash
   cd /path/to/hatchet
   ./hack/dev/compression-test/scripts/build_all.sh enabled
   ```

3. **Run all tests**:

   ```bash
   cd hack/dev/compression-test
   ./scripts/run_all_tests.sh enabled
   ```

4. **Generate report**:
   ```bash
   ./scripts/generate_report.sh
   ```

## What You Need

- **HATCHET_CLIENT_TOKEN**: Required - Your Hatchet API token
- **HATCHET_CLIENT_HOST_PORT**: Required - Where your engine gRPC is running (e.g., `localhost:7070`)
- **Engine running**: You need to have your Hatchet engine running separately (not managed by these scripts)

## What Gets Tested

Each SDK (Go, TypeScript, Python) will:

- Connect to your engine
- Emit 600 events over 60 seconds (10 events/second)
- Each event has a 100KB payload
- Network traffic is measured and compared

## Results

Results are saved in `results/` directory:

- `results/enabled/` - Test results with compression
- `results/disabled/` - Test results without compression (run on main branch)

The final report shows bandwidth savings from compression.
