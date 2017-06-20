package stringencoder

type EncodableAsString interface {
	GetEncodedString() string
	NewFromEncodedString(encoded string)
}