package main

import (
	"bytes"
	"fmt"
	"github.com/twmb/murmur3"
	"hash"
	"hash/adler32"
	"hash/crc32"
	"math/rand"
	"sync"
	"unsafe"
)

const (
	prime64a = 11400714785074694791
	prime64b = 14029467366897019727
	prime64c = 1609587929392839161
	prime64d = 9650029242287828579
	prime64e = 2870177450012600261

	m1 = 16877499708836156737
	m2 = 2820277070424839065
	m3 = 9497967016996688599
	m4 = 15839092249703872147

	flagCRCHash    = "crcHash"
	flagMurMurHash = "murHash"
	flagAdlerHash  = "adlerHash"
	flagXXHash     = "xxHash"
	flagMemHash    = "memHash"
)

func hashSelector(b string) (func(b []byte, seed uint64) []byte, error) {
	switch b {
	case flagCRCHash:
		return crcHash, nil
	case flagXXHash:
		return xxHash, nil
	case flagAdlerHash:
		return adlerHash, nil
	case flagMemHash:
		return memHash, nil
	case flagMurMurHash:
		return murHash, nil
	default:
		return nil, fmt.Errorf("invalid param hash (%v)", b)
	}
}

var padding = bytes.Repeat([]byte("0"), 4)

var hashKey = [4]uintptr{
	uintptr(rand.Int63()) | 1,
	uintptr(rand.Int63()) | 1,
	uintptr(rand.Int63()) | 1,
	uintptr(rand.Int63()) | 1,
}

var crcPool = sync.Pool{
	New: func() interface{} {
		return crc32.New(crc32.MakeTable(crc32.Castagnoli))
	},
}

var adlerPool = sync.Pool{
	New: func() interface{} {
		return adler32.New()
	},
}

var murPool = sync.Pool{
	New: func() interface{} {
		return murmur3.New64()
	},
}

// mur64hash
func murHash(b []byte, seed uint64) []byte {
	return genericHash(b, seed, &murPool)
}

// crc32hash
func crcHash(b []byte, seed uint64) []byte {
	return append(genericHash(b, seed, &crcPool), padding...)
}

// adler32hash
func adlerHash(b []byte, seed uint64) []byte {
	return append(genericHash(b, seed, &adlerPool), padding...)
}

func genericHash(b []byte, seed uint64, pool *sync.Pool) []byte {
	c := pool.Get().(hash.Hash)
	c.Reset()
	_, _ = c.Write(u642b(seed ^ 0x36))
	_, _ = c.Write(b)
	sum1 := c.Sum(nil)
	c.Reset()
	_, _ = c.Write(u642b(seed ^ 0x50))
	_, _ = c.Write(sum1)
	sum1 = c.Sum(nil)
	pool.Put(c)
	return sum1
}

// xxHash, modified
func xxHash(b []byte, seed uint64) []byte {
	n := len(b)
	var h64 uint64

	if n >= 32 {
		v1 := seed + prime64a + prime64b
		v2 := seed + prime64b
		v3 := seed
		v4 := seed - prime64a
		p := 0
		for n := n - 32; p <= n; p += 32 {
			sub := b[p:][:32] //BCE hint for compiler
			v1 = rol31(v1+u64(sub[:])*prime64b) * prime64a
			v2 = rol31(v2+u64(sub[8:])*prime64b) * prime64a
			v3 = rol31(v3+u64(sub[16:])*prime64b) * prime64a
			v4 = rol31(v4+u64(sub[24:])*prime64b) * prime64a
		}

		h64 = rol1(v1) + rol7(v2) + rol12(v3) + rol18(v4)

		v1 *= prime64b
		v2 *= prime64b
		v3 *= prime64b
		v4 *= prime64b

		h64 = (h64^(rol31(v1)*prime64a))*prime64a + prime64d
		h64 = (h64^(rol31(v2)*prime64a))*prime64a + prime64d
		h64 = (h64^(rol31(v3)*prime64a))*prime64a + prime64d
		h64 = (h64^(rol31(v4)*prime64a))*prime64a + prime64d

		h64 += uint64(n)

		b = b[p:]
		n -= p
	} else {
		h64 = seed + prime64e + uint64(n)
	}

	p := 0
	for n := n - 8; p <= n; p += 8 {
		sub := b[p : p+8]
		h64 ^= rol31(u64(sub)*prime64b) * prime64a
		h64 = rol27(h64)*prime64a + prime64d
	}
	if p+4 <= n {
		sub := b[p : p+4]
		h64 ^= uint64(u32(sub)) * prime64a
		h64 = rol23(h64)*prime64b + prime64c
		p += 4
	}
	for ; p < n; p++ {
		h64 ^= uint64(b[p]) * prime64e
		h64 = rol11(h64) * prime64a
	}

	h64 ^= h64 >> 33
	h64 *= prime64b
	h64 ^= h64 >> 29
	h64 *= prime64c
	h64 ^= h64 >> 32

	return u642b(h64)
}

