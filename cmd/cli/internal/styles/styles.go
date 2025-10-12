package styles

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Color palette
	PrimaryColor = lipgloss.Color("#00D4AA")
	AccentColor  = lipgloss.Color("#F59E0B")
	TextColor    = lipgloss.Color("#E5E7EB")
	MutedColor   = lipgloss.Color("#9CA3AF")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	BannerStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			Align(lipgloss.Left).
			Padding(1, 0)

	HelpStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			Underline(true).
			MarginBottom(1)

	CommandStyle = lipgloss.NewStyle().
			Foreground(AccentColor).
			Bold(true)

	DescriptionStyle = lipgloss.NewStyle().
				Foreground(MutedColor)
)

// ASCII art banner for Kibaship
const banner = `
██╗  ██╗██╗██████╗  █████╗ ███████╗██╗  ██╗██╗██████╗
██║ ██╔╝██║██╔══██╗██╔══██╗██╔════╝██║  ██║██║██╔══██╗
█████╔╝ ██║██████╔╝███████║███████╗███████║██║██████╔╝
██╔═██╗ ██║██╔══██╗██╔══██║╚════██║██╔══██║██║██╔═══╝
██║  ██╗██║██████╔╝██║  ██║███████║██║  ██║██║██║
╚═╝  ╚═╝╚═╝╚═════╝ ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═╝╚═╝
`

const subtitle = "⚡ The complete paas platform powered by kubernetes ⚡"

// PrintBanner displays the Kibaship ASCII art banner
func PrintBanner() {
	fmt.Print(BannerStyle.Render(banner))
	fmt.Print(BannerStyle.Render(subtitle))
	fmt.Println()
}
