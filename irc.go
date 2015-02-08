package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

type IrcMessage struct {
	prefix  []byte
	command []byte
	params  [][]byte
	trailer []byte
}

type ClientState int

const (
	nickReceived = ClientState(1 << iota)
	userReceived
)

type ClientChannelMode int

const (
	ChannelModeNone = ClientChannelMode(0)
	ChannelOperator = ClientChannelMode(1 << iota)
	ChannelSpeak
)

type CallbackFunc func(...interface{})

type CallbackRequest struct {
	callback CallbackFunc
	args     []interface{}
}

type Channel struct {
	clients map[*Client]ClientChannelMode
	name    []byte
	topic   []byte
	mutex   sync.Mutex
}

type Client struct {
	state       ClientState
	mode        IrcMode
	conn        net.Conn
	pass        []byte
	nick        []byte
	user        []byte
	realname    []byte
	errbox      chan error
	outbox      chan []byte
	inbox       chan IrcMessage
	callbackbox chan CallbackRequest
	channels    []*Channel
}

type ClientFunc struct {
	f               func(*Client, [][]byte, []byte) (bool, []byte)
	minParams       int
	trailerRequired bool
}

type messageTarget interface {
	send(sender *Client, command string, trailer []byte, params ...[]byte)
}

/* These mutexes will be performance bottlenecks.  Hashing the nick
and channel name (unpredictably), and sharding the maps using the
first two characters would be a simple way to dramatically reduce lock
contention.  Also, maybe it would make sense to use a chan per IRC
channel to handle join/part, instead of a mutex; this could in theory
be faster, but this page
http://www.arkxu.com/post/58998283664/performance-benchmark-go-lang-compares-to-java
implies that it is not in practice.

Also, these are keyed by the result of hashing byte slices, because
golang strings can't actually hold arbitrary binary data and byte slices
can't be hash keys.  And because there are no golang generics,
I can't abstract any of this stuff. FML.
*/

var nickToClient = make(map[string]*Client, 0)
var nickMutex sync.Mutex

var nameToChannel = make(map[string]*Channel, 0)
var channelMutex sync.Mutex

func makeHashKey(k []byte) string {
	//FIXME: handle weird finnish character substitutes
	k = bytes.ToLower(k)
	h := sha256.New()
	return hex.EncodeToString(h.Sum(k))
}

var commandHandlers = map[string]ClientFunc{
	"NICK":    {nickCommand, 1, false},
	"PASS":    {passCommand, 1, false},
	"PRIVMSG": {privmsgCommand, 1, true},
	"QUIT":    {quitCommand, 0, false},
	"USER":    {userCommand, 3, true},
	"JOIN":    {joinCommand, 1, false},
	"PING":    {pingCommand, 1, false},
	"WHOIS":   {whoisCommand, 1, false},
	"WHO":     {whoCommand, 1, false},
	"TOPIC":   {topicCommand, 1, true},
}

func (client *Client) isRegistered() bool {
	nickAndUser := nickReceived | userReceived
	return client.state&nickAndUser == nickAndUser
}

func (client *Client) serverName() []byte {
	//TODO: memoize
	return getHost(client.conn.LocalAddr())
}

func topicCommand(client *Client, params [][]byte, trailer []byte) (bool, []byte) {
	name := params[0]
	channel, ok := getChannel(name)
	if !ok {
		client.numeric(ERR_NOSUCHCHANNEL, name)
	}
	if client.isInChannel(channel) {
		channel.topic = trailer
	} else {
		client.numeric(ERR_CANNOTSENDTOCHAN, name)
	}
	channel.sendMessage(false, client, "TOPIC", trailer, channel.name)
	return false, nil
}

func whoCommand(client *Client, params [][]byte, _ []byte) (bool, []byte) {
	operOnly := false
	if len(params) == 2 && len(params[1]) == 1 && params[1][0] == byte('o') {
		operOnly = true
	}
	mask := params[0]
	if mask[0] == '#' {
		client.channelWhoCommand(mask, operOnly)
	} else {
		client.userWhoCommand(mask, operOnly)
	}
	return false, nil
}

