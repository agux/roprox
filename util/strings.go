package util

import "strings"

//Rot13 Cipher obscures the characters in a string by rotating characters
//13 places in the alphabet. Other characters are not changed.
func Rot13(str string) string {
	return strings.Map(runeRot13, str)
}

func runeRot13(r rune) rune {
	if r >= 'a' && r <= 'z' {
		// Rotate lowercase letters 13 places.
		if r > 'm' {
			return r - 13
		} else {
			return r + 13
		}
	} else if r >= 'A' && r <= 'Z' {
		// Rotate uppercase letters 13 places.
		if r > 'M' {
			return r - 13
		} else {
			return r + 13
		}
	}
	// Do nothing.
	return r
}
