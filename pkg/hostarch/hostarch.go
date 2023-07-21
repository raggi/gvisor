// Copyright 2021 The gVisor Authors.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package hostarch contains host arch address operations for user memory.
package hostarch

import (
	"math/bits"

	"golang.org/x/sys/unix"
)

var (
	// PageSize is the system page size.
	// arm64 support 4K/16K/64K page size,
	// which can be get by unix.Getpagesize().
	PageSize int

	// PageShift is the binary log of the system page size.
	PageShift int
)

func init() {
	PageSize = unix.Getpagesize()
	// Make an assumption on a system where unix.Getpagesize() returns 0 that we
	// have a 4K page size.
	if PageSize == 0 {
		PageSize = 4096
	}

	PageShift = bits.Len(uint(PageSize)) - 1

	// Safety check.
	if 1<<PageShift != PageSize {
		panic("1<<PageShift != PageSize")
	}
}
