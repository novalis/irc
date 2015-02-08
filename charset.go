package main

func isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

func isLetter(c byte) bool {
	return ('A' <= c && c < 'Z') ||
		('a' <= c && c < 'z')
}

func isSpecial(c byte) bool {
	return c == '_' || c == '[' || c == ']' || c == '\\' ||
		c == '`' || c == '^' || c == '{' || c == '}'
}
