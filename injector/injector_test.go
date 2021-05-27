package injector

import (
	"io"
	"io/ioutil"
	"os"
	"testing"
)

const isoPath = `../isos/rhcos-4.6.1-x86_64-live.x86_64.iso`
const ignitionStart = 8302592
const ignitionLength = 262144

func TestEmptyIgntitionAndStreamSize(t *testing.T) {
	iso, err := os.Open(isoPath)
	if err != nil {
		t.Fatalf("Failed to open iso: %v", err)
	}
	r, err := NewRHCOSStreamReader(iso, "")
	if err != nil {
		t.Fatalf("Failed to create stream reader: %v", err)
	}

	count, err := io.Copy(ioutil.Discard, r)
	if err != nil {
		t.Fatalf("Failed to copy from stream reader: %v", err)
	}

	info, err := iso.Stat()
	if err != nil {
		t.Fatalf("Failed to stat test iso (%s): %v", isoPath, err)
	}
	if count != info.Size() {
		t.Fatalf("Failed to read entire iso file, expected %d bytes, got %d", info.Size(), count)
	}

	if r.ignitionAreaStart != ignitionStart {
		t.Fatalf("Read incorrect ignition start, expected %d, got %d", ignitionStart, r.ignitionAreaStart)
	}
	if r.ignitionAreaLength != ignitionLength {
		t.Fatalf("Read incorrect ignition length, expected %d, got %d", ignitionLength, r.ignitionAreaLength)
	}
}
