package main

import (
	"encoding/binary"
	"fmt"
	"math"
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

func randomBytes(c *gin.Context) {
	maxSize := 256

	sizeVar := c.DefaultQuery("size", "1")
	size, err := strconv.Atoi(sizeVar)
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Size parameter should be an integer between 1 and %d bytes.", maxSize))
		return
	}

	if size < 1 {
		c.String(http.StatusBadRequest, "Size should not be smaller than 1 byte.")
		return
	}

	if size > maxSize {
		c.String(http.StatusBadRequest, fmt.Sprintf("Size should not exceed %d bytes per request.", maxSize))
		return
	}

	buf := make([]byte, size)
	n, err := trueRNG.Read(buf)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error fetching random bytes.")
		log.Error(err)
		return
	}

	// Print result
	c.String(http.StatusOK, fmt.Sprintf("%x", buf[:n]))
}

func randomNumber(c *gin.Context) {
	size := 4
	divisor := uint32(10)
	int32MaxNumber := binary.BigEndian.Uint32([]byte{0xFF, 0xFF, 0xFF, 0xFF})

	// Handle mod bias
	remainder := (float64(int32MaxNumber) + float64(1)) / float64(divisor)
	log.Infof("Division result: %.2f", remainder)

	remainder = math.Floor(remainder)
	log.Infof("Remainder: %.2f", remainder)

	cutOffNumber := uint32(remainder) * divisor
	log.Infof("Cutoff: %d", cutOffNumber)

	rng := uint32(0)
	numberGenerated := false
	for !numberGenerated {
		buf := make([]byte, size)
		_, err := trueRNG.Read(buf)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error fetching random bytes.")
			log.Error(err)
			return
		}

		rng := binary.BigEndian.Uint32(buf) // 8 bits * size 4 = 32
		log.Infof("[DEBUG] rng: %d int32MaxNumber: %d", rng, int32MaxNumber)

		if rng < cutOffNumber {
			numberGenerated = true
		}
	}

	// Print result
	c.String(http.StatusOK, fmt.Sprintf("%d", (rng%divisor)+1))
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

	router.GET("/", randomBytes)
	router.GET("/number", randomNumber)

	router.Run(":" + port)
}