func (client *Client) channelWhoCommand(mask []byte, operOnly bool) {
	channel, ok := getChannel(mask)
	if !ok {
		client.numeric(ERR_NOSUCHCHANNEL, mask)
		return
	}
	channel.mutex.Lock()
	for target, mode := range channel.clients {
		modeStr := []byte("H") // FIXME: this is H for "here" or G for "away"

		if mode&ChannelOperator != 0 {
			modeStr = append(modeStr, byte('@'))
		} else if mode&ChannelSpeak != 0 {
			modeStr = append(modeStr, byte('+'))
		}

		client.numeric(RPL_WHOREPLY,
			channel.name,
			target.user,
			target.host(),
			client.serverName(),
			target.nick,
			modeStr,
			"0",
			target.realname,
		)
	}

	client.numeric(RPL_ENDOFWHO, mask)

	channel.mutex.Unlock()
}

func (client *Client) userWhoCommand(mask []byte, operOnly bool) {
	//Assume that mask does not contain wildcards. On a busy
	//server, a mask containing wildcards could match zillions of
	//users, and that would be too much data to send
	target, ok := getClient(mask)
	if !ok {
		client.numeric(ERR_NOSUCHNICK, mask)
		return
	}
	//we cannot actually send a reply here, because we cannot find
	//a channel that target is in safely.  We need to instead ask
	//target to get back to us about that.

	//It turns out that real IRC servers just return '*' for channel
	//in cases where you don't share a channel with that user, but
	//I don't see why.
	remainder := func(args ...interface{}) {
		channel := args[0].([]byte)
		mode := args[1].(ClientChannelMode)

		if operOnly {
			if mode&ChannelOperator == 0 {
				//this is insane: the RFC has me return
				//nothing here, so the client cannot
				//detect a failure.
				return
			}
		}

		modeStr := []byte("H") // FIXME: this is H for "here" or G for "away"

		if mode&ChannelOperator != 0 {
			modeStr = append(modeStr, byte('@'))
		} else if mode&ChannelSpeak != 0 {
			modeStr = append(modeStr, byte('+'))
		}
		client.numeric(RPL_WHOREPLY,
			channel,
			target.user,
			target.host(),
			client.serverName(),
			target.nick,
			modeStr,
			"0",
			target.realname,
		)

		client.numeric(RPL_ENDOFWHO, target)
	}
	target.requestCallback(client.getChannelAndMode, client, remainder)
}

func (client *Client) getChannelAndMode(args ...interface{}) {
	replyTo := args[0].(*Client)
	replyWith := args[1].(CallbackFunc)
	if len(client.channels) == 0 {
		replyTo.requestCallback(replyWith, "*", ChannelModeNone)
	} else {
		replyTo.requestCallback(replyWith, client.channels[0].name,
			ChannelModeNone)
	}
}

func whoisCommand(client *Client, params [][]byte, _ []byte) (bool, []byte) {
	//Parameters: [ <target> ] <mask> *( "," <mask> )
	start := 0
	if len(params) >= 2 && !validNick(params[0]) {
		//ignore target server parameter
		start = 1
	}
	for _, mask := range params[start:] {
		//as per usual, assume the mask does not contain wildcards
		target, ok := getClient(mask)
		if ok {
			client.numeric(RPL_WHOISUSER, target.nick, target.user,
				target.host(),
				target.realname)
		} else {
			client.numeric(ERR_NOSUCHNICK, target)
		}
	}
	return false, nil
}

func pingCommand(client *Client, params [][]byte, _ []byte) (bool, []byte) {
	server := params[0]
	client.send(nil, "PONG", nil, server)
	return false, nil
}

func passCommand(client *Client, params [][]byte, _ []byte) (bool, []byte) {
	if client.isRegistered() {
		client.numeric(ERR_ALREADYREGISTRED)
		return false, nil
	}
	client.pass = params[0]
	return false, nil
}

//WTF, golang -- I see assert used in the stdlib?!
func assert(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}

func (channel *Channel) send(sender *Client, command string, trailer []byte, params ...[]byte) {
	channel.sendMessage(true, sender, command, trailer, params...)
}

