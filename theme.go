package main

import (
	_ "embed"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

//go:embed images/start.svg
var startIconBytes []byte
var startIconRes fyne.Resource

//go:embed images/stop.svg
var stopIconBytes []byte
var stopIconRes fyne.Resource

//go:embed images/mine.svg
var mineIconBytes []byte
var mineIconRes fyne.Resource

//go:embed images/deposit.svg
var depositIconBytes []byte
var depositIconRes fyne.Resource

//go:embed images/withdraw.svg
var withdrawIconBytes []byte
var withdrawIconRes fyne.Resource

const (
	StartIcon    fyne.ThemeIconName = "start.svg"
	StopIcon     fyne.ThemeIconName = "stop.svg"
	MineIcon     fyne.ThemeIconName = "mine.svg"
	DepositIcon  fyne.ThemeIconName = "deposit.svg"
	WithdrawIcon fyne.ThemeIconName = "withdraw.svg"
)

type switchboardTheme struct{}

func (t switchboardTheme) Init() {
	startIconRes = theme.NewThemedResource(fyne.NewStaticResource(string(StartIcon), startIconBytes))
	stopIconRes = theme.NewThemedResource(fyne.NewStaticResource(string(StopIcon), stopIconBytes))
	mineIconRes = theme.NewThemedResource(fyne.NewStaticResource(string(MineIcon), mineIconBytes))
	depositIconRes = theme.NewThemedResource(fyne.NewStaticResource(string(DepositIcon), depositIconBytes))
	withdrawIconRes = theme.NewThemedResource(fyne.NewStaticResource(string(WithdrawIcon), withdrawIconBytes))
}

func (t switchboardTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameSeparator:
		return color.RGBA{0x00, 0x00, 0x00, 0x00}
	case theme.ColorNameSelection:
		return theme.DefaultTheme().Color(theme.ColorNameHover, variant)
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t switchboardTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	switch name {
	case StartIcon:
		return startIconRes
	case StopIcon:
		return stopIconRes
	case MineIcon:
		return mineIconRes
	case DepositIcon:
		return depositIconRes
	case WithdrawIcon:
		return withdrawIconRes
	default:
		return theme.DefaultTheme().Icon(name)
	}
}

func (t switchboardTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t switchboardTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
