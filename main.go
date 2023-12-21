package main

import (
	"os"
	"strconv"
	"time"

	"github.com/tarm/serial"
	"go.uber.org/zap"
)

var (
	zapLogger, _ = zap.NewProduction()
	log          = zapLogger.Sugar()
)

func main() {
	// Variables
	name := os.Getenv("SERIAL_DEVICE_NAME")

	baud, err := strconv.Atoi(os.Getenv("SERIAL_BAUD_RATE"))
	if err != nil {
		log.Fatal(err)
	}

	size, err := strconv.Atoi(os.Getenv("SERIAL_DATA_SIZE"))
	if err != nil {
		log.Fatal(err)
	}

	timeout, err := strconv.Atoi(os.Getenv("SERIAL_READ_TIMEOUT"))
	if err != nil {
		log.Fatal(err)
	}

	// Set up serial port for reading
	config := &serial.Config{
		Name:        name,
		Baud:        baud,
		Size:        byte(size),
		ReadTimeout: time.Duration(timeout),
	}

	// Read bytes from the serial port
	stream, err := serial.OpenPort(config)
	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, size)
	n, err := stream.Read(buf)
	if err != nil {
		log.Fatal(err)
	}

	// Print result
	log.Infoln("%x", buf[:n])

	// Exit
	log.Info("End.")
	os.Exit(0)
}
