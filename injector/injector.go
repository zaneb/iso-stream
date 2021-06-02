package injector

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/carbonin/overreader"
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
	haveIgnitionInfo   bool
	ignitionInfo       *bytes.Buffer
	ignitionAreaStart  int64
	ignitionAreaLength int64

	// compressed ignition archive bytes reader
	ignition       io.ReadSeeker
	ignitionLength int64
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
		ignitionInfo:   new(bytes.Buffer),
		ignition:       bytes.NewReader(ignitionBytes),
		ignitionLength: int64(len(ignitionBytes)),
	}

	// set up limit reader for the space before the header
	b4HeaderReader := io.LimitReader(isoReader, headerStart)

	// set up a tee reader to write to the ignition info using a bytes buffer
	ignitionInfoReader := io.TeeReader(io.LimitReader(isoReader, 24), sr.ignitionInfo)

	sr.infoReader = io.MultiReader(b4HeaderReader, ignitionInfoReader)

	return sr, nil
}

func (r *RHCOSStreamReader) transformIgnitionInfo() error {
	infoBytes := make([]byte, 24)
	n, err := r.ignitionInfo.Read(infoBytes)
	if err != nil {
		return errors.Wrap(err, "failed to read embed area data from buffer")
	}
	if n != 24 {
		return errors.New(fmt.Sprintf("incorrect embed info size, expected 24, got %d", n))
	}

	res := bytes.Compare(infoBytes[0:8], []byte(coreISOMagic))
	if res != 0 {
		return errors.New(fmt.Sprintf("Could not find magic string in object header (%s)", infoBytes[0:8]))
	}

	r.ignitionAreaStart = int64(binary.LittleEndian.Uint64(infoBytes[8:16]))
	r.ignitionAreaLength = int64(binary.LittleEndian.Uint64(infoBytes[16:24]))
	r.haveIgnitionInfo = true

	if r.ignitionAreaLength < r.ignitionLength {
		return errors.New(fmt.Sprintf("ignition length (%d) exceeds embed area size (%d)", r.ignitionLength, r.ignitionAreaLength))
	}

	return nil
}

func (r *RHCOSStreamReader) Read(p []byte) (int, error) {
	var err error

	if !r.haveIgnitionInfo {
		n, err := r.infoReader.Read(p)
		if err == io.EOF {
			err = r.transformIgnitionInfo()
		}
		return n, err
	}

	if r.contentReader == nil {
		// Set the offset to the distance to the ignition taking into account that we've already read the system area
		ignitionRange := &overreader.Range{
			Content: r.ignition,
			Offset:  r.ignitionAreaStart - headerEnd - 1,
		}
		r.contentReader, err = overreader.NewReader(r.isoReader, ignitionRange)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to create overwrite reader")
		}
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
