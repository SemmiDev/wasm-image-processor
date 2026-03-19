# Go WASM Image Processor

Demonstrasi WebAssembly dengan Go.
Semua pemrosesan gambar (grayscale, invert, edge detection, dll)
berjalan sepenuhnya di browser via Go WASM — tanpa server backend.

## Quick Start

```bash
# 1. Build Go source ke WASM
GOOS=js GOARCH=wasm go build -o web/static/main.wasm ./cmd/wasm/

# 2. Salin wasm_exec.js dari Go SDK
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" web/static/

# 3. Jalankan server (WASM butuh HTTP server, tidak bisa via file://)
go run main.go

# 4. Buka browser
open http://localhost:8080
```

Atau gunakan Makefile:
```bash
make dev
```

## Struktur Project

```
wasm-image-processor/
├── cmd/wasm/
│   └── main.go          ← Kode Go yang dikompilasi ke WASM
│                          (filter grayscale, invert, sobel, dll)
├── web/static/
│   ├── index.html       ← Frontend (HTML + JS)
│   ├── main.wasm        ← Output binary (di-generate oleh build)
│   └── wasm_exec.js     ← Go WASM bridge (dari Go SDK)
├── main.go              ← HTTP server
└── Makefile
```

## Konsep yang Dipelajari

**Linear Memory**: WASM dan JS berbagi byte array yang sama.
`js.CopyBytesToGo` dan `js.CopyBytesToJS` adalah operasi copy
yang memindahkan data antara heap JS dan heap Go.

**Function Export**: Fungsi Go diekspos ke JS dengan:
```go
js.Global().Set("goGrayscale", js.FuncOf(grayscale))
```
Lalu dipanggil dari JS seperti fungsi biasa:
```js
window.goGrayscale(imageData.data);
```

**Blocking main()**: Go WASM butuh `main()` yang tidak keluar.
Channel kosong `<-done` melakukan ini dengan efisien (tidak busy-wait).

## Filter yang Tersedia

- **Grayscale** — Perceptual luminance (ITU-R BT.601)
- **Invert** — Membalik setiap channel warna
- **Brightness** — Tambah/kurangi kecerahan dengan parameter
- **Edge Detect** — Sobel operator (computer vision technique)
- **Pixelate** — Rata-rata warna per blok NxN

## Optimasi untuk Production

Binary WASM yang dihasilkan Go cukup besar (~1.6MB) karena
membawa runtime Go. Beberapa teknik optimasi:

1. **wasm-opt** dari Binaryen: `wasm-opt -O3 -o out.wasm main.wasm`
2. **Gzip/Brotli compression** di server — WASM sangat compressible
3. **TinyGo** — alternatif compiler yang menghasilkan binary jauh lebih kecil,
   tapi dengan keterbatasan fitur Go standar
4. **Caching**: Set header `Cache-Control: max-age=31536000` untuk .wasm
