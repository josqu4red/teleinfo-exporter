package main

import (
	"bufio"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	ms "github.com/mitchellh/mapstructure"
	"github.com/tarm/serial"
)

// map[ADCO:000000000000 BASE:007619994 HHPHC:A IINST:002 IMAX:090 ISOUSC:30 MOTDETAT:000000 OPTARIF:BASE PAPP:00500 PTEC:TH..]

type Teleinfo struct {
	Reader *bufio.Reader
}

type TeleinfoFrame struct {
	Index               uint `mapstructure:"BASE"`
	IntensityInstant    uint `mapstructure:"IINST"`
	IntensityMax        uint `mapstructure:"IMAX"`
	IntensitySubscribed uint `mapstructure:"ISOUSC"`
	PowerApparent       uint `mapstructure:"PAPP"`
	CollectionTime      time.Duration
}

func NewTeleinfo() *Teleinfo {
	config := &serial.Config{
		Name:        "/dev/ttyAMA0",
		Baud:        1200,
		Parity:      serial.ParityEven,
		ReadTimeout: time.Second * 1,
		Size:        7,
	}
	stream, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("Unable to open serial port: %v", err)
	}

	return &Teleinfo{Reader: bufio.NewReader(stream)}
}

func (t *Teleinfo) GetData() (frame *TeleinfoFrame, err error) {
	start := time.Now()

	slice, err := readFrame(t.Reader)
	if err != nil {
		log.Printf("Failed to read data: %v\n", err)
		return nil, err
	}

	frame, err = parseFrame(slice)
	if err != nil {
		log.Printf("Failed to parse data: %v\n", err)
		return nil, err
	}

	frame.CollectionTime = time.Since(start)

	return frame, nil
}

func readFrame(reader *bufio.Reader) (slice []byte, err error) {
	// "\x02\nADCO 000000000000 @\r\nOPTARIF BASE 0\r\nISOUSC 30 9\r\nBASE 007619848 6\r\nPTEC TH.. $\r\nIINST 002 Y\r\nIMAX 090 H\r\nPAPP 00420 '\r\nHHPHC A ,\r\nMOTDETAT 000000 B\r\x03"
	reader.ReadSlice('\x02')              // Read until frame start, discard incomplete frame
	slice, err = reader.ReadSlice('\x03') // Read until frame end
	if err != nil {
		return nil, err
	}

	return slice, nil
}

func parseFrame(slice []byte) (frame *TeleinfoFrame, err error) {
	str := strings.Trim(string(slice), "\r\n\x02\x03") // Remove leading/trailing chars
	tuples := strings.Split(str, "\r\n")

	frameMap := make(map[string]interface{})
	for _, tuple := range tuples {
		fields := strings.Fields(tuple)
		frameMap[fields[0]] = fields[1]
	}

	config := &ms.DecoderConfig{
		DecodeHook: paddedIntStringToUintHookFunc(),
		Result:     &frame,
	}

	decoder, err := ms.NewDecoder(config)
	if err != nil {
		return nil, err
	}

	err = decoder.Decode(frameMap)
	if err != nil {
		return nil, err
	}

	// fmt.Printf("input: %q\n", slice)
	// fmt.Printf("trimmed: %q\n", str)
	// fmt.Printf("split: %+q\n", tuples)
	// fmt.Printf("map: %v\n", frameMap)
	// fmt.Printf("%v\n", frame)
	// fmt.Printf("\n")

	return frame, err
}

func paddedIntStringToUintHookFunc() ms.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String || t.Kind() != reflect.Uint {
			return data, nil
		}
		return strconv.ParseInt(data.(string), 10, t.Bits())
	}
}
