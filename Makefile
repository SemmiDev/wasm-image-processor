# Makefile untuk Go WASM Image Processor
# Penggunaan:
#   make build   — compile Go ke WASM
#   make serve   — jalankan HTTP server
#   make dev     — build + serve sekaligus
#   make clean   — hapus build artifacts

GOROOT := $(shell go env GOROOT 2>/dev/null || echo "/usr/lib/go-1.22")
WASM_OUT := web/static/main.wasm
WASM_SRC := ./cmd/wasm/

.PHONY: build serve dev clean

## build: Compile Go source ke WebAssembly
build:
	@echo "🔨 Compiling Go → WASM..."
	GOOS=js GOARCH=wasm go build -o $(WASM_OUT) $(WASM_SRC)
	@echo "✅ Output: $(WASM_OUT) ($$(du -h $(WASM_OUT) | cut -f1))"

## copy-glue: Salin wasm_exec.js dari Go SDK
copy-glue:
	@cp "$(GOROOT)/lib/wasm/wasm_exec.js" web/static/
	@echo "✅ wasm_exec.js copied"

## serve: Jalankan HTTP server
serve:
	@echo "🚀 Starting server at http://localhost:8080"
	go run main.go

## dev: Build lalu serve
dev: build copy-glue serve

## clean: Bersihkan build artifacts
clean:
	rm -f $(WASM_OUT)
	@echo "🧹 Cleaned"

## size: Analisis ukuran binary
size: build
	@echo "\n📊 Binary size analysis:"
	@ls -lh $(WASM_OUT)
	@echo "\nNote: Untuk production, gunakan wasm-opt dari Binaryen:"
	@echo "  wasm-opt -O3 -o optimized.wasm $(WASM_OUT)"
