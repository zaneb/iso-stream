package injector

import (
	"compress/gzip"
	"fmt"
	"io"

	"github.com/cavaliercoder/go-cpio"
	"github.com/pkg/errors"
)

type RHCOSStreamReader struct {
	isoReader io.Reader
	// ignition file info
	haveIgnitionInfo    bool
	ignitionInfo        [16]byte // the actual 16 bytes from the underlying iso
	ignitionOffsetBytes int64
	ignitionLengthBytes int64
	// current location in the underlying iso
	location int64
	// put magic string here in case we span read calls
	magic [8]byte
	// compressed ignition archive bytes reader
	ignition io.Reader
	// have we read through the entire ignition
	ignitionEOF bool
}

const headerStart = 32744
const headerEnd = 32767
const coreISOMagic = "coreiso+"

func NewRHCOSStreamReader(isoReader io.Reader, ignitionContent string) (*RHCOSStreamReader, error) {
	ignitionBytes, err := ignitionImageArchive(ignitionContent)
	if err != nil {
		return nil, errors.Wrap("Failed to create compressed ignition archive")
	}
	return &RHCOSStreamReader{
		isoReader: isoReader,
		ignition:  bytes.NewReader(ignitionBytes),
	}, nil
}

func (r *RHCOSStreamReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	//cases
	//  don't have ignition info
	//    will encounter header
	//      read header bytes into ignitionInfo
	//    won't encounter header
	//      read len(p)
	//    finished reading header
	//      evaluate ignitionInfo and set offset and length (maybe use start and end here?)
	//  have ignition info
	//    will encounter ignition space
	//      read from ignition reader
	//      also increment location and nop read from isoReader
	//  ...
}

func ignitionImageArchive(ignitionConfig string) ([]byte, error) {
	ignitionBytes := []byte(ignitionConfig)

	// Create CPIO archive
	archiveBuffer := new(bytes.Buffer)
	cpioWriter := cpio.NewWriter(archiveBuffer)
	if err := cpioWriter.WriteHeader(&cpio.Header{Name: "config.ign", Mode: 0o100_644, Size: int64(len(ignitionBytes))}); err != nil {
		return nil, errors.Wrap(err, "Failed to write CPIO header")
	}
	if _, err := cpioWriter.Write(ignitionBytes); err != nil {

		return nil, errors.Wrap(err, "Failed to write CPIO archive")
	}
	if err := cpioWriter.Close(); err != nil {
		return nil, errors.Wrap(err, "Failed to close CPIO archive")
	}

	// Run gzip compression
	compressedBuffer := new(bytes.Buffer)
	gzipWriter := gzip.NewWriter(compressedBuffer)
	if _, err := gzipWriter.Write(archiveBuffer.Bytes()); err != nil {
		return nil, errors.Wrap(err, "Failed to gzip ignition config")
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, errors.Wrap(err, "Failed to gzip ignition config")
	}

	return compressedBuffer.Bytes(), nil
}
