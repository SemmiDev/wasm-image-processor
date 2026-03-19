// +build js,wasm
// File ini hanya akan dikompilasi ketika target adalah js/wasm.
// Go akan mengabaikan file ini pada build platform lain.

package main

import (
	"syscall/js"
)

// ===================================================================
// KONSEP KUNCI: "syscall/js" adalah jembatan antara Go dan JavaScript.
// Kita mengekspos fungsi Go ke JS dengan mendaftarkannya ke
// objek global "window" via js.Global().Set(...)
// ===================================================================

func main() {
	// Channel ini penting! Tanpanya, program Go akan langsung exit
	// setelah main() selesai, dan semua fungsi yang kita register
	// ke JS akan hilang. Channel ini membuat program "hidup terus".
	done := make(chan struct{}, 0)

	// Daftarkan semua fungsi Go agar bisa dipanggil dari JavaScript
	js.Global().Set("goGrayscale", js.FuncOf(grayscale))
	js.Global().Set("goInvert", js.FuncOf(invert))
	js.Global().Set("goAdjustBrightness", js.FuncOf(adjustBrightness))
	js.Global().Set("goEdgeDetect", js.FuncOf(edgeDetect))
	js.Global().Set("goPixelate", js.FuncOf(pixelate))

	// Beri tahu JS bahwa Go WASM sudah siap
	js.Global().Call("onGoWasmReady")

	<-done // Blokir selamanya agar WASM tetap "hidup"
}

// -------------------------------------------------------------------
// POLA UMUM: Setiap fungsi yang diekspos ke JS harus memiliki
// signature: func(this js.Value, args []js.Value) interface{}
//
// "args[0]" biasanya adalah Uint8ClampedArray dari ImageData canvas,
// yaitu array RGBA pixel: [R,G,B,A, R,G,B,A, ...]
// -------------------------------------------------------------------

// grayscale mengubah gambar menjadi hitam-putih menggunakan
// rumus luminance: Y = 0.299R + 0.587G + 0.114B
// Nilai koefisien ini bukan sembarangan — ini adalah bobot perceptual
// yang mencerminkan sensitivitas mata manusia terhadap warna.
func grayscale(this js.Value, args []js.Value) interface{} {
	pixelData := args[0] // Uint8ClampedArray
	length := pixelData.Length()

	// Salin data pixel dari JS ke Go slice untuk diproses
	data := make([]byte, length)
	js.CopyBytesToGo(data, pixelData)

	// Proses setiap pixel (4 byte per pixel: R, G, B, A)
	for i := 0; i < length; i += 4 {
		r := float64(data[i])
		g := float64(data[i+1])
		b := float64(data[i+2])
		// a := data[i+3] — Alpha channel tidak kita ubah

		// Perceptual luminance formula (ITU-R BT.601)
		gray := uint8(0.299*r + 0.587*g + 0.114*b)

		data[i] = gray   // R
		data[i+1] = gray // G
		data[i+2] = gray // B
		// data[i+3] dibiarkan — alpha tetap sama
	}

	// Salin kembali data yang sudah diproses ke JS array
	js.CopyBytesToJS(pixelData, data)
	return nil
}

// invert membalik setiap nilai warna: nilai baru = 255 - nilai_lama
// Ini secara matematis sama dengan XOR dengan 0xFF pada tiap channel.
func invert(this js.Value, args []js.Value) interface{} {
	pixelData := args[0]
	length := pixelData.Length()

	data := make([]byte, length)
	js.CopyBytesToGo(data, pixelData)

	for i := 0; i < length; i += 4 {
		data[i] = 255 - data[i]     // R
		data[i+1] = 255 - data[i+1] // G
		data[i+2] = 255 - data[i+2] // B
		// Alpha channel (i+3) tidak disentuh
	}

	js.CopyBytesToJS(pixelData, data)
	return nil
}

// adjustBrightness menambah atau mengurangi kecerahan.
// args[0] = pixel data, args[1] = nilai brightness (-100 sampai 100)
func adjustBrightness(this js.Value, args []js.Value) interface{} {
	pixelData := args[0]
	amount := args[1].Int() // Nilai antara -100 dan 100
	length := pixelData.Length()

	data := make([]byte, length)
	js.CopyBytesToGo(data, pixelData)

	for i := 0; i < length; i += 4 {
		// clamp memastikan nilai tetap di range 0-255
		data[i] = clamp(int(data[i]) + amount)
		data[i+1] = clamp(int(data[i+1]) + amount)
		data[i+2] = clamp(int(data[i+2]) + amount)
	}

	js.CopyBytesToJS(pixelData, data)
	return nil
}

