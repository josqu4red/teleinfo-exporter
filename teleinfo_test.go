package main

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestParseFrame(t *testing.T) {
	input := []byte("\nADCO 031762270346 @\r\nOPTARIF BASE 0\r\nISOUSC 30 9\r\nBASE 007640930 (\r\nPTEC TH.. $\r\nIINST 002 Y\r\nIMAX 090 H\r\nPAPP 00390 -\r\nHHPHC A ,\r\nMOTDETAT 000000 B\r\x03")
	output := &TeleinfoFrame{
		Index:               7640930,
		IntensityInstant:    2,
		IntensityMax:        90,
		IntensitySubscribed: 30,
		PowerApparent:       390,
		CollectionTime:      time.Duration(0),
	}

	frame, err := parseFrame(input)
	if err != nil {
		t.Errorf("err should be nil, got: %w", err)
	}

	if !cmp.Equal(frame, output) {
		t.Errorf("frame should be %v, got: %v", output, frame)
	}
}
