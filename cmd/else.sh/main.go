// Command else.sh hands unknown shell commands to your coding agent.
package main

import (
	"os"

	"github.com/cristi-boboc/else.sh/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
