package rng

func BuildCharset(lowers, uppers, numbers, symbols bool) []byte {
	var b []byte
	if lowers {
		b = append(b, []byte("abcdefghijklmnopqrstuvwxyz")...)
	}
	if uppers {
		b = append(b, []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")...)
	}
	if numbers {
		b = append(b, []byte("0123456789")...)
	}
	if symbols {
		b = append(b, []byte("!#$%&()*+,-./:;<=>?@[]^_{|}~")...)
	}
	return b
}
