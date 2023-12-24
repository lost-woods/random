package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tarm/serial"
	"go.uber.org/zap"
)

var (
	zapLogger, _ = zap.NewProduction()
	log          = zapLogger.Sugar()

	trueRNG = newSerial()
	port    = "777"
)

func randomNumber(c *gin.Context) {
	// Variables
	size, err := strconv.Atoi(os.Getenv("SERIAL_DATA_SIZE"))
	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, size)
	n, err := trueRNG.Read(buf)
	if err != nil {
		log.Fatal(err)
	}

	// Print result
	c.String(http.StatusOK, fmt.Sprintf("%x", buf[:n]))
}

func newSerial() *serial.Port {
	name := os.Getenv("SERIAL_DEVICE_NAME")

	baud, err := strconv.Atoi(os.Getenv("SERIAL_BAUD_RATE"))
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
		Size:        8, // Hard coded by the library
		ReadTimeout: time.Duration(timeout),
	}

	stream, err := serial.OpenPort(config)
	if err != nil {
		log.Fatal(err)
	}

	return stream
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.GET("/", randomNumber)
	router.Run(":" + port)
}
