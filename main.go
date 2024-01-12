package main

import (
	"encoding/binary"
	"errors"
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

func generateRandomNumber(min int, max int) (int32, error) {
	// Sanity check
	if min < -1000000000 {
		return 0, errors.New("the minimum value should not be lower than -1,000,000,000")
	}

	if min >= 1000000000 {
		return 0, errors.New("the minimum value should not be higher than 999,999,999")
	}

	if max <= -1000000000 {
		return 0, errors.New("the maximum value should not be lower than -999,999,999")
	}

	if max > 1000000000 {
		return 0, errors.New("the maximum value should not be higher than 1,000,000,000")
	}

	if min >= max {
		return 0, errors.New("the minimum value should be smaller than the maximum value")
	}

	// Processing
	size := 4 // Hard-coded for uint32 size
	int32MaxNumber := binary.BigEndian.Uint32([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	divisor := uint32(max - min + 1) // Max divisor for -1B to +1B range both inclusive, including 0 is 2,000,000,001

	// Handle mod bias
	// https://research.kudelskisecurity.com/2020/07/28/the-definitive-guide-to-modulo-bias-and-how-to-avoid-it/
	whole, remainder := math.Modf((float64(int32MaxNumber) + float64(1)) / float64(divisor))
	cutoffNumber := uint32(whole) * divisor // May overflow to 0

	// Generate the number
	rng := uint32(0)
	numberGenerated := false
	for !numberGenerated {
		buf := make([]byte, size)
		_, err := trueRNG.Read(buf)
		if err != nil {
			return 0, errors.New("error fetching random bytes")
		}

		rng = binary.BigEndian.Uint32(buf)                 // 8 bits * size 4 = 32
		if remainder == float64(0) || rng < cutoffNumber { // If this is an exact division, skip the cutoff, which will have overflowed
			numberGenerated = true
		}
	}

	// Send result
	finalNumber := int32(float64(rng%divisor) + float64(min)) // Can never exceed -1B or 1B, which is within signed int32 (2,147,483,647)
	return finalNumber, nil
}

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
	// Input
	minVar := c.DefaultQuery("min", "1")
	min, err := strconv.Atoi(minVar)
	if err != nil {
		c.String(http.StatusBadRequest, "The minimum value could not be read.")
		return
	}

	maxVar := c.DefaultQuery("max", "100")
	max, err := strconv.Atoi(maxVar)
	if err != nil {
		c.String(http.StatusBadRequest, "The maximum value could not be read.")
		return
	}

	// Attempt to get a random number
	rng, err := generateRandomNumber(min, max)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.String(http.StatusOK, fmt.Sprintf("%d", rng))
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
	router.GET("/bytes", randomBytes)

	router.Run(":" + port)
}
