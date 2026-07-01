package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	DEBUGConf = DebugConfig{
		Items: []DebugItem{
			{
				Fmt:  "%.2fFPS",
				Args: func() []any { return []any{ebiten.ActualFPS()} },
				Inner: []DebugItem{
					{
						Fmt:  "%.2fTPS",
						Args: func() []any { return []any{ebiten.ActualTPS()} },
					},
				},
			},
		},
	}
)

type DebugItem struct {
	Inner []DebugItem
	Args  func() []any
	Fmt   string
}

type DebugConfig struct {
	Items []DebugItem
}

func (d DebugItem) String() string {
	if d.Args == nil {
		return d.Fmt
	}
	return fmt.Sprintf(d.Fmt, d.Args()...)
}