// _memHash, modified
func _memHash(p unsafe.Pointer, seed, s uintptr) uintptr {
	h := uint64(seed + s*hashKey[0])
tail:
	switch {
	case s == 0:
	case s < 4:
		h ^= uint64(*(*byte)(p))
		h ^= uint64(*(*byte)(add(p, s>>1))) << 8
		h ^= uint64(*(*byte)(add(p, s-1))) << 16
		h = rol31(h*m1) * m2
	case s <= 8:
		h ^= uint64(r32(p))
		h ^= uint64(r32(add(p, s-4))) << 32
		h = rol31(h*m1) * m2
	case s <= 16:
		h ^= r64(p)
		h = rol31(h*m1) * m2
		h ^= r64(add(p, s-8))
		h = rol31(h*m1) * m2
	case s <= 32:
		h ^= r64(p)
		h = rol31(h*m1) * m2
		h ^= r64(add(p, 8))
		h = rol31(h*m1) * m2
		h ^= r64(add(p, s-16))
		h = rol31(h*m1) * m2
		h ^= r64(add(p, s-8))
		h = rol31(h*m1) * m2
	default:
		v1 := h
		v2 := uint64(seed * hashKey[1])
		v3 := uint64(seed * hashKey[2])
		v4 := uint64(seed * hashKey[3])
		for s >= 32 {
			v1 ^= r64(p)
			v1 = rol31(v1*m1) * m2
			p = add(p, 8)
			v2 ^= r64(p)
			v2 = rol31(v2*m2) * m3
			p = add(p, 8)
			v3 ^= r64(p)
			v3 = rol31(v3*m3) * m4
			p = add(p, 8)
			v4 ^= r64(p)
			v4 = rol31(v4*m4) * m1
			p = add(p, 8)
			s -= 32
		}
		h = v1 ^ v2 ^ v3 ^ v4
		goto tail
	}

	h ^= h >> 29
	h *= m3
	h ^= h >> 32
	return uintptr(h)
}

func memHash(b []byte, seed uint64) []byte {
	if len(b) == 0 {
		return u642b(seed)
	}
	// For 64-bit architectures, we use the memHash directly. Otherwise,
	// we use two parallel memHash on the lower and upper 32 bits.
	if unsafe.Sizeof(uintptr(0)) == 8 {
		return u642b(uint64(_memHash(unsafe.Pointer(&b[0]), uintptr(seed), uintptr(len(b)))))
	}
	lo := _memHash(unsafe.Pointer(&b[0]), uintptr(seed), uintptr(len(b)))
	hi := _memHash(unsafe.Pointer(&b[0]), uintptr(seed>>32), uintptr(len(b)))
	return u642b(uint64(hi)<<32 | uint64(lo))
}

// utils

func add(p unsafe.Pointer, x uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

func r64(p unsafe.Pointer) uint64 {
	q := (*[8]byte)(p)
	return uint64(q[0]) | uint64(q[1])<<8 | uint64(q[2])<<16 | uint64(q[3])<<24 |
		uint64(q[4])<<32 | uint64(q[5])<<40 | uint64(q[6])<<48 | uint64(q[7])<<56
}

func r32(p unsafe.Pointer) uint32 {
	q := (*[4]byte)(p)
	return uint32(q[0]) | uint32(q[1])<<8 | uint32(q[2])<<16 | uint32(q[3])<<24
}

func u64(buf []byte) uint64 {
	return uint64(buf[0]) | uint64(buf[1])<<8 | uint64(buf[2])<<16 |
		uint64(buf[3])<<24 | uint64(buf[4])<<32 | uint64(buf[5])<<40 | uint64(buf[6])<<48 | uint64(buf[7])<<56
}

func u32(buf []byte) uint32 {
	return uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
}

func rol1(u uint64) uint64 {
	return u<<1 | u>>63
}

func rol7(u uint64) uint64 {
	return u<<7 | u>>57
}

func rol11(u uint64) uint64 {
	return u<<11 | u>>53
}

func rol12(u uint64) uint64 {
	return u<<12 | u>>52
}

func rol18(u uint64) uint64 {
	return u<<18 | u>>46
}

func rol23(u uint64) uint64 {
	return u<<23 | u>>41
}

func rol27(u uint64) uint64 {
	return u<<27 | u>>37
}

func rol31(u uint64) uint64 {
	return u<<31 | u>>33
}

func u642b(x uint64) []byte {
	return []byte{byte(x >> 0), byte(x >> 8), byte(x >> 16), byte(x >> 24),
		byte(x >> 32), byte(x >> 40), byte(x >> 48), byte(x >> 56)}
}

