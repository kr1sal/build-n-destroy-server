package game

import (
	"context"
	"fmt"
	"time"
)

func Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second / 60)
	defer ticker.Stop()
	ticks := 0

	for {
		select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ticks++
				fmt.Printf("Tick %v\n", ticks)
		}
	}
}