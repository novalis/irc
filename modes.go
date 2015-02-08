package main

type IrcMode int

const (
	IrcModeAway = IrcMode(1 << iota)
	IrcModeInvisible
	IrcModeWallops
	IrcModeRestricted
	IrcModeOperator
	IrcModeLocalOperator
	IrcModeReceivesServerNotices
)

const (
	InitialModeMask = IrcModeInvisible | IrcModeWallops
	IrcModeLetters  = "aiwroOs"
)
