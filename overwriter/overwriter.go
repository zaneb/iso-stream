package overwriter

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"sort"
)

type Range struct {
	Content io.ReadSeeker
	Offset  int64
	length  int64
}

func (r *Range) locationInRange(location int64) bool {
	if location >= r.Offset && location <= r.Offset+r.length {
		return true
	}
	return false
}

func (r *Range) setLength() error {
	size, err := r.Content.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	_, err = r.Content.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	r.length = size
	return nil
}

type rangeSlice []*Range

func (s rangeSlice) Len() int           { return len(s) }
func (s rangeSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s rangeSlice) Less(i, j int) bool { return s[i].Offset < s[j].Offset }

// Valid returns no error iff none of the content in the slice overlaps and the slice is sorted
func (s rangeSlice) valid() error {
	if len(s) == 0 {
		return nil
	}

	if !sort.IsSorted(s) {
		return fmt.Errorf("Cannot check the validity of an unsorted rangeSlice")
	}

	// check all but the last range for overlap with the following range
	for i := 0; i < len(s)-1; i++ {
		if s[i].Offset+s[i].length > s[i+1].Offset {
			return fmt.Errorf("Range list is invalid: range with offset %d and length %d overlaps with range at offset %d", s[i].Offset, s[i].length, s[i+1].Offset)
		}
	}

	return nil
}

type OverwriteReader struct {
	base   io.Reader
	ranges rangeSlice

	// location for the next read in the base reader
	location int64
	// current range index to read from (if the location is in the range)
	currentRangeIndex int
}

var _ io.Reader = &OverwriteReader{}

func NewOverwriteReader(baseReader io.Reader, overwriters ...*Range) (*OverwriteReader, error) {
	ranges := rangeSlice(overwriters)
	for _, r := range ranges {
		if err := r.setLength(); err != nil {
			return nil, err
		}
	}

	sort.Sort(ranges)
	if err := ranges.valid(); err != nil {
		return nil, err
	}

	return &OverwriteReader{
		base:   baseReader,
		ranges: ranges,
	}, nil
}

func (r *OverwriteReader) Read(p []byte) (int, error) {
	// does the current range reader include the current location?
	if r.readFromOverrideRange() {
		count, err := r.currentRange().Content.Read(p)
		fmt.Printf("\n\nRead %d bytes from override range:\n%x\n", count, p[0:count])

		// increment the underlying reader by count
		skipped, copyErr := io.CopyN(ioutil.Discard, r.base, int64(count))

		if err != nil && err != io.EOF {
			return count, err
		} else if err == io.EOF {
			// increment to the next range if we've reached the end of this one
			r.currentRangeIndex++
		}

		// Fail if we can't track the base reader with the override reader
		if copyErr != nil && copyErr != io.EOF {
			return count, fmt.Errorf("failed to advance underlying reader: %w", copyErr)
		}
		if skipped != int64(count) && copyErr != io.EOF {
			return count, fmt.Errorf("failed to advance underlying reader the correct number of bytes: expected %d, got %d", count, skipped)
		}

		r.location += int64(count)
		return count, nil
	}

	readBuf := p
	// if we have more ranges, don't read more than the distance to the next one
	if r.haveMoreRanges() && int64(len(p)) > r.currentRangeDistance() {
		readBuf = p[0:r.currentRangeDistance()]
	}

	count, err := r.base.Read(readBuf)
	r.location += int64(count)
	return count, err
}

func (r *OverwriteReader) readFromOverrideRange() bool {
	if !r.haveMoreRanges() {
		return false
	}
	return r.currentRange().locationInRange(r.location)
}

func (r *OverwriteReader) currentRange() *Range {
	return r.ranges[r.currentRangeIndex]
}

func (r *OverwriteReader) currentRangeDistance() int64 {
	if !r.haveMoreRanges() {
		return math.MaxInt64
	}
	return r.currentRange().Offset - r.location
}

func (r *OverwriteReader) haveMoreRanges() bool {
	return r.currentRangeIndex < len(r.ranges)
}
