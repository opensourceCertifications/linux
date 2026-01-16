package monitor

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"
)

const ntpEpochOffset = 2208988800

func ntpTimestamp(t time.Time, offset time.Duration) (sec uint32, frac uint32) {
	tt := t.Add(offset)

	unixSec := tt.Unix()
	if unixSec < 0 {
		panic(fmt.Sprintf("ntpTimestamp: negative unix time (%d) for %v", unixSec, tt))
	}

	// unixSec is now guaranteed >= 0
	ntpSec64 := uint64(unixSec) + ntpEpochOffset

	// Nanosecond() returns int in [0, 999,999,999], but we make that explicit for gosec.
	ns := tt.Nanosecond()
	if ns < 0 {
		panic(fmt.Sprintf("ntpTimestamp: negative nanoseconds (%d) for %v", ns, tt))
	}
	if ns >= 1_000_000_000 {
		// Should never happen, but documents the invariant for tooling/future edits
		panic(fmt.Sprintf("ntpTimestamp: nanoseconds out of range (%d) for %v", ns, tt))
	}
	nsec := uint64(ns)

	ntpFrac64 := (nsec << 32) / 1_000_000_000
	if ntpFrac64 > 0xFFFFFFFF {
		panic(fmt.Sprintf("ntpTimestamp: ntpFrac overflow: %d", ntpFrac64))
	}

	// Convert seconds with an explicit “era wrap” mask, but make it obvious it's in range.
	sec64 := ntpSec64 & 0xFFFFFFFF
	if sec64 > 0xFFFFFFFF {
		// logically impossible due to mask; keeps static analyzers happy if they track the branch
		panic(fmt.Sprintf("ntpTimestamp: masked seconds out of range: %d", sec64))
	}

	sec32 := uint32(sec64)
	frac32 := uint32(ntpFrac64)
	return sec32, frac32
}

// ServeFakeNTP serves a fake NTP server on listenAddr (e.g., ":123") that
func ServeFakeNTP(ctx context.Context, listenAddr string, offset time.Duration) error {
	pc, err := net.ListenPacket("udp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen udp %s: %w", listenAddr, err)
	}
	defer func() {
		if cerr := pc.Close(); cerr != nil && ctx.Err() == nil {
			log.Printf("fake ntp: close: %v", cerr)
		}
	}()

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = pc.Close()
		case <-done:
		}
	}()
	defer close(done)

	buf := make([]byte, 1024)

	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("read: %w", err)
		}
		if n < 48 {
			continue
		}

		req := buf[:n]
		resp := make([]byte, 48)

		// LI=0, VN=4, Mode=4
		resp[0] = (0 << 6) | (4 << 3) | 4
		resp[1] = 2
		resp[2] = 6
		resp[3] = 0xEC // -20

		binary.BigEndian.PutUint32(resp[4:8], 0)
		binary.BigEndian.PutUint32(resp[8:12], 0)
		copy(resp[12:16], []byte("LOCL"))

		now := time.Now()

		refSec, refFrac := ntpTimestamp(now.Add(-1*time.Second), offset)
		binary.BigEndian.PutUint32(resp[16:20], refSec)
		binary.BigEndian.PutUint32(resp[20:24], refFrac)

		copy(resp[24:32], req[40:48]) // originate = client transmit

		recvSec, recvFrac := ntpTimestamp(now, offset)
		binary.BigEndian.PutUint32(resp[32:36], recvSec)
		binary.BigEndian.PutUint32(resp[36:40], recvFrac)

		txSec, txFrac := ntpTimestamp(now, offset)
		binary.BigEndian.PutUint32(resp[40:44], txSec)
		binary.BigEndian.PutUint32(resp[44:48], txFrac)

		_, _ = pc.WriteTo(resp, addr)
	}
}
