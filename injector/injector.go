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
	haveIgnitionInfo  bool
	ignitionInfo      [24]byte // the actual 24 bytes from the underlying iso
	ignitionAreaStart int64
	ignitionAreaEnd   int64
	// current location in the underlying iso
	location int64
	// compressed ignition archive bytes reader
	ignition       io.Reader
	ignitionLength int
	// have we read through the entire ignition
	ignitionEOF bool
}

// header is inclusive of these bytes
const headerStart = 32744
const headerEnd = 32767
const coreISOMagic = "coreiso+"

func NewRHCOSStreamReader(isoReader io.Reader, ignitionContent string) (*RHCOSStreamReader, error) {
	ignitionBytes, err := ignitionImageArchive(ignitionContent)
	if err != nil {
		return nil, errors.Wrap("Failed to create compressed ignition archive")
	}
	return &RHCOSStreamReader{
		isoReader:      isoReader,
		ignition:       bytes.NewReader(ignitionBytes),
		ignitionLength: len(ignitionBytes),
	}, nil
}

func (r *RHCOSStreamReader) readFromISO(p []byte) (int, error) {
	count, err := r.isoReader.Read(p)
	r.location += count
	return count, err
}

func (r *RHCOSStreamReader) transformIgnitionInfo() error {
	res := bytes.Compare(r.ignitionInfo[0:8], []byte(coreISOMagic))
	if res != 0 {
		return errors.New(fmt.Sprintf("Could not find magic string in object header (%s)", headerString[0:8]))
	}

	r.ignitionAreaStart = int64(binary.LittleEndian.Uint64(r.ignitionInfo[8:16]))
	r.ignitionAreaEnd = r.ignitionAreaStart + int64(binary.LittleEndian.Uint64(r.ignitionInfo[16:24]))

	return nil
}

// Will a read of size `size` intersect with the header info given the current location?
func (r *RHCOSStreamReader) intersectsHeader(size int) bool {
	startLoc := r.location
	endLoc := startLoc + size

	if endLoc < headerStart || startLoc > headerEnd {
		return false
	}

	return true
}

func (r *RHCOSStreamReader) Read(p []byte) (int, error) {
	bufLen := len(p)
	if bufLen == 0 {
		return 0, nil
	}

	startLoc := r.location
	endLoc := startLoc + bufLen

	// location where the next byte should be written in p
	writeStart := func() int {
		return r.location - startLoc
	}

	if !r.haveIgnitionInfo {
		if !r.intersectsHeader(bufLen) {
			// entire read is before the header
			// (1)
			return r.readFromISO(p)
		} else {
			// if we are going to read just to or past the end of the header, handle the header first, then move on
			// this is so we will be able to find the start of the ignition area in case the one read would cover both
			if endLoc >= headerEnd {
				// (3), (4)
				// read from start location to end of header into "p"
				sliceEnd := headerEnd - startLoc + 1
				pSlice := p[writeStart():sliceEnd]

				count, err := r.readFromISO(pSlice)
				if count != len(pSlice) {
					// were not able to read entire header
					// return and we will address this in the next call if the error is recoverable
					return count, err
				}

				// now have the entire header between r.ignitionInfo and pSlice
				// set r.ignitionInfo properly
				// we know the end of pSlice is the end of the header
				// So either pSlice contains the entire header or only the end

				pSrc := pSlice
				if startLoc < headerStart {
					pSrc = pSlice[(headerStart - startLoc):]
				}
				infoDst := ignitionInfo
				if startLoc > headerStart {
					infoDst = ignitionInfo[startLoc-headerStart:]
				}
				copy(infoDst, pSrc)
			} else {
				// (2), (6)
				// won't finish the header in this call to Read
			}
		}
	}

	// separate case where we might have read the header as a part of this read
	if r.haveIgnitionInfo {
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
