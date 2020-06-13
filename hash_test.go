package main

import (
	"crypto/rand"
	"fmt"
	"testing"
)

var testBytes []byte
var grs int64

func BenchmarkHash(b *testing.B) {
	sizes := []int64{32, 64, 128, 256, 512, 1024, 10240, 30720}
	for _, grs = range sizes {
		testBytes = make([]byte, grs)
		readN, err := rand.Read(testBytes)
		if int64(readN) != grs {
			panic(fmt.Sprintf("expect %d but got %d", grs, readN))
		}
		if err != nil {
			panic(err)
		}
		b.Run(fmt.Sprintf("MapHash64-%v-%d", flagMemHash, grs), BenchmarkMapHash64)
		b.Run(fmt.Sprintf("MurMur64-%v-%d", flagMurMurHash, grs), BenchmarkMurMur64)
		b.Run(fmt.Sprintf("Adler32-%v-%d", flagAdlerHash, grs), BenchmarkAdler32)
		b.Run(fmt.Sprintf("Crc32-%v-%d", flagCRCHash, grs), BenchmarkCrc32)
		b.Run(fmt.Sprintf("xxHash64-%v-%d", flagXXHash, grs), BenchmarkxxHash64)
		fmt.Println()
	}
}

func BenchmarkMurMur64(b *testing.B) {
	b.SetBytes(grs)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		murHash(testBytes, 0)
	}
	//print(len(murHash(testBytes, 0)))
}

func BenchmarkMapHash64(b *testing.B) {
	b.SetBytes(grs)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		memHash(testBytes, 0)
	}
	//print(len(memHash(testBytes, 0)))
}

func BenchmarkAdler32(b *testing.B) {
	b.SetBytes(grs)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		adlerHash(testBytes, 0)
	}
	//print(len(adlerHash(testBytes, 0)))
}

func BenchmarkCrc32(b *testing.B) {
	b.SetBytes(grs)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		crcHash(testBytes, 0)
	}
	//print(len(crcHash(testBytes, 0)))
}

func BenchmarkxxHash64(b *testing.B) {
	b.SetBytes(grs)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		xxHash(testBytes, 0)
	}
	//print(len(xxHash(testBytes, 0)))
}
