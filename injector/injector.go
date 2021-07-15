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

const headerLen = 24
const headerStart = (1 << 15) - headerLen
const coreISOMagic = "coreiso+"

func NewRHCOSStreamReader(isoReader io.ReadSeeker, ignitionContent string) (io.ReadSeeker, error) {
	ignitionReader, err := IgnitionImageArchive(ignitionContent)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create compressed ignition archive")
	}

	areaStart, areaLength, err := CoreOSIgnitionArea(isoReader)
	if err != nil {
		return nil, err
	}

	ignitionLength := uint64(ignitionReader.Size())
	if areaLength < ignitionLength {
		return nil, errors.New(fmt.Sprintf("ignition length (%d) exceeds embed area size (%d)", ignitionLength, areaLength))
	}

	ignitionOverlay := Overlay{
		Reader: ignitionReader,
		Offset: int64(areaStart),
		Length: ignitionReader.Size(),
	}
	contentReader, err := NewOverlayReader(isoReader, ignitionOverlay)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create overwrite reader")
	}

	return contentReader, nil
}

func CoreOSIgnitionArea(isoReader io.ReadSeeker) (start, length uint64, err error) {
	if _, err := isoReader.Seek(headerStart, io.SeekStart); err != nil {
		return 0, 0, err
	}
	infoBytes := make([]byte, headerLen)
	isoReader.Read(infoBytes)
	if _, err := isoReader.Seek(0, io.SeekStart); err != nil {
		return 0, 0, err
	}

	if bytes.Compare(infoBytes[0:8], []byte(coreISOMagic)) != 0 {
		return 0, 0, fmt.Errorf("Could not find magic string in object header (%s)", infoBytes[0:8])
	}

	start = binary.LittleEndian.Uint64(infoBytes[8:16])
	length = binary.LittleEndian.Uint64(infoBytes[16:24])

	return
}

func IgnitionImageArchive(ignitionConfig string) (*bytes.Reader, error) {
	ignitionBytes := []byte(ignitionConfig)

	// Run gzip compression
	compressedBuffer := new(bytes.Buffer)
	gzipWriter := gzip.NewWriter(compressedBuffer)

	// Create CPIO archive
	cpioWriter := cpio.NewWriter(gzipWriter)
	if err := cpioWriter.WriteHeader(&cpio.Header{
		Name: "config.ign",
		Mode: 0o100_644,
		Size: int64(len(ignitionBytes)),
	}); err != nil {
		return nil, errors.Wrap(err, "Failed to write CPIO header")
	}
	if _, err := cpioWriter.Write(ignitionBytes); err != nil {
		return nil, errors.Wrap(err, "Failed to write CPIO archive")
	}
	if err := cpioWriter.Close(); err != nil {
		return nil, errors.Wrap(err, "Failed to close CPIO archive")
	}

	if err := gzipWriter.Close(); err != nil {
		return nil, errors.Wrap(err, "Failed to gzip ignition config")
	}

	return bytes.NewReader(compressedBuffer.Bytes()), nil
}
