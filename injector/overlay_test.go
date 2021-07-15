package injector

import (
	"io"
	"testing"

	"strings"
)

func TestOverlayCopy(t *testing.T) {
	testCases := []struct {
		Name     string
		Offset   int64
		Length   int64
		Expected string
	}{
		{
			Name:     "at start",
			Offset:   0,
			Length:   4,
			Expected: "overefghij",
		},
		{
			Name:     "in middle",
			Offset:   3,
			Length:   4,
			Expected: "abcoverhij",
		},
		{
			Name:     "at end",
			Offset:   6,
			Length:   4,
			Expected: "abcdefover",
		},
		{
			Name:     "across end",
			Offset:   8,
			Length:   4,
			Expected: "abcdefghover",
		},
		{
			Name:     "beyond end",
			Offset:   10,
			Length:   4,
			Expected: "abcdefghijover",
		},
		{
			Name:     "empty at start",
			Offset:   0,
			Length:   0,
			Expected: "abcdefghij",
		},
		{
			Name:     "empty in middle",
			Offset:   5,
			Length:   0,
			Expected: "abcdefghij",
		},
		{
			Name:     "empty at end",
			Offset:   9,
			Length:   0,
			Expected: "abcdefghij",
		},
		{
			Name:     "empty over end",
			Offset:   10,
			Length:   0,
			Expected: "abcdefghij",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			base := "abcdefghij"
			overlay := "overlay"

			reader, err := NewOverlayReader(strings.NewReader(base),
				Overlay{
					Reader: strings.NewReader(overlay),
					Offset: tc.Offset,
					Length: tc.Length,
				})
			if err != nil {
				t.Fatalf("Got unexpected creation error: %v", err)
			}

			output, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("Got unexpected read error: %v", err)
			}
			if string(output) != tc.Expected {
				t.Errorf("Got \"%s\", expected \"%s\"", string(output), tc.Expected)
			}

			newOffset, err := reader.Seek(3, io.SeekStart)
			if err != nil {
				t.Fatalf("Got unexpected seek error: %v", err)
			}
			if newOffset != 3 {
				t.Errorf("Unexpected seek result: %d", newOffset)
			}
			rangeOutput := make([]byte, 6)
			io.ReadFull(reader, rangeOutput)
			if string(rangeOutput) != tc.Expected[3:9] {
				t.Errorf("Got \"%s\", expected \"%s\"", string(rangeOutput), tc.Expected[3:9])
			}
		})
	}
}
