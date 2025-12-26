package rng

type Card struct {
	Value string `json:"value"`
	Suit  string `json:"suit"`
}

func AddDeck(numDecks int, jokers bool) []Card {
	perDeck := 52
	if jokers {
		perDeck += 2
	}
	deck := make([]Card, 0, numDecks*perDeck)

	suits := []string{"Hearts", "Diamonds", "Clubs", "Spades"}
	values := []string{"Ace", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine", "Ten", "Jack", "Queen", "King"}

	for d := 0; d < numDecks; d++ {
		for _, suit := range suits {
			for _, v := range values {
				deck = append(deck, Card{Value: v, Suit: suit})
			}
		}
		if jokers {
			deck = append(deck, Card{Value: "Joker", Suit: "Red"})
			deck = append(deck, Card{Value: "Joker", Suit: "Black"})
		}
	}
	return deck
}

func RemoveCard(deck []Card, index int) []Card {
	return append(deck[:index], deck[index+1:]...)
}
