package injector

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/cavaliercoder/go-cpio"
	"github.com/pkg/errors"
)

type RHCOSStreamReader struct {
	// underlying reader for the actual iso
	isoReader io.Reader
	// reader configured to record the ignition info
	infoReader io.Reader
	// reader to use for the customized content
	contentReader io.Reader

	// ignition file info
	haveIgnitionInfo  bool
	ignitionInfo      [24]byte // the actual 24 bytes from the underlying iso
	ignitionAreaStart int64
	ignitionAreaLen   int64
	// compressed ignition archive bytes reader
	ignition       io.Reader
	ignitionLength int
}

// header is inclusive of these bytes
const headerStart = 32744
const headerEnd = 32767
const coreISOMagic = "coreiso+"

func NewRHCOSStreamReader(isoReader io.Reader, ignitionContent string) (*RHCOSStreamReader, error) {
	ignitionBytes, err := ignitionImageArchive(ignitionContent)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create compressed ignition archive")
	}

	sr := &RHCOSStreamReader{
		isoReader:      isoReader,
		ignition:       bytes.NewReader(ignitionBytes),
		ignitionLength: len(ignitionBytes),
	}

	// set up limit reader for the space before the header
	b4HeaderReader := io.LimitReader(isoReader, headerStart-1)

	// set up a tee reader to write to the ignition info using a bytes buffer
	buf := bytes.NewBuffer(sr.ignitionInfo[:])
	ignitionInfoReader := io.TeeReader(io.LimitReader(isoReader, 24), buf)

	sr.infoReader = io.MultiReader(b4HeaderReader, ignitionInfoReader)

	return sr, nil
}

func (r *RHCOSStreamReader) transformIgnitionInfo() error {
	res := bytes.Compare(r.ignitionInfo[0:8], []byte(coreISOMagic))
	if res != 0 {
		return errors.New(fmt.Sprintf("Could not find magic string in object header (%s)", r.ignitionInfo[0:8]))
	}

	r.ignitionAreaStart = int64(binary.LittleEndian.Uint64(r.ignitionInfo[8:16]))
	r.ignitionAreaLen = int64(binary.LittleEndian.Uint64(r.ignitionInfo[16:24]))
	r.haveIgnitionInfo = true

	return nil
}

func (r *RHCOSStreamReader) Read(p []byte) (count int, err error) {
	if !r.haveIgnitionInfo {
		n, err := r.infoReader.Read(p)
		if err == io.EOF {
			err = r.transformIgnitionInfo()
		}
		return n, err
	}

	if r.contentReader == nil {
		beforeIgnitionReader := io.LimitReader(r.isoReader, r.ignitionAreaStart - headerEnd)
		// TODO figure out how to advance r.isoReader past the ignition while reading the ignition
		afterIgnitionReader :=
	}

	return r.contentReader.Read(p)
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
