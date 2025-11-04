package main

import (
	"fmt"
	"time"

	"github.com/imunhatep/cronx"
)

func main() {
	// Every 2 minutes
	c1, err := cronx.New("*/1 * * * * *",
		cronx.WithLocation(time.Local),
		cronx.WithSeconds(),
	)
	if err != nil {
		panic(err)
	}
	defer c1.Stop()

	fmt.Println("Started cron (*/1 * * * * *)…")
	for t := range c1.C {
		fmt.Println("tick at:", t.Format(time.RFC3339))
	}
}
