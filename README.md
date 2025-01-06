[![Build](https://github.com/lost-woods/random/actions/workflows/publish.yml/badge.svg)](https://github.com/lost-woods/random/actions/workflows/publish.yml)

# random
True RNG REST API Server
Listens on port `777`

Environment Variables:
`SERIAL_BAUD_RATE` 300
`SERIAL_READ_TIMEOUT` 10
`SERIAL_DEVICE_NAME` /dev/TrueRNG (or wherever the TrueRNG serial device is located)
`API_KEY` empty (or string that is expected in the request header (with key `X-API-KEY`) to grant access to the API)


Supports the following endpoints:

### Random Numbers
`/` Generates a random number from 1-100

Optional parameters:
- `min` inclusive
- `max` inclusive


### Random Bytes
`/bytes` Generates a random byte

Optional parameters:
- `size` how many bytes to generate


### Random Cards
`/cards` Draws a random card from a deck

Optional parameters:
- `decks` how many decks should be included
- `jokers` whether jokers should be included
- `cards` how many cards to draw


### Random Strings
`/strings` Generates a random string

Optional parameters:
- `size` the size of the string
- `lowercase` whether to include lower case characters
- `uppercase` whether to include upper case characters
- `numbers` whether to include number characters
- `symbols` whether to include symbol characters


### Random Event
`/percent` Enter a percent chance and check for a pass or fail

Optional parameters:
- `percent` the percent chance of success
