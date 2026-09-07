package main

import (
	"time"

	bs "charm.land/bubbles/v2/spinner"
	"github.com/bisonschweizag/gws-cli/internal/spinner"
)

const spinDuration = 2 * time.Second

type spinnerDemo struct {
	title string
	sp    bs.Spinner
}

var spinnerDemos = []spinnerDemo{
	{title: "Line", sp: bs.Line},
	{title: "Dot", sp: bs.Dot},
	{title: "MiniDot", sp: bs.MiniDot},
	{title: "Pulse", sp: bs.Pulse},
	{title: "Points", sp: bs.Points},
	{title: "Globe", sp: bs.Globe},
	{title: "Moon", sp: bs.Moon},
	{title: "Monkey", sp: bs.Monkey},
	// {title: "Meter", sp: bs.Meter},
	// {title: "Ellipsis", sp: bs.Ellipsis},
}

func main() {
	println("Spinners Demo")
	for _, demo := range spinnerDemos {
		spin(demo.title, demo.sp)
	}
}

func spin(title string, spinnerConfig bs.Spinner) {
	sp := spinner.Start(title, spinnerConfig)
	time.Sleep(spinDuration)
	sp.Stop()
}
