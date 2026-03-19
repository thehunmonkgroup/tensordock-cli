package main

import (
	"log"

	"github.com/thehunmonkgroup/tensordock-cli/commands"
)

func main() {
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	commands.Execute()
}
