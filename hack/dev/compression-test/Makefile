.PHONY: setup build-baseline build-compression test-baseline test-compression report help

help:
	@echo "Compression Testing Suite"
	@echo ""
	@echo "Available targets:"
	@echo "  setup              - Initial setup (create network, directories)"
	@echo "  build-baseline      - Build baseline images (requires: git checkout main)"
	@echo "  build-compression   - Build compression images (current branch)"
	@echo "  test-baseline       - Run all baseline tests"
	@echo "  test-compression    - Run all compression tests"
	@echo "  report              - Generate comparison report"
	@echo ""
	@echo "Individual SDK tests:"
	@echo "  test-go-baseline    - Test Go SDK (baseline)"
	@echo "  test-go-compression - Test Go SDK (compression)"
	@echo "  test-ts-baseline    - Test TypeScript SDK (baseline)"
	@echo "  test-ts-compression - Test TypeScript SDK (compression)"
	@echo "  test-py-baseline    - Test Python SDK (baseline)"
	@echo "  test-py-compression - Test Python SDK (compression)"

setup:
	./scripts/setup.sh

build-baseline:
	@echo "Building baseline images (main branch)..."
	@echo "Make sure you're on the main branch!"
	@read -p "Press enter to continue..."
	@cd ../.. && docker build -t go-disabled-compression -f hack/dev/compression-test/Dockerfile.client-go .
	@cd ../.. && docker build -t typescript-disabled-compression -f hack/dev/compression-test/Dockerfile.client-ts .
	@cd ../.. && docker build -t python-disabled-compression -f hack/dev/compression-test/Dockerfile.client-python .
	@echo "Baseline images built successfully!"

build-compression:
	@echo "Building compression images (current branch)..."
	@read -p "Press enter to continue..."
	@cd ../.. && docker build -t go-enabled-compression -f hack/dev/compression-test/Dockerfile.client-go .
	@cd ../.. && docker build -t typescript-enabled-compression -f hack/dev/compression-test/Dockerfile.client-ts .
	@cd ../.. && docker build -t python-enabled-compression -f hack/dev/compression-test/Dockerfile.client-python .
	@echo "Compression images built successfully!"

test-baseline:
	./scripts/run_all_tests.sh disabled

test-compression:
	./scripts/run_all_tests.sh enabled

test-go-baseline:
	./scripts/run_test.sh go disabled

test-go-compression:
	./scripts/run_test.sh go enabled

test-ts-baseline:
	./scripts/run_test.sh typescript disabled

test-ts-compression:
	./scripts/run_test.sh typescript enabled

test-py-baseline:
	./scripts/run_test.sh python disabled

test-py-compression:
	./scripts/run_test.sh python enabled

report:
	./scripts/generate_report.sh

clean:
	docker rm -f hatchet-engine hatchet-client-go hatchet-client-typescript hatchet-client-python 2>/dev/null || true
	rm -rf results/*.log results/*.summary
