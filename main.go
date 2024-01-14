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

type Card struct {
	Value string `json:"value"`
	Suit  string `json:"suit"`
}

func addDeck(numDecks int, jokers bool) []Card {
	deck := []Card{}
	decksAdded := 0

	suits := []string{"Hearts", "Diamonds", "Clubs", "Spades"}

	for decksAdded < numDecks {
		i := 0

		for i < 4 {
			suit := suits[i]

			deck = append(deck, Card{Value: "Ace", Suit: suit})
			deck = append(deck, Card{Value: "Two", Suit: suit})
			deck = append(deck, Card{Value: "Three", Suit: suit})
			deck = append(deck, Card{Value: "Four", Suit: suit})
			deck = append(deck, Card{Value: "Five", Suit: suit})
			deck = append(deck, Card{Value: "Six", Suit: suit})
			deck = append(deck, Card{Value: "Seven", Suit: suit})
			deck = append(deck, Card{Value: "Eight", Suit: suit})
			deck = append(deck, Card{Value: "Nine", Suit: suit})
			deck = append(deck, Card{Value: "Ten", Suit: suit})
			deck = append(deck, Card{Value: "Jack", Suit: suit})
			deck = append(deck, Card{Value: "Queen", Suit: suit})
			deck = append(deck, Card{Value: "King", Suit: suit})

			i += 1
		}

		if jokers {
			deck = append(deck, Card{Value: "Joker", Suit: "Red"})
			deck = append(deck, Card{Value: "Joker", Suit: "Black"})
		}

		decksAdded += 1
	}

	return deck
}

func removeCard(deck []Card, index int) []Card {
	return append(deck[:index], deck[index+1:]...)
}

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

func randomCards(c *gin.Context) {
	// Input
	numDecksVar := c.DefaultQuery("decks", "1")
	numDecks, err := strconv.Atoi(numDecksVar)
	if err != nil {
		c.String(http.StatusBadRequest, "The number of decks could not be read.")
		return
	}

	jokersVar := c.DefaultQuery("jokers", "false")
	jokers, err := strconv.ParseBool(jokersVar)
	if err != nil {
		c.String(http.StatusBadRequest, "The jokers flag could not be read.")
		return
	}

	numCardsVar := c.DefaultQuery("cards", "1")
	numCards, err := strconv.Atoi(numCardsVar)
	if err != nil {
		c.String(http.StatusBadRequest, "The number of cards to pick could not be read.")
		return
	}

	if numDecks < 1 {
		c.String(http.StatusBadRequest, "There must be at least 1 deck.")
		return
	}

	if numDecks > 100 {
		c.String(http.StatusBadRequest, "There can be a maximum of 100 decks.")
		return
	}

	if numCards < 1 {
		c.String(http.StatusBadRequest, "There must be at least 1 card to pick.")
		return
	}

	// Generate new deck
	deck := addDeck(numDecks, jokers)

	// Pick and display cards
	if numCards > len(deck) {
		c.String(http.StatusBadRequest, "There are more cards to pick than cards in the deck.")
		return
	}

	out := ""
	for numCards > 0 {
		index := int32(0)
		if len(deck) > 1 {
			index, err = generateRandomNumber(0, len(deck)-1)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error fetching a random card.")
				return
			}
		}

		out = out + deck[int(index)].Value + " of " + deck[int(index)].Suit + "\n"
		deck = removeCard(deck, int(index))
		numCards = numCards - 1
	}

	// Remove trailing \n
	if len(out) > 0 {
		out = out[:len(out)-1]
	}

	c.String(http.StatusOK, out)
}

func randomStrings(c *gin.Context) {
	maxSize := 256

	// Input
	sizeVar := c.DefaultQuery("size", "10")
	size, err := strconv.Atoi(sizeVar)
	if err != nil {
		c.String(http.StatusBadRequest, "The size value could not be read.")
		return
	}

	lowersVar := c.DefaultQuery("lowercase", "true")
	lowers, err := strconv.ParseBool(lowersVar)
	if err != nil {
		c.String(http.StatusBadRequest, "The lowercase flag could not be read.")
		return
	}

	uppersVar := c.DefaultQuery("uppercase", "true")
	uppers, err := strconv.ParseBool(uppersVar)
	if err != nil {
		c.String(http.StatusBadRequest, "The uppercase flag could not be read.")
		return
	}

	numbersVar := c.DefaultQuery("numbers", "true")
	numbers, err := strconv.ParseBool(numbersVar)
	if err != nil {
		c.String(http.StatusBadRequest, "The numbers flag could not be read.")
		return
	}

	symbolsVar := c.DefaultQuery("symbols", "true")
	symbols, err := strconv.ParseBool(symbolsVar)
	if err != nil {
		c.String(http.StatusBadRequest, "The symbols flag could not be read.")
		return
	}

	if size < 1 {
		c.String(http.StatusBadRequest, "The string must have at least size 1.")
		return
	}

	if size > maxSize {
		c.String(http.StatusBadRequest, fmt.Sprintf("The string must have at most size %d.", maxSize))
		return
	}

	if !uppers && !lowers && !numbers && !symbols {
		c.String(http.StatusBadRequest, "At least one flag must be set.")
		return
	}

	chars := ""
	lowerChars := "abcdefghijklmnopqrstuvwxyz"
	upperChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberChars := "0123456789"
	symbolsChars := "!#$%&()*+,-./:;<=>?@[]^_{|}~"

	if lowers {
		chars = chars + lowerChars
	}

	if uppers {
		chars = chars + upperChars
	}

	if numbers {
		chars = chars + numberChars
	}

	if symbols {
		chars = chars + symbolsChars
	}

	out := ""
	for size > 0 {
		index, err := generateRandomNumber(0, len(chars)-1)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error fetching a random character.")
			return
		}

		out = out + string(chars[int(index)])
		size = size - 1
	}

	c.String(http.StatusOK, out)
}

func randomPercent(c *gin.Context) {
	// Input
	percentVar := c.DefaultQuery("percent", "25")
	percent, err := strconv.ParseFloat(percentVar, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "The percent chance value could not be read.")
		return
	}

	if percent <= float64(0) {
		c.String(http.StatusBadRequest, "The percent chance cannot be 0 or lower.")
		return
	}

	if percent >= float64(100) {
		c.String(http.StatusBadRequest, "The percent chance cannot be 100 or larger.")
		return
	}

	success := 1
	retries := 5
	chance := float64(100) / percent
	whole, remainder := math.Modf(chance)
	for retries > 0 && remainder != float64(0) {
		success = success * 10
		chance = chance * 10

		whole, remainder = math.Modf(chance)
		retries = retries - 1
	}

	if int(whole) <= 1 || int(whole) > 1000000000 {
		c.String(http.StatusBadRequest, "The percent chance is too small.")
		return
	}

	roll, err := generateRandomNumber(1, int(whole))
	if err != nil {
		c.String(http.StatusInternalServerError, "Error fetching a random number.")
		return
	}

	out := "Fail"
	if int(roll) <= success {
		out = "Pass"
	}

	c.String(http.StatusOK, fmt.Sprintf("Rolled %d from %d/%d\n%s", int(roll), success, int(whole), out))
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
	router.GET("/cards", randomCards)
	router.GET("/strings", randomStrings)
	router.GET("/percent", randomPercent)

	router.Run(":" + port)
}
