package main

import (
	"reflect"
	"testing"
)

func TestParseIrcCommand(t *testing.T) {
	cases := []struct {
		in   string
		want *IrcMessage
	}{
		{"Hello", nil},
		{"Hello  ", nil},
		{"", nil},
		{"H", nil},
		{"H \000", nil},
		{"H :\000", nil},
		{"NICK :nick\n", nil},
		{"NICK :nick", &IrcMessage{
			command: []byte("NICK"),
			params:  [][]byte{},
			trailer: []byte("nick"),
		}},
		{"0", nil},
		{"01", nil},
		{"012", nil},
		{"Hello ", &IrcMessage{
			command: []byte("Hello"),
			params:  [][]byte{},
		}},
		{"Hello :world", &IrcMessage{
			command: []byte("Hello"),
			params:  [][]byte{},
			trailer: []byte("world"),
		}},
		{"Hello :world ", &IrcMessage{
			command: []byte("Hello"),
			params:  [][]byte{},
			trailer: []byte("world "),
		}},
		{"Hello :world world", &IrcMessage{
			command: []byte("Hello"),
			params:  [][]byte{},
			trailer: []byte("world world"),
		}},
		{"Hello :world :world", &IrcMessage{
			command: []byte("Hello"),
			params:  [][]byte{},
			trailer: []byte("world :world"),
		}},
		{"Hello world :message", &IrcMessage{
			command: []byte("Hello"),
			params:  [][]byte{[]byte("world")},
			trailer: []byte("message"),
		}},
		//we intentionally want to fail on messages with prefix
		{":xprefix Hello world :message", nil},
	}
	for _, c := range cases {
		got, _ := parseIrcMessage([]byte(c.in))
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Failed to parse(%q) == %q, want %q", c.in, got, c.want)
		}
	}
}
