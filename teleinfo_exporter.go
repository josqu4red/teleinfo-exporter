package main

import (
	"bufio"
	"fmt"
	"log"
	"time"

	"github.com/tarm/serial"
)

func main() {
	config := &serial.Config{
		Name:        "/dev/ttyAMA0",
		Baud:        1200,
		Parity:      serial.ParityEven,
		ReadTimeout: time.Second * 1,
		Size:        7,
	}

	stream, err := serial.OpenPort(config)
	if err != nil {
		log.Fatal(err)
	}
	reader := bufio.NewReader(stream)

	for {
		frame, _ := GetTeleinfoData(reader)
		fmt.Printf("%v\n", frame)
	}
}