func (channel *Channel) sendMessage(skipSender bool, sender *Client, command string, trailer []byte, params ...[]byte) {
	id := sender.formatClientId()
	clients := make([]*Client, 0)
	channel.mutex.Lock()
	for client, _ := range channel.clients {
		if client != sender || !skipSender {
			clients = append(clients, client)
		}
	}
	channel.mutex.Unlock()
	message := IrcMessage{
		prefix:  id,
		command: []byte(command),
		params:  params,
		trailer: trailer,
	}
	for _, client := range clients {
		client.outbox <- message.wireFormat()
	}
}

func (client *Client) send(sender *Client, command string, trailer []byte, params ...[]byte) {
	var id []byte
	if sender != nil {
		id = sender.formatClientId()
	} else {
		id = nil
	}
	message := IrcMessage{
		prefix:  id,
		command: []byte(command),
		params:  params,
		trailer: trailer,
	}
	client.outbox <- message.wireFormat()
}

func quitCommand(client *Client, params [][]byte, trailer []byte) (bool, []byte) {
	if trailer == nil {
		trailer = []byte("(no reason)")
	}
	client.send(nil, "ERROR", []byte("quitting"))
	return true, trailer
}

func userCommand(client *Client, params [][]byte, trailer []byte) (bool, []byte) {
	if client.state&userReceived != 0 {
		client.numeric(ERR_ALREADYREGISTRED)
		return false, nil
	}
	modeString := string(params[1])
	mode := 0
	if modeString != "*" {
		newMode, err := strconv.Atoi(modeString)
		if err == nil {
			mode = newMode
		} else {
			// According to the protocol, I should return
			// client.numeric(ERR_UMODEUNKNOWNFLAG).  But since
			// IRC clients don't follow the protocol, we'll just
			// silently ignore this case
		}
	}
	client.user = params[0]
	client.mode = IrcMode(mode) & InitialModeMask
	//params[2] is unused
	client.realname = trailer

	client.state |= userReceived

	if client.state&nickReceived != 0 {
		client.welcome()
	}
	return false, nil
}

func (client *Client) getChannel(name []byte) *Channel {
	for _, channel := range client.channels {
		if bytes.Compare(channel.name, name) == 0 {
			return channel
		}
	}
	return nil
}

func getClient(name []byte) (*Client, bool) {
	key := makeHashKey(name)
	nickMutex.Lock()
	client, ok := nickToClient[key]
	nickMutex.Unlock()
	return client, ok
}

func (client *Client) isInChannel(needle *Channel) bool {
	for _, channel := range client.channels {
		if needle == channel {
			return true
		}
	}
	return false
}

func privmsgCommand(client *Client, params [][]byte, msg []byte) (bool, []byte) {
	if !client.isRegistered() {
		client.numeric(ERR_NOTREGISTERED)
		return false, nil
	}

	name := params[0]
	var recipient messageTarget
	if name[0] == '#' {
		channel := client.getChannel(name)
		if channel == nil {
			client.numeric(ERR_NOSUCHCHANNEL, name)
			return false, nil
		}
		if !client.isInChannel(channel) {
			client.numeric(ERR_CANNOTSENDTOCHAN, name)
			return false, nil
		}
		recipient = channel
	} else if recipient == nil {
		var ok bool
		recipient, ok = getClient(name)
		if ok {
			client.numeric(ERR_NOSUCHNICK, name)
			return false, nil
		}
	}
	recipient.send(client, "PRIVMSG", msg, name)
	return false, nil
}

func (channel *Channel) part(client *Client) {
	channel.mutex.Lock()
	delete(channel.clients, client)
	channel.mutex.Unlock()
}

type ChannelMember struct {
	nick []byte
	mode ClientChannelMode
}

