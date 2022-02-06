package main

import (
	"log"
)

func main() {

	t := NewTeleinfo()
	for {
		frame, err := t.GetData()
		if err != nil {
			log.Printf("Unable to open serial port: %v", err)
		}
		log.Printf("%v\n", frame)
	}
}
