package main

import (
	"1szt/flags"
	"1szt/motd"
	_ "1szt/quic"
	_ "1szt/raptorq"
)

func main() {
	motd.Run()
	flags.Run()
}
