package main

import (
	"bufio"
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
		log.Fatalf("Unable to open serial port: %v", err)
	}

	t := &TeleinfoCollector{Reader: bufio.NewReader(stream)}
	reg.MustRegister(t)
	return t
}

func (t *TeleinfoCollector) GetData() (frame *TeleinfoFrame, err error) {
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

func (t *TeleinfoCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(t, ch)
}

func (t *TeleinfoCollector) Collect(ch chan<- prometheus.Metric) {
	frame, _ := t.GetData()

	ch <- prometheus.MustNewConstMetric(index, prometheus.GaugeValue, float64(frame.Index))
	ch <- prometheus.MustNewConstMetric(intensityInstant, prometheus.GaugeValue, float64(frame.IntensityInstant))
	ch <- prometheus.MustNewConstMetric(intensityMax, prometheus.GaugeValue, float64(frame.IntensityMax))
	ch <- prometheus.MustNewConstMetric(intensitySubscribed, prometheus.GaugeValue, float64(frame.IntensitySubscribed))
	ch <- prometheus.MustNewConstMetric(powerApparent, prometheus.GaugeValue, float64(frame.PowerApparent))
	ch <- prometheus.MustNewConstMetric(collectionTime, prometheus.GaugeValue, float64(frame.CollectionTime.Seconds()))
}

func readFrame(reader *bufio.Reader) (slice []byte, err error) {
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