// edgeDetect mendeteksi tepi/edge menggunakan Sobel operator.
// Ini adalah teknik yang dipakai dalam computer vision.
// Sobel bekerja dengan menghitung gradien (perubahan intensitas)
// di arah X dan Y, lalu menggabungkannya.
func edgeDetect(this js.Value, args []js.Value) interface{} {
	pixelData := args[0]
	width := args[1].Int()
	height := args[2].Int()

	data := make([]byte, pixelData.Length())
	js.CopyBytesToGo(data, pixelData)

	// Kernel Sobel untuk deteksi edge
	// Gx mendeteksi perubahan horizontal, Gy vertikal
	sobelX := [3][3]int{{-1, 0, 1}, {-2, 0, 2}, {-1, 0, 1}}
	sobelY := [3][3]int{{-1, -2, -1}, {0, 0, 0}, {1, 2, 1}}

	// Konversi ke grayscale dulu untuk perhitungan yang lebih bersih
	gray := make([]int, width*height)
	for i := 0; i < len(data); i += 4 {
		px := i / 4
		gray[px] = int(float64(data[i])*0.299 + float64(data[i+1])*0.587 + float64(data[i+2])*0.114)
	}

	result := make([]byte, len(data))

	// Aplikasikan kernel Sobel ke setiap pixel (kecuali border)
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			gx, gy := 0, 0
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					pixel := gray[(y+ky)*width+(x+kx)]
					gx += pixel * sobelX[ky+1][kx+1]
					gy += pixel * sobelY[ky+1][kx+1]
				}
			}
			// Magnitude gradien: sqrt(gx^2 + gy^2), lalu clamp ke 255
			magnitude := int(sqrtApprox(gx*gx + gy*gy))
			if magnitude > 255 {
				magnitude = 255
			}
			idx := (y*width + x) * 4
			result[idx] = byte(magnitude)
			result[idx+1] = byte(magnitude)
			result[idx+2] = byte(magnitude)
			result[idx+3] = 255 // Full opacity
		}
	}

	js.CopyBytesToJS(pixelData, result)
	return nil
}

// pixelate membuat efek pikselasi dengan membagi gambar menjadi blok-blok
// dan mengisi setiap blok dengan warna rata-rata dari piksel di dalamnya.
func pixelate(this js.Value, args []js.Value) interface{} {
	pixelData := args[0]
	width := args[1].Int()
	height := args[2].Int()
	blockSize := args[3].Int() // Ukuran blok piksel

	if blockSize < 2 {
		blockSize = 2
	}

	data := make([]byte, pixelData.Length())
	js.CopyBytesToGo(data, pixelData)
	result := make([]byte, len(data))

	for y := 0; y < height; y += blockSize {
		for x := 0; x < width; x += blockSize {
			// Hitung warna rata-rata dalam blok ini
			rSum, gSum, bSum, count := 0, 0, 0, 0
			for by := 0; by < blockSize && y+by < height; by++ {
				for bx := 0; bx < blockSize && x+bx < width; bx++ {
					idx := ((y+by)*width + (x+bx)) * 4
					rSum += int(data[idx])
					gSum += int(data[idx+1])
					bSum += int(data[idx+2])
					count++
				}
			}
			avgR := byte(rSum / count)
			avgG := byte(gSum / count)
			avgB := byte(bSum / count)

			// Isi seluruh blok dengan warna rata-rata
			for by := 0; by < blockSize && y+by < height; by++ {
				for bx := 0; bx < blockSize && x+bx < width; bx++ {
					idx := ((y+by)*width + (x+bx)) * 4
					result[idx] = avgR
					result[idx+1] = avgG
					result[idx+2] = avgB
					result[idx+3] = 255
				}
			}
		}
	}

	js.CopyBytesToJS(pixelData, result)
	return nil
}

// --- Helper functions ---

// clamp memastikan nilai integer berada dalam range [0, 255]
func clamp(v int) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
}

// sqrtApprox: implementasi integer square root (Newton's method)
// Menghindari import "math" yang bisa menambah ukuran binary
func sqrtApprox(n int) int {
	if n < 0 {
		return 0
	}
	if n == 0 {
		return 0
	}
	x := n
	for {
		x1 := (x + n/x) / 2
		if x1 >= x {
			return x
		}
		x = x1
	}
}
