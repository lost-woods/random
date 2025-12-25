package api

import (
    "bytes"
    "fmt"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"

    "github.com/lost-woods/random/src/cards"
    "github.com/lost-woods/random/src/rng"
)

type Handlers struct {
    r      rng.Reader
    health *rng.Health
    log    *zap.SugaredLogger
}

func NewHandlers(r rng.Reader, h *rng.Health, log *zap.SugaredLogger) *Handlers {
    return &Handlers{r: r, health: h, log: log}
}

// Clients select JSON via the Accept header.
func wantsJSON(c *gin.Context) bool {
    accept := strings.ToLower(c.GetHeader("Accept"))
    return strings.Contains(accept, "application/json")
}

func respondErr(c *gin.Context, status int, msg string) {
    if wantsJSON(c) {
        c.JSON(status, gin.H{"error": msg})
        return
    }
    c.String(status, msg)
}

func respondOKWithID(c *gin.Context, text string, obj gin.H, requestID string) {
    if wantsJSON(c) {
        obj["request_id"] = requestID
        c.JSON(http.StatusOK, obj)
        return
    }
    // Plain text: add request id on a new line.
    c.String(http.StatusOK, text+"\nrequest_id: "+requestID)
}

// newUUIDv4FromRNG generates an RFC4122 UUID v4 using the same RNG stream.
// This is generated ONLY after a successful outcome is computed, so it does not bias outcomes.
func newUUIDv4FromRNG(r rng.Reader, h *rng.Health) (string, error) {
    b := make([]byte, 16)
    if err := rng.ReadFull(r, b); err != nil {
        if h != nil {
            h.Set(false, "error fetching random bytes for uuid: "+err.Error())
        }
        return "", err
    }
    // Set version (4) and variant (10xx)
    b[6] = (b[6] & 0x0f) | 0x40
    b[8] = (b[8] & 0x3f) | 0x80
    return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func (h *Handlers) rngOK(c *gin.Context) bool {
    ok, msg, _ := h.health.Snapshot()
    if !ok {
        respondErr(c, http.StatusServiceUnavailable, "RNG unhealthy: "+msg)
        return false
    }
    return true
}

func (h *Handlers) RandomBytes(c *gin.Context) {
    const maxSize = 256

    sizeVar := c.DefaultQuery("size", "1")
    size, err := strconv.Atoi(sizeVar)
    if err != nil {
        respondErr(c, http.StatusBadRequest, fmt.Sprintf("Size parameter should be an integer between 1 and %d bytes.", maxSize))
        return
    }
    if size < 1 {
        respondErr(c, http.StatusBadRequest, "Size should not be smaller than 1 byte.")
        return
    }
    if size > maxSize {
        respondErr(c, http.StatusBadRequest, fmt.Sprintf("Size should not exceed %d bytes per request.", maxSize))
        return
    }

    if !h.rngOK(c) {
        return
    }

    buf := make([]byte, size)
    if err := rng.ReadFull(h.r, buf); err != nil {
        h.health.Set(false, "error fetching random bytes: "+err.Error())
        h.log.Error(err)
        respondErr(c, http.StatusInternalServerError, "Error fetching random bytes.")
        return
    }

    hex := fmt.Sprintf("%x", buf)
    requestID, uuidErr := newUUIDv4FromRNG(h.r, h.health)
    if uuidErr != nil {
        respondErr(c, http.StatusInternalServerError, "Error generating request id.")
        return
    }
    respondOKWithID(c, hex, gin.H{"bytes_hex": hex, "size": size}, requestID)
}

func (h *Handlers) RandomNumber(c *gin.Context) {
    minVar := c.DefaultQuery("min", "1")
    min, err := strconv.Atoi(minVar)
    if err != nil {
        respondErr(c, http.StatusBadRequest, "The minimum value could not be read.")
        return
    }

    maxVar := c.DefaultQuery("max", "100")
    max, err := strconv.Atoi(maxVar)
    if err != nil {
        respondErr(c, http.StatusBadRequest, "The maximum value could not be read.")
        return
    }

    if !h.rngOK(c) {
        return
    }

    n, err := rng.UniformInt32(h.r, h.health, min, max)
    if err != nil {
        respondErr(c, http.StatusBadRequest, err.Error())
        return
    }

    requestID, uuidErr := newUUIDv4FromRNG(h.r, h.health)
    if uuidErr != nil {
        respondErr(c, http.StatusInternalServerError, "Error generating request id.")
        return
    }

    respondOKWithID(c, fmt.Sprintf("%d", n), gin.H{"value": n, "min": min, "max": max}, requestID)
}

func (h *Handlers) RandomCards(c *gin.Context) {
    numDecksVar := c.DefaultQuery("decks", "1")
    numDecks, err := strconv.Atoi(numDecksVar)
    if err != nil {
        respondErr(c, http.StatusBadRequest, "The number of decks could not be read.")
        return
    }

    jokersVar := c.DefaultQuery("jokers", "false")
    jokers, err := strconv.ParseBool(jokersVar)
    if err != nil {
        respondErr(c, http.StatusBadRequest, "The jokers flag could not be read.")
        return
    }

    numCardsVar := c.DefaultQuery("cards", "1")
    numCards, err := strconv.Atoi(numCardsVar)
    if err != nil {
        respondErr(c, http.StatusBadRequest, "The number of cards to pick could not be read.")
        return
    }

    if numDecks < 1 {
        respondErr(c, http.StatusBadRequest, "There must be at least 1 deck.")
        return
    }
    if numDecks > 100 {
        respondErr(c, http.StatusBadRequest, "There can be a maximum of 100 decks.")
        return
    }
    if numCards < 1 {
        respondErr(c, http.StatusBadRequest, "There must be at least 1 card to pick.")
        return
    }

    if !h.rngOK(c) {
        return
    }

    deck := cards.AddDeck(numDecks, jokers)
    if numCards > len(deck) {
        respondErr(c, http.StatusBadRequest, "There are more cards to pick than cards in the deck.")
        return
    }

    if wantsJSON(c) {
        picked := make([]cards.Card, 0, numCards)
        for i := 0; i < numCards; i++ {
            index := int32(0)
            if len(deck) > 1 {
                index, err = rng.UniformInt32(h.r, h.health, 0, len(deck)-1)
                if err != nil {
                    respondErr(c, http.StatusInternalServerError, "Error fetching a random card.")
                    return
                }
            }
            picked = append(picked, deck[int(index)])
            deck = cards.RemoveCard(deck, int(index))
        }

        requestID, uuidErr := newUUIDv4FromRNG(h.r, h.health)
        if uuidErr != nil {
            respondErr(c, http.StatusInternalServerError, "Error generating request id.")
            return
        }

        c.JSON(http.StatusOK, gin.H{
            "request_id": requestID,
            "decks":      numDecks,
            "jokers":     jokers,
            "count":      numCards,
            "cards":      picked,
        })
        return
    }

    var out bytes.Buffer
    for i := 0; i < numCards; i++ {
        index := int32(0)
        if len(deck) > 1 {
            index, err = rng.UniformInt32(h.r, h.health, 0, len(deck)-1)
            if err != nil {
                respondErr(c, http.StatusInternalServerError, "Error fetching a random card.")
                return
            }
        }

        card := deck[int(index)]
        out.WriteString(card.Value)
        out.WriteString(" of ")
        out.WriteString(card.Suit)
        if i != numCards-1 {
            out.WriteByte('\n')
        }

        deck = cards.RemoveCard(deck, int(index))
    }

    requestID, uuidErr := newUUIDv4FromRNG(h.r, h.health)
    if uuidErr != nil {
        respondErr(c, http.StatusInternalServerError, "Error generating request id.")
        return
    }

    c.String(http.StatusOK, out.String()+"\nrequest_id: "+requestID)
}

func (h *Handlers) RandomStrings(c *gin.Context) {
    const maxSize = 256

    sizeVar := c.DefaultQuery("size", "10")
    size, err := strconv.Atoi(sizeVar)
    if err != nil {
        respondErr(c, http.StatusBadRequest, "The size value could not be read.")
        return
    }

    lowersVar := c.DefaultQuery("lowercase", "true")
    lowers, err := strconv.ParseBool(lowersVar)
    if err != nil {
        respondErr(c, http.StatusBadRequest, "The lowercase flag could not be read.")
        return
    }

    uppersVar := c.DefaultQuery("uppercase", "true")
    uppers, err := strconv.ParseBool(uppersVar)
    if err != nil {
        respondErr(c, http.StatusBadRequest, "The uppercase flag could not be read.")
        return
    }

    numbersVar := c.DefaultQuery("numbers", "true")
    numbers, err := strconv.ParseBool(numbersVar)
    if err != nil {
        respondErr(c, http.StatusBadRequest, "The numbers flag could not be read.")
        return
    }

    symbolsVar := c.DefaultQuery("symbols", "true")
    symbols, err := strconv.ParseBool(symbolsVar)
    if err != nil {
        respondErr(c, http.StatusBadRequest, "The symbols flag could not be read.")
        return
    }

    if size < 1 {
        respondErr(c, http.StatusBadRequest, "The string must have at least size 1.")
        return
    }
    if size > maxSize {
        respondErr(c, http.StatusBadRequest, fmt.Sprintf("The string must have at most size %d.", maxSize))
        return
    }
    if !uppers && !lowers && !numbers && !symbols {
        respondErr(c, http.StatusBadRequest, "At least one flag must be set.")
        return
    }

    if !h.rngOK(c) {
        return
    }

    charset := rng.BuildCharset(lowers, uppers, numbers, symbols)
    var out bytes.Buffer
    out.Grow(size)

    for i := 0; i < size; i++ {
        index, err := rng.UniformInt32(h.r, h.health, 0, len(charset)-1)
        if err != nil {
            respondErr(c, http.StatusInternalServerError, "Error fetching a random character.")
            return
        }
        out.WriteByte(charset[int(index)])
    }

    s := out.String()
    requestID, uuidErr := newUUIDv4FromRNG(h.r, h.health)
    if uuidErr != nil {
        respondErr(c, http.StatusInternalServerError, "Error generating request id.")
        return
    }

    respondOKWithID(c, s, gin.H{
        "value":     s,
        "size":      size,
        "lowercase": lowers,
        "uppercase": uppers,
        "numbers":   numbers,
        "symbols":   symbols,
    }, requestID)
}

func (h *Handlers) RandomPercent(c *gin.Context) {
    percentStr := c.DefaultQuery("percent", "25")

    if !h.rngOK(c) {
        return
    }

    num, den, err := rng.ParsePercentExact(percentStr)
    if err != nil {
        respondErr(c, http.StatusBadRequest, err.Error())
        return
    }

    roll, err := rng.UniformInt32(h.r, h.health, 1, den)
    if err != nil {
        respondErr(c, http.StatusInternalServerError, "Error fetching a random number.")
        return
    }

    pass := int(roll) <= num
    result := "Fail"
    if pass {
        result = "Pass"
    }

    requestID, uuidErr := newUUIDv4FromRNG(h.r, h.health)
    if uuidErr != nil {
        respondErr(c, http.StatusInternalServerError, "Error generating request id.")
        return
    }

    respondOKWithID(
        c,
        fmt.Sprintf("Rolled %d from %d/%d\n%s", int(roll), num, den, result),
        gin.H{
            "percent_input": percentStr,
            "success":       num,
            "out_of":        den,
            "roll":          int(roll),
            "result":        result,
            "pass":          pass,
        },
        requestID,
    )
}

func (h *Handlers) Health(c *gin.Context) {
    ok, msg, t := h.health.Snapshot()
    if ok {
        requestID, uuidErr := newUUIDv4FromRNG(h.r, h.health)
        if uuidErr != nil {
            respondErr(c, http.StatusInternalServerError, "Error generating request id.")
            return
        }
        respondOKWithID(c, fmt.Sprintf("OK (last checked %s)", t.Format(time.RFC3339)),
            gin.H{"ok": true, "last_checked": t.Format(time.RFC3339)}, requestID)
        return
    }
    respondErr(c, http.StatusServiceUnavailable, fmt.Sprintf("UNHEALTHY: %s (last checked %s)", msg, t.Format(time.RFC3339)))
}

// --- middleware helpers ---
func APIKeyFromEnv() string { return os.Getenv("API_KEY") }

func CheckHeader(headerName, expectedValue string) gin.HandlerFunc {
    return func(c *gin.Context) {
        if expectedValue == "" {
            c.AbortWithStatus(http.StatusForbidden)
            return
        }
        if c.GetHeader(headerName) != expectedValue {
            c.AbortWithStatus(http.StatusForbidden)
            return
        }
        c.Next()
    }
}
