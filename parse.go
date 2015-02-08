package main

import (
	"errors"
)

// <command>  ::= <letter> { <letter> } | <number> <number> <number>
// also tests for and removes the leading space for params
func parseIrcCommand(data []byte) ([]byte, []byte, error) {
	command := make([]byte, 0)
	if isDigit(data[0]) {
		//command is a 3 digit command
		if len(data) < 4 {
			return nil, nil, errors.New("too short numeric command")
		}
		if !(isDigit(data[1]) && isDigit(data[2])) {
			return nil, nil, errors.New("numeric commands are three digits")
		}
		if data[3] != ' ' {
			return nil, nil, errors.New("commands must end in space")
		}
		command = data[:3]
		data = data[4:]
	} else {
		for i, c := range data {
			if !isLetter(c) {
				if c != ' ' {
					return nil, nil, errors.New("commands must end in space")
				}
				command = data[:i]
				data = data[i+1:]
				break
			}
		}
		if len(command) == 0 {
			return nil, nil, errors.New("Command must consist of 1 or more letters")
		}
	}
	return command, data, nil
}

func parseIrcMessage(data []byte) (*IrcMessage, error) {
	// a message has an optional prefix, but we're going to
	// forbid messages with prefixes, since they are only useful
	// in a multi-server context

	if len(data) < 2 {
		return nil, errors.New("too short")
	}

	if data[0] == ':' {
		return nil, errors.New("incoming message prefixes are unsupported")
	}

	command, data, err := parseIrcCommand(data)
	if err != nil {
		return nil, err
	}

	// parse space-separated params
	newParam := true
	lastParam := false
	start := 0
	params := make([][]byte, 0)
	for i, c := range data {
		if newParam && c == ':' {
			start = i + 1
			lastParam = true
			newParam = false
			continue
		}
		if c == 0 || c == '\r' || c == '\n' {
			return nil, errors.New("character is forbidden: " + string(c))
		} else if c == ' ' && !lastParam {
			// consume param
			if start == i {
				// this is an empty param, which is forbidden
				return nil, errors.New("forbidden empty param")
			}
			param := data[start:i]
			params = append(params, param)
			start = i + 1
			newParam = true
		} else {
			newParam = false
		}
	}
	//consume last param, if any
	var trailer []byte
	if lastParam {
		trailer = data[start:]
	} else if start != len(data) {
		params = append(params, data[start:])
	}

	return &IrcMessage{
		command: command,
		params:  params,
		trailer: trailer,
	}, nil
}
