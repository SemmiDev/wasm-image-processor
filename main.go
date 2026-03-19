// Server sederhana untuk serve file WASM.
//
// MENGAPA BUTUH SERVER KHUSUS?
// Browser menerapkan aturan CORS dan membutuhkan file .wasm
// di-serve dengan Content-Type: application/wasm
// agar bisa menggunakan WebAssembly.instantiateStreaming()
// (yang lebih efisien dari instantiate biasa).
// Membuka index.html langsung via file:// tidak akan bekerja.

package main

import (
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
)

func main() {
	// Pastikan .wasm di-serve dengan MIME type yang benar
	// Beberapa OS mungkin tidak punya ini secara default
	mime.AddExtensionType(".wasm", "application/wasm")

	port := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		port = ":" + p
	}

	// Serve semua file statis dari folder web/static
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/", withLogging(fs))

	fmt.Printf("🚀 Server running at http://localhost%s\n", port)
	fmt.Printf("   WASM file: web/static/main.wasm\n")
	fmt.Printf("   Press Ctrl+C to stop\n\n")

	log.Fatal(http.ListenAndServe(port, nil))
}

// withLogging adalah middleware sederhana untuk melihat request masuk
func withLogging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s", r.Method, r.URL.Path)
		h.ServeHTTP(w, r)
	})
}
