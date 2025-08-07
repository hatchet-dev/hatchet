package main

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/cmd/hatchet/cli"
)

func main() {
	fmt.Printf(`  
  _   _       _       _          _   
 | | | | __ _| |_ ___| |__   ___| |_ 
 | |_| |/ _` + "`" + ` | __/ __| '_ \ / _ \ __|
 |  _  | (_| | || (__| | | |  __/ |_ 
 |_| |_|\__,_|\__\___|_| |_|\___|\__|

`)

	cli.Execute()
}
