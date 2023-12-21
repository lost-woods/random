package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/tarm/serial"
	"go.uber.org/zap"
)

var (
	zapLogger, _ = zap.NewProduction()
	log          = zapLogger.Sugar()
)

func main() {
	// Debug
	config := &serial.Config{
		Name:        "/dev/random",
		Baud:        9600,
		ReadTimeout: 1,
		Size:        8,
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
