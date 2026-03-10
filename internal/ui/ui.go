package ui

import (
	"github.com/pterm/pterm"
)

// Banner muestra el banner principal de la aplicación
func Banner() {
	pterm.Println()
	pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("Pen", pterm.NewStyle(pterm.FgCyan)),
		pterm.NewLettersFromStringWithStyle("pot", pterm.NewStyle(pterm.FgMagenta)),
	).Render()

	pterm.DefaultCenter.WithCenterEachLineSeparately().Println(
		pterm.LightWhite("Manager — Instalador interactivo de Penpot en Docker"),
	)
	pterm.Println()
}

// Success muestra un mensaje de éxito
func Success(msg string) {
	pterm.Success.Println(msg)
}

// Error muestra un mensaje de error
func Error(msg string) {
	pterm.Error.Println(msg)
}

// Info muestra un mensaje informativo
func Info(msg string) {
	pterm.Info.Println(msg)
}

// Warning muestra una advertencia
func Warning(msg string) {
	pterm.Warning.Println(msg)
}

// Separator imprime una línea separadora
func Separator() {
	pterm.Println()
	pterm.ThemeDefault.PrimaryStyle.Println("────────────────────────────────────────────")
	pterm.Println()
}

// Spinner crea un spinner con un mensaje
func Spinner(msg string) (*pterm.SpinnerPrinter, error) {
	return pterm.DefaultSpinner.WithText(msg).Start()
}
