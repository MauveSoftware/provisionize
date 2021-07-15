package utils

// ReverseString reverses a string
func ReverseString(str string) string {
	res := make([]byte, len(str))
	len := len(str)

	for i := 0; i < len/2; i++ {
		res[i] = str[len-1-i]
		res[len-1-i] = str[i]
	}

	if len%2 > 0 {
		res[len/2] = str[len/2]
	}

	return string(res)
}
