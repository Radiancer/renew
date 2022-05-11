package main

import (
	"github.com/fatih/color"
)

var (

	// TODO: support more colors
	colorMap = map[string]color.Attribute{
		"red":     color.FgRed,
		"green":   color.FgGreen,
		"yellow":  color.FgYellow,
		"blue":    color.FgBlue,
		"magenta": color.FgMagenta,
		"cyan":    color.FgCyan,
		"white":   color.FgWhite,
		"hiblue":  color.FgHiBlue,
	}
)

func getColor(name string) color.Attribute {
	if v, ok := colorMap[name]; ok {
		return v
	}
	return color.FgWhite
}
func colorPrint(colstr string, str string) {
	col := getColor(colstr)
	color.New(col).Println(str)
}