func (channel *Channel) names(client *Client) {
	channel.mutex.Lock()
	nicks := make([]ChannelMember, 0, len(channel.clients))
	for client, mode := range channel.clients {
		nicks = append(nicks, ChannelMember{client.nick, mode})
	}
	channel.mutex.Unlock()
	//TODO: check if RPL_NOTOPIC is sent on join for missing topic
	//TODO: is an empty topic different from no topic?
	client.numeric(RPL_TOPIC, channel.name, channel.topic)
	//send the channels in max-512-byte chunks; 10 bytes are needed for
	//other params, spaces, and the terminator
	buf := make([]byte, 0, 502-len(channel.name))
	eq := []byte("=")
	//353 = #channelName :[the list]
	for _, member := range nicks {
		nick := member.nick
		mode := member.mode
		//flush nicks if necessary
		if len(buf)+2+len(nick) > cap(buf) {
			client.numeric(RPL_NAMREPLY, eq, channel.name, buf)
			buf = buf[:0]
		}
		if len(buf) > 0 {
			buf = append(buf, ' ')
		}
		if mode&ChannelOperator != 0 {
			buf = append(buf, '@')
		} else if mode&ChannelSpeak != 0 {
			buf = append(buf, '+')
		}
		buf = append(buf, nick...)
	}
	if len(nicks) != 0 {
		//flush remaining nicks
		client.numeric(RPL_NAMREPLY, eq, channel.name, buf)
	}

	client.numeric(RPL_ENDOFNAMES, channel.name)
}

func (client *Client) formatClientId() []byte {
	buflen := len(client.nick) + len(client.user) + 1
	buf := make([]byte, 0, buflen)
	buf = append(buf, client.nick...)
	buf = append(buf, '!')
	buf = append(buf, client.user...)
	buf = append(buf, '@')
	buf = append(buf, client.host()...)
	return buf
}

func (channel *Channel) join(client *Client) {
	channel.mutex.Lock()
	_, ok := channel.clients[client]
	if ok {
		channel.mutex.Unlock()
		//user is already in channel
		client.numeric(ERR_USERONCHANNEL, client.nick, channel.name)
		return
	}
	mode := ChannelModeNone
	if len(channel.clients) == 0 {
		mode = ChannelOperator
	}
	channel.clients[client] = mode
	channel.mutex.Unlock()

	channel.sendMessage(false, client, "JOIN", nil, channel.name)

	client.channels = append(client.channels, channel)

	channel.names(client)
}

func validChannelName(name []byte) bool {
	//TODO: this does not implement the true IRC rules; it only allows
	//traditional #foo channels with reasonable names

	if len(name) == 0 || len(name) > 50 {
		return false
	}
	if name[0] != '#' {
		return false
	}
	for i, c := range name {
		if i == 0 {
			continue
		}
		if !(isLetter(c) || isDigit(c) || isSpecial(c) || c == '-') {
			return false
		}
	}
	return true
}

func getChannel(name []byte) (*Channel, bool) {
	key := makeHashKey(name)
	channelMutex.Lock()
	defer channelMutex.Unlock()
	channel, ok := nameToChannel[key]
	return channel, ok
}

func joinCommand(client *Client, params [][]byte, _ []byte) (bool, []byte) {
	if !client.isRegistered() {
		//can't join a channel when you have not yet
		//registered a nickname
		client.numeric(ERR_NOTREGISTERED)
		return false, nil
	}

	names := params[0]
	//JOIN 0 parts from all channels
	if len(names) == 1 && names[0] == '0' {
		for _, channel := range client.channels {
			channel.part(client)
		}
		client.channels = make([]*Channel, 0)
		return false, nil
	}

	start := 0
	for i, c := range names {
		found := false
		if i == len(names)-1 {
			i++
			found = true
		} else if c == ',' {
			found = true
		}
		if found == false {
			continue
		}
		name := names[start:i]
		start = i + 1
		if client.mode&IrcModeRestricted != 0 {
			client.numeric(ERR_RESTRICTED)
		}
		if !validChannelName(name) {
			client.numeric(ERR_NOSUCHCHANNEL, name)
			continue
		}

		key := makeHashKey(name)
		channelMutex.Lock()
		channel, ok := nameToChannel[key]
		if !ok {
			channel = &Channel{
				clients: make(map[*Client]ClientChannelMode),
				name:    name,
			}
			nameToChannel[key] = channel
		}
		channelMutex.Unlock()
		channel.join(client)
	}
	return false, nil
}

func (client *Client) host() []byte {
	return getHost(client.conn.RemoteAddr())
}

func getHost(addr net.Addr) []byte {
	remoteAddr := fmt.Sprintf("%s", addr)
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		panic(fmt.Sprint("failed to parse %s", remoteAddr))
	}
	return []byte(fmt.Sprint(host))
}

func (client *Client) welcome() {
	client.numeric(RPL_WELCOME, client.nick, client.user, client.host())
}

