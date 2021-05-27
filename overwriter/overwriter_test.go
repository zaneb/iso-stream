package overwriter

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"
)

// every character in the alphabet repeated 5 times
const testBaseString = `aaaaabbbbbcccccdddddeeeeefffffggggghhhhhiiiiijjjjjkkkkklllllmmmmmnnnnnooooopppppqqqqqrrrrrssssstttttuuuuuvvvvvwwwwwxxxxxyyyyyzzzzz`

func testReader(expected []byte, base io.Reader, ranges ...*Range) error {
	r, err := NewOverwriteReader(base, ranges...)
	if err != nil {
		return fmt.Errorf("Failed to create overwrite reader: %w", err)
	}
	res, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("Failed to read from overwrite reader: %v", err)
	}
	if !bytes.Equal(res, expected) {
		return fmt.Errorf("Overwrite reader returned incorrect data: expected following to match:\n%s\n%s", expected, res)
	}
	return nil
}

func TestOverwrites(t *testing.T) {
	base := strings.NewReader(testBaseString)
	range1 := &Range{
		Content: strings.NewReader("12345"),
		Offset:  5,
	}
	range2 := &Range{
		Content: strings.NewReader("0000000000"),
		Offset:  25,
	}

	expected := []byte(`aaaaa12345cccccdddddeeeee0000000000hhhhhiiiiijjjjjkkkkklllllmmmmmnnnnnooooopppppqqqqqrrrrrssssstttttuuuuuvvvvvwwwwwxxxxxyyyyyzzzzz`)
	err := testReader(expected, base, range1, range2)
	if err != nil {
		t.Fatalf("Happy path test failed: %v", err)
	}
}

func TestOverwritesUnordered(t *testing.T) {
	base := strings.NewReader(testBaseString)
	range1 := &Range{
		Content: strings.NewReader("---"),
		Offset:  50,
	}
	range2 := &Range{
		Content: strings.NewReader("0000000000"),
		Offset:  25,
	}
	range3 := &Range{
		Content: strings.NewReader("12345"),
		Offset:  5,
	}

	expected := []byte(`aaaaa12345cccccdddddeeeee0000000000hhhhhiiiiijjjjj---kklllllmmmmmnnnnnooooopppppqqqqqrrrrrssssstttttuuuuuvvvvvwwwwwxxxxxyyyyyzzzzz`)
	err := testReader(expected, base, range1, range2, range3)
	if err != nil {
		t.Fatalf("Unordered test failed: %v", err)
	}
}

func TestAdjacentOverwrites(t *testing.T) {
	base := strings.NewReader(testBaseString)
	range1 := &Range{
		Content: strings.NewReader("12345"),
		Offset:  5,
	}
	range2 := &Range{
		Content: strings.NewReader("0000000000"),
		Offset:  10,
	}

	expected := []byte(`aaaaa123450000000000eeeeefffffggggghhhhhiiiiijjjjjkkkkklllllmmmmmnnnnnooooopppppqqqqqrrrrrssssstttttuuuuuvvvvvwwwwwxxxxxyyyyyzzzzz`)
	err := testReader(expected, base, range1, range2)
	if err != nil {
		t.Fatalf("Adjacent test failed: %v", err)
	}
}

func TestOverwriteAtStart(t *testing.T) {
	base := strings.NewReader(testBaseString)
	range1 := &Range{
		Content: strings.NewReader("123456789"),
		Offset:  0,
	}
	expected := []byte(`123456789bcccccdddddeeeeefffffggggghhhhhiiiiijjjjjkkkkklllllmmmmmnnnnnooooopppppqqqqqrrrrrssssstttttuuuuuvvvvvwwwwwxxxxxyyyyyzzzzz`)
	err := testReader(expected, base, range1)
	if err != nil {
		t.Fatalf("Adjacent test failed: %v", err)
	}
}

func TestOverwriteAtEnd(t *testing.T) {
	base := strings.NewReader(testBaseString)
	range1 := &Range{
		Content: strings.NewReader("123456789"),
		Offset:  121,
	}
	expected := []byte(`aaaaabbbbbcccccdddddeeeeefffffggggghhhhhiiiiijjjjjkkkkklllllmmmmmnnnnnooooopppppqqqqqrrrrrssssstttttuuuuuvvvvvwwwwwxxxxxy123456789`)
	err := testReader(expected, base, range1)
	if err != nil {
		t.Fatalf("At end test failed: %v", err)
	}
}

// TODO is this the behavior we want?
// Alternative would be to not print the extra stuff
func TestOverwriteOverEnd(t *testing.T) {
	base := strings.NewReader(testBaseString)
	range1 := &Range{
		Content: strings.NewReader("123456789----------"),
		Offset:  121,
	}
	expected := []byte(`aaaaabbbbbcccccdddddeeeeefffffggggghhhhhiiiiijjjjjkkkkklllllmmmmmnnnnnooooopppppqqqqqrrrrrssssstttttuuuuuvvvvvwwwwwxxxxxy123456789----------`)
	err := testReader(expected, base, range1)
	if err != nil {
		t.Fatalf("Over end test failed: %v", err)
	}
}

func TestOverlapDetect(t *testing.T) {
	base := strings.NewReader(testBaseString)
	range1 := &Range{
		Content: strings.NewReader("12345"),
		Offset:  5,
	}
	range2 := &Range{
		Content: strings.NewReader("0000000000"),
		Offset:  7,
	}

	_, err := NewOverwriteReader(base, range1, range2)
	if err == nil {
		t.Fatalf("Expected overlapping ranges to fail")
	}
}
