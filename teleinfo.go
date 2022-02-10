package main

import (
	"bufio"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	ms "github.com/mitchellh/mapstructure"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tarm/serial"
)

var (
	index = prometheus.NewDesc(
		"teleinfo_index_kwh",
		"Current value of index in kilowatt.hour",
		nil, nil,
	)
	intensityInstant = prometheus.NewDesc(
		"teleinfo_intensity_instant_amp",
		"Current intensity demand in ampere",
		nil, nil,
	)
	intensityMax = prometheus.NewDesc(
		"teleinfo_intensity_max_amp",
		"Max intensity in ampere",
		nil, nil,
	)
	intensitySubscribed = prometheus.NewDesc(
		"teleinfo_intensity_subscribed_amp",
		"Subscribed intensity in ampere",
		nil, nil,
	)
	powerApparent = prometheus.NewDesc(
		"teleinfo_power_apparent_va",
		"Current apparent power in volt.ampere",
		nil, nil,
	)
	collectionTime = prometheus.NewDesc(
		"teleinfo_collection_time_seconds",
		"Teleinfo data collection duration in seconds",
		nil, nil,
	)
)

type TeleinfoCollector struct {
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

func NewTeleinfoCollector(serialDev string, reg prometheus.Registerer) *TeleinfoCollector {
	config := &serial.Config{
		Name:        serialDev,
		Baud:        1200,
		Parity:      serial.ParityEven,
		ReadTimeout: time.Second * 1,
		Size:        7,
	}
	stream, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("Unable to open serial port: %w", err)
	}

	t := &TeleinfoCollector{Reader: bufio.NewReader(stream)}
	reg.MustRegister(t)
	return t
}

func (t *TeleinfoCollector) GetData() (frame *TeleinfoFrame, err error) {
	start := time.Now()

	slice, err := readFrame(t.Reader)
	if err != nil {
		return nil, fmt.Errorf("read data: %w\n", err)
	}

	frame, err = parseFrame(slice)
	if err != nil {
		return nil, fmt.Errorf("parse data: %w\n", err)
	}

	frame.CollectionTime = time.Since(start)

	return frame, nil
}

func (t *TeleinfoCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(t, ch)
}

func (t *TeleinfoCollector) Collect(ch chan<- prometheus.Metric) {
	frame, err := t.GetData()
	if err != nil {
		log.Printf("Error collecting metrics: %w", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(index, prometheus.GaugeValue, float64(frame.Index))
	ch <- prometheus.MustNewConstMetric(intensityInstant, prometheus.GaugeValue, float64(frame.IntensityInstant))
	ch <- prometheus.MustNewConstMetric(intensityMax, prometheus.GaugeValue, float64(frame.IntensityMax))
	ch <- prometheus.MustNewConstMetric(intensitySubscribed, prometheus.GaugeValue, float64(frame.IntensitySubscribed))
	ch <- prometheus.MustNewConstMetric(powerApparent, prometheus.GaugeValue, float64(frame.PowerApparent))
	ch <- prometheus.MustNewConstMetric(collectionTime, prometheus.GaugeValue, float64(frame.CollectionTime.Seconds()))
}

func readFrame(reader *bufio.Reader) (slice []byte, err error) {
	reader.ReadSlice('\x03')              // Read until frame start, discard incomplete frame
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
		fields, err := splitTuple(tuple)
		if err != nil {
			return nil, err
		}
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

	return frame, nil
}

func splitTuple(tuple string) (fields []string, err error) {
	fields = strings.Split(tuple, " ")
	if nb := len(fields); nb != 3 {
		return nil, fmt.Errorf("expected 3 elements, got %d", len(fields))
	}

	checksum := 0
	for _, v := range tuple[:len(tuple)-2] {
		checksum += int(v)
	}
	checksum = (checksum & 63) + 32

	if int(rune(fields[2][0])) != checksum {
		return nil, fmt.Errorf("invalid checksum")
	}
	return fields[:2], err
}

func paddedIntStringToUintHookFunc() ms.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String || t.Kind() != reflect.Uint {
			return data, nil
		}
		return strconv.ParseInt(data.(string), 10, t.Bits())
	}
}