func validNick(nick []byte) bool {
	if len(nick) == 0 || len(nick) > 9 {
		return false
	}
	if !(isLetter(nick[0]) || isSpecial(nick[0])) {
		return false
	}
	for _, c := range nick {
		if !(isLetter(c) || isSpecial(c) || isDigit(c) || c == '-') {
			return false
		}
	}
	return true
}

func nickCommand(client *Client, params [][]byte, _ []byte) (bool, []byte) {
	nick := params[0]
	// check that nick is legit
	if !validNick(nick) {
		client.numeric(ERR_ERRONEUSNICKNAME, nick)
		return false, nil
	}

	nickMutex.Lock()
	//this holds the mutex for slightly longer than necessary but
	//otherwise I would have to call it in three places and that
	//would be sad
	defer nickMutex.Unlock()

	key := makeHashKey(nick)
	if _, ok := nickToClient[key]; ok {
		client.numeric(ERR_NICKNAMEINUSE, nick)
		return false, nil
	}

	if client.state&nickReceived != 0 {
		//change of nick
		if client.mode&IrcModeRestricted != 0 {
			client.numeric(ERR_RESTRICTED)
			return false, nil
		}
		oldKey := makeHashKey(client.nick)
		delete(nickToClient, oldKey)
		//TODO: send out change messages to all clients
		//in users' channels
	} else {
		client.state |= nickReceived
	}
	client.nick = nick
	nickToClient[key] = client
	if client.state&userReceived != 0 {
		client.welcome()
	} else {
		client.sendToChannels("NICK", nick)
	}
	return false, nil
}

func (client *Client) requestCallback(callback CallbackFunc, args ...interface{}) {
	client.callbackbox <- CallbackRequest{
		callback: callback,
		args:     args,
	}
}

func (client *Client) readLoop() {
	remainder := make([]byte, 0)
	for {
		//512 bytes is the max len of an IRC message
		data := make([]byte, 512)
		n, err := client.conn.Read(data)
		if err != nil {
			client.errbox <- err
			// we've hit an error, so reading any more would
			// be pointless
			return
		}
		data = data[:n]
		if len(remainder) != 0 {
			// copy over any bytes that were remainder
			// from previous packets
			newdata := make([]byte, len(remainder)+len(data))
			copy(newdata, remainder)
			copy(newdata[len(remainder):], data)
			data = newdata
		}

		// iterate through the message until we find
		// end-of-message.
		prev := byte(0)
		start := 0
		for i, c := range data {
			if c == '\n' {
				//done message
				packet := make([]byte, 0, i-start)
				if prev == '\r' {
					packet = append(packet, data[start:i-1]...)
				} else {
					// incomprehensibly, some IRC clients manage
					// to fuck this up and forget the \r
					packet = append(packet, data[start:i]...)
				}
				if i == start+1 {
					// "RFC: Empty messages are silently ignored"
					start = i + 1
					continue
				}
				message, err := parseIrcMessage(packet)
				if err != nil {
					fmt.Printf("Error parsing message: %s %s\n", data, err)
				} else {
					// send message if we read some.
					fmt.Printf("Parsed message: %s\n", message)
					client.inbox <- *message
				}
				start = i + 1
			}
			prev = c
		}
		remainder = data[start:]
	}
}

func byteSprintf(template string, args ...interface{}) []byte {
	buf := make([]byte, 0, 512)
	arg := 0
	prev := 'x'
	for _, c := range template {
		if prev == '%' {
			if c == 's' {
				val := args[arg]
				arg++
				switch t := val.(type) {
				case []byte:
					buf = append(buf, t...)
				default:
					str := fmt.Sprint(t)
					buf = append(buf, []byte(str)...)
				}
			} else if c == '%' {
				buf = append(buf, '%')
			} else {
				panic("bogus template")
			}
		} else if c != '%' {
			buf = append(buf, byte(c))
		}
		prev = c
	}
	return buf
}

