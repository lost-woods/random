package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"github.com/tarm/serial"
	"go.uber.org/zap"
)

var (
	zapLogger, _ = zap.NewProduction()
	log          = zapLogger.Sugar()
)

func main() {
	// Debug
	baud, _ := strconv.Atoi(os.Getenv("SERIAL_BAUD_RATE"))
	size, _ := strconv.Atoi(os.Getenv("SERIAL_DATA_SIZE"))
	config := &serial.Config{
		Name:        os.Getenv("SERIAL_DEVICE_NAME"),
		Baud:        baud,
		Size:        byte(size),
		ReadTimeout: 1,
	}

	stream, err := serial.OpenPort(config)
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		fmt.Println(scanner.Text()) // Println will add back the final '\n'
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	log.Info("End.")
	os.Exit(0)
}
