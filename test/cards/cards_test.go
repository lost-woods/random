package cards_test

import (
	"testing"

	"github.com/lost-woods/random/src/rng"
)

func TestDeck_UniqueSingleDeck(t *testing.T) {
	deck := rng.AddDeck(1, false)
	seen := map[string]bool{}

	for _, card := range deck {
		key := card.Value + "_" + card.Suit
		if seen[key] {
			t.Fatalf("duplicate card: %s", key)
		}
		seen[key] = true
	}

	if len(seen) != 52 {
		t.Fatalf("expected 52 unique cards, got %d", len(seen))
	}
}

func TestDeck_MultipleDecksMultiplicity(t *testing.T) {
	decks := 3
	deck := rng.AddDeck(decks, false)
	counts := map[string]int{}

	for _, c := range deck {
		key := c.Value + "_" + c.Suit
		counts[key]++
	}

	for k, v := range counts {
		if v != decks {
			t.Fatalf("card %s appears %d times, want %d", k, v, decks)
		}
	}
}
