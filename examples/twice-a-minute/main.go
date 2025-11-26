package main

import (
	"fmt"
	"time"

	"github.com/imunhatep/cronx"
)

func main() {
	// Every 30 seconds
	c1, err := cronx.New("0,30 */1 * * * *",
		cronx.WithLocation(time.Local),
		cronx.WithSeconds(),
	)
	if err != nil {
		panic(err)
	}
	defer c1.Stop()

	fmt.Println("Started cron (0,30 */1 * * * *)…")
	for t := range c1.C {
		fmt.Println("tick at:", t.Format(time.RFC3339))
	}
}
