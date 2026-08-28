package domain

import "time"



type Move struct {
	GameID string
	Ply int
	Color Color
	SAN string // ex: "Nf3"
	UCI string // ex: "e2e4"
	FENAfter string
	PlayedAt time.Time
}

type Color int
const (
	White Color = iota
	Black
)

var colors = map[Color]string{
	White: "white",
	Black: "black",
}

func (c Color) String() string {
	return colors[c]
}