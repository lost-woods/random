package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lost-woods/random/src/rng"
)

func (h *Handlers) RandomBytes(c *gin.Context) {
	const maxSize = 256

	sizeVar := c.DefaultQuery("size", "1")
	size, err := strconv.Atoi(sizeVar)
	if err != nil || size < 1 || size > maxSize {
		responder{c}.err(http.StatusBadRequest,
			fmt.Sprintf("Size must be an integer between 1 and %d.", maxSize))
		return
	}

	h.handleRNG(c, func() (string, gin.H, int, string) {
		buf := make([]byte, size)
		if _, err := io.ReadFull(h.r, buf); err != nil {
			if h.health != nil {
				h.health.Set(false, "error fetching random bytes: "+err.Error())
			}
			h.log.Error(err)
			return "", nil, http.StatusInternalServerError, "Error fetching random bytes."
		}

		hex := fmt.Sprintf("%x", buf)
		return hex, gin.H{"bytes": hex, "size": size}, 0, ""
	})
}

func (h *Handlers) RandomNumber(c *gin.Context) {
	min, err := strconv.Atoi(c.DefaultQuery("min", "1"))
	if err != nil {
		responder{c}.err(http.StatusBadRequest, "Invalid min value.")
		return
	}

	max, err := strconv.Atoi(c.DefaultQuery("max", "100"))
	if err != nil {
		responder{c}.err(http.StatusBadRequest, "Invalid max value.")
		return
	}

	h.handleRNG(c, func() (string, gin.H, int, string) {
		n, err := rng.UniformInt32(h.r, h.health, min, max)
		if err != nil {
			return "", nil, http.StatusBadRequest, err.Error()
		}

		return fmt.Sprintf("%d", n),
			gin.H{"number": n, "min": min, "max": max},
			0, ""
	})
}

func (h *Handlers) RandomCards(c *gin.Context) {
	numDecks, err := strconv.Atoi(c.DefaultQuery("decks", "1"))
	if err != nil || numDecks < 1 || numDecks > 100 {
		responder{c}.err(http.StatusBadRequest, "Invalid deck count.")
		return
	}

	jokers, err := strconv.ParseBool(c.DefaultQuery("jokers", "false"))
	if err != nil {
		responder{c}.err(http.StatusBadRequest, "Invalid jokers flag.")
		return
	}

	numCards, err := strconv.Atoi(c.DefaultQuery("cards", "1"))
	if err != nil || numCards < 1 {
		responder{c}.err(http.StatusBadRequest, "Invalid card count.")
		return
	}

	h.handleRNG(c, func() (string, gin.H, int, string) {
		deck := rng.AddDeck(numDecks, jokers)
		if numCards > len(deck) {
			return "", nil, http.StatusBadRequest,
				"There are more cards to pick than cards in the deck."
		}

		picked := make([]rng.Card, 0, numCards)
		for i := 0; i < numCards; i++ {
			index := int32(0)
			if len(deck) > 1 {
				var err error
				index, err = rng.UniformInt32(h.r, h.health, 0, len(deck)-1)
				if err != nil {
					return "", nil, http.StatusInternalServerError,
						"Error fetching a random card."
				}
			}
			picked = append(picked, deck[int(index)])
			deck = rng.RemoveCard(deck, int(index))
		}

		var out bytes.Buffer
		for i, c := range picked {
			out.WriteString(c.Value + " of " + c.Suit)
			if i < len(picked)-1 {
				out.WriteByte('\n')
			}
		}

		return out.String(), gin.H{
			"decks":  numDecks,
			"jokers": jokers,
			"cards":  numCards,
			"drawn":  picked,
		}, 0, ""
	})
}

func (h *Handlers) RandomStrings(c *gin.Context) {
	const maxSize = 256

	size, err := strconv.Atoi(c.DefaultQuery("size", "10"))
	if err != nil || size < 1 || size > maxSize {
		responder{c}.err(http.StatusBadRequest, "Invalid size.")
		return
	}

	lowers, err := strconv.ParseBool(c.DefaultQuery("lowercase", "true"))
	if err != nil {
		responder{c}.err(http.StatusBadRequest, "Invalid lowercase flag.")
		return
	}

	uppers, err := strconv.ParseBool(c.DefaultQuery("uppercase", "true"))
	if err != nil {
		responder{c}.err(http.StatusBadRequest, "Invalid uppercase flag.")
		return
	}

	numbers, err := strconv.ParseBool(c.DefaultQuery("numbers", "true"))
	if err != nil {
		responder{c}.err(http.StatusBadRequest, "Invalid numbers flag.")
		return
	}

	symbols, err := strconv.ParseBool(c.DefaultQuery("symbols", "true"))
	if err != nil {
		responder{c}.err(http.StatusBadRequest, "Invalid symbols flag.")
		return
	}

	if !lowers && !uppers && !numbers && !symbols {
		responder{c}.err(http.StatusBadRequest, "At least one flag must be set.")
		return
	}

	h.handleRNG(c, func() (string, gin.H, int, string) {
		charset := rng.BuildCharset(lowers, uppers, numbers, symbols)
		var out bytes.Buffer
		out.Grow(size)

		for i := 0; i < size; i++ {
			index, err := rng.UniformInt32(h.r, h.health, 0, len(charset)-1)
			if err != nil {
				return "", nil, http.StatusInternalServerError,
					"Error fetching a random character."
			}
			out.WriteByte(charset[int(index)])
		}

		s := out.String()
		return s, gin.H{
			"string":    s,
			"size":      size,
			"lowercase": lowers,
			"uppercase": uppers,
			"numbers":   numbers,
			"symbols":   symbols,
		}, 0, ""
	})
}

func (h *Handlers) RandomPercent(c *gin.Context) {
	percentStr := c.DefaultQuery("percent", "25")

	h.handleRNG(c, func() (string, gin.H, int, string) {
		num, den, err := rng.ParsePercentExact(percentStr)
		if err != nil {
			return "", nil, http.StatusBadRequest, err.Error()
		}

		roll, err := rng.UniformInt32(h.r, h.health, 1, den)
		if err != nil {
			return "", nil, http.StatusInternalServerError,
				"Error fetching a random number."
		}

		pass := int(roll) <= num
		result := "Fail"
		if pass {
			result = "Pass"
		}

		text := fmt.Sprintf("Rolled %d from %d/%d\n%s", roll, num, den, result)
		return text, gin.H{
			"percent": percentStr,
			"success": num,
			"out_of":  den,
			"roll":    int(roll),
			"pass":    pass,
		}, 0, ""
	})
}

func (h *Handlers) Health(c *gin.Context) {
	if h.health == nil {
		responder{c}.err(http.StatusServiceUnavailable, "UNHEALTHY: missing health monitor")
		return
	}

	ok, msg, t := h.health.Snapshot()
	if ok {
		responder{c}.ok(
			fmt.Sprintf("OK (last checked %s)", t.Format(time.RFC3339)),
			gin.H{"ok": true, "last_checked": t.Format(time.RFC3339)},
			"health-check",
		)
		return
	}

	responder{c}.err(http.StatusServiceUnavailable,
		fmt.Sprintf("UNHEALTHY: %s (last checked %s)", msg, t.Format(time.RFC3339)))
}