func (client *Client) numeric(code NumericMessage, args ...interface{}) {
	//Some messages from the RFC are multi-parameter; some have
	//leading colons; and some are just plain text.  Much
	//consistent.  Assume that any colon in the message is a final arg
	//indicator (which is, I believe, the case)
	//Add leading colons where necessary.
	colon := ":"
	if strings.IndexByte(code.template, ':') != -1 {
		colon = ""
	}
	//we would like to use sprintf here, but we can't, because we
	//might have invalid utf-8, which sprintf would helpfully
	//mangle.  Thanks, sprintf!

	//also, we would like to call this as sprintf(template,
	//server, nick, args...), but golang can't do that for some
	//idiotic reason.

	args = append(args, nil, nil, nil)
	copy(args[3:], args[:len(args)-3])
	args[0] = client.serverName()
	args[1] = []byte(fmt.Sprintf("%03d", code.number))
	args[2] = client.nick

	template := ":%s %s %s " + colon + code.template + "\r\n"
	reply := byteSprintf(template, args...)
	assert(len(reply) <= 512, "IRC messages must be shorter than 512 bytes.")

	client.outbox <- reply
}

func (msg *IrcMessage) wireFormat() []byte {
	buf := make([]byte, 0, 512)
	if msg.prefix != nil {
		buf = append(buf, ':')
		buf = append(buf, msg.prefix...)
		buf = append(buf, ' ')
	}
	buf = append(buf, msg.command...)
	for _, param := range msg.params {
		buf = append(buf, ' ')
		buf = append(buf, param...)
	}
	if msg.trailer != nil {
		buf = append(buf, ' ')
		buf = append(buf, ':')
		buf = append(buf, msg.trailer...)
	}
	//it's possible that at this point, the message is already too long,
	//because a client can send a 512-byte privmsg that's then prepended
	//with a prefix.  Truncate as necessary.
	buf = buf[:510]

	buf = append(buf, '\r')
	buf = append(buf, '\n')
	return buf
}

func (client *Client) sendToChannels(command string, message []byte) {
	for _, channel := range client.channels {
		channel.send(client, command, message)
	}
}

func (client *Client) disconnect(message []byte) {
	for _, channel := range client.channels {
		channel.mutex.Lock()
		delete(channel.clients, client)
		channel.mutex.Unlock()
		channel.send(client, "QUIT", message)
	}
	client.conn.Close()
	nickMutex.Lock()
	key := makeHashKey(client.nick)
	delete(nickToClient, key)
	nickMutex.Unlock()
}

func (client *Client) loop() {
	go client.readLoop()
	for {
		select {
		case outgoing := <-client.outbox:
			fmt.Printf("Wrote %s\n", outgoing)
			client.conn.Write(outgoing)
		case incoming := <-client.inbox:
			command := string(incoming.command)
			handler, ok := commandHandlers[command]
			if !ok {
				fmt.Printf("unhandled command %s\n", command)
				continue
			}
			if len(incoming.params) < handler.minParams {
				client.numeric(ERR_NEEDMOREPARAMS, command)
				fmt.Printf("too few params (%d/%d) for command %s\n",
					len(incoming.params), handler.minParams, command)
				continue
			}
			if incoming.trailer == nil && handler.trailerRequired {
				fmt.Printf("missing trailer for command %s\n",
					command)
				continue

			}
			params := incoming.params
			trailer := incoming.trailer
			if ok, msg := handler.f(client, params, trailer); ok {
				// whatever the client sent, it's bad enough
				// that we have to terminate them
				client.disconnect(msg)
				return
			}
		case err := <-client.errbox:
			strerr := fmt.Sprint(err)
			client.disconnect([]byte(strerr))
			return

		case callback := <-client.callbackbox:
			callback.callback(callback.args...)
		}
	}
}

func handleConnection(conn net.Conn) {
	// Here, we want to read to the connection, but also
	// be able to write to it.  It's OK if we block
	// in this goroutine as long as some other routine
	// can write to the writer

	client := Client{
		conn:        conn,
		outbox:      make(chan []byte, 10),
		inbox:       make(chan IrcMessage, 10),
		errbox:      make(chan error, 10),
		callbackbox: make(chan CallbackRequest, 10),
	}

	client.loop()
}

func main() {
	ln, err := net.Listen("tcp", ":6667")
	if err != nil {
		fmt.Printf("Listen failed: %s\n", err)
		os.Exit(1)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("Accept failed: %s\n", err)
			os.Exit(1)
		}
		go handleConnection(conn)
	}
}
