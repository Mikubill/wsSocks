package main

import (
	"bytes"
	"encoding/hex"
	"golang.org/x/sys/cpu"
	"hash/crc64"
	"sync/atomic"
	"time"
)

const interval = 60

var (
	crcTable   = crc64.MakeTable(crc64.ECMA)
	local      = solve(0)
	hashWorker = memHash
	hashFlag   = flagMemHash
)

func init() {
	// choice a alg for hash
	if cpu.X86.HasSSE42 {
		hashWorker = crcHash
		hashFlag = flagCRCHash
	} else {
		// add AES
		hashWorker = memHash
		hashFlag = flagMemHash
	}
}

func generateCode(p []byte, h func([]byte, uint64) []byte) []byte {
	return h(p, atomic.LoadUint64(&local))
}

func validateCode(p []byte, q []byte, h func([]byte, uint64) []byte) bool {
	if bytes.Equal(q, h(p, atomic.LoadUint64(&local))) {
		return true
	}
	if bytes.Equal(q, h(p, solve(0))) {
		return true
	}
	if bytes.Equal(q, h(p, solve(-1))) {
		return true
	}
	if bytes.Equal(q, h(p, solve(+1))) {
		return true
	}
	return false
}

func validateStringCode(p, q string, h func([]byte, uint64) []byte) bool {
	s, err := hex.DecodeString(q)
	if err != nil {
		return false
	}
	return validateCode([]byte(p), s, h)
}

func solve(delta int64) uint64 {
	v := time.Now().Unix()/interval + delta
	b := []byte{byte(v), byte(v >> 8), byte(v >> 16),
		byte(v >> 24), byte(v >> 32), byte(v >> 40), byte(v >> 48), byte(v >> 56)}

	bs := crc64.New(crcTable)
	_, _ = bs.Write(authKey)
	_, _ = bs.Write(b)

	return bs.Sum64()
}

func timeUpdater() {
	for {
		atomic.StoreUint64(&local, solve(0))
		time.Sleep(5 * time.Second)
	}
}
