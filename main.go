package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"golang.org/x/term"
)

type model struct {
	loaded   bool           // whether or not the viewport is loaded
	viewport viewport.Model // mostly holds glamour output
	fitWidth int            // best fit content width max 80ch
	// terminal dims:
	cmdWidth  int
	cmdHeight int
}

const (
	host                       = ""
	port                       = "22"
	maxWidth                   = 80
	useHighPerformanceRenderer = true

	headerHeight = 4
	footerHeight = 4

	top = `# hey, i'm max

I study computer science and philosophy at Suffolk University, and design and develop web tools in my spare time. I can't comment on their usefulness, but they're hopefully a little cool.

## Recent Projects`

	prog1 = `### Hypersearch

Hypersearch is a Chromium extension that provides power-user search tools. Slim down and streamline Google search result pages by filtering out spam results and blocking unnecessary info cards.`

	prog2 = `### asciish

Asciish is a Vite/Rollup extension that provides build-time Unicode injection using shortcodes. This helps to keep source code UTF-8 compliant while allowing for the use of complex Unicode characters on a webpage.`

	prog3 = `### escape-time

Escape time is a small WebGL fractal explorer thrown together over a weekend for a Suffolk University Math Society presentation. It supports a few different fractals and was mostly a WebGL learning experience.`

	prog4 = `### Clippy

Clippy.mov is a web-based video editor built using FFmpeg.wasm. I stopped developing it after the new school semester started, and recently came back to it. It's in active development.`

	bot = `## Want to get in touch?

Thanks, that's awesome! :D
`
)

var (
	glamourRenderer  *glamour.TermRenderer // current renderer
	ApplyNormal      = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#27272A", Dark: "#A1A1AA"}).Render
	ApplySubtle      = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#52525B", Dark: "#71717A"}).Render
	ApplyMuted       = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#D4D4D8", Dark: "#3F3F46"}).Render
	ApplyHighlight   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#B824BB", Dark: "#F93EFD"}).Render
	tableBorderColor = lipgloss.AdaptiveColor{Light: "#D4D4D8", Dark: "#3F3F46"}

	projRender1, projRender2, projRender3, projRender4 string
	flex                                               string
)

// loads server
func main() {
	// set glamour env
	os.Setenv("GLAMOUR_STYLE", "./style.json")

	// set up a new wish server. This allows us to serve a terminal UI over SSH
	srv, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		// ED25519 key generated by default
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)

	if err != nil {
		log.Error("Couldn't start server", err)
	}

	// allocate a channel for signals, and forward Interrupt, SIGINT and SIGTERM signals to it
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("Server started @", "host", host, "port", port)

	// this goroutine listens for incoming connections in a concurrent fashion during server startup
	go func() {
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", err)
			// if there's an error with starting the server, send a signal to the done channel
			done <- nil
		}
	}()

	// by default this blocks flow until a signal is received.
	// a signal is received when (1.) we receive an interrupt signal of some kind, or (2.) the server is closed
	// we're done with the channel once we receive a signal, so the <- into nothing will discard the signal
	<-done

	// we've received a signal, so we need to shut down the server
	// TODO: add docs for this... why ctx all the way down here?
	log.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// right before we exit the main function, cancel the context()
	defer func() { cancel() }()

	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not shutdown server", err)
	}
}

// initializes state and returns a new model and bubbletea options
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	// get the width/height of the viewport for a rough initial size estimate
	physWidth, physHeight, _ := term.GetSize(int(os.Stdout.Fd()))

	computedWidth := min(physWidth, maxWidth)

	vp := viewport.New(physWidth, physHeight-headerHeight-footerHeight)
	vp.HighPerformanceRendering = useHighPerformanceRenderer

	m := model{
		viewport: vp,
		fitWidth: computedWidth,
		loaded:   false,
	}

	// FIXME: dont use tea.WithMouseCellMotion() because it seems to break the viewport when scrolling fast
	return m, []tea.ProgramOption{tea.WithMouseCellMotion(), tea.WithAltScreen()}
}

// this function runs in two cases:
//  1. we've navigated to a new file (needsNewFile = true)
//  2. the best fit width has changed (needsNewFile = false)
//
// and handles updating viewport/glamour/list logic
func RerenderContent(m model, needsNewFile bool, needsNewTerm bool) (model, tea.Cmd) {
	// we've navigated to the posts "page", which *does not* use the viewport for rendering
	// in this case, we set the viewport to 0x0. This is a hacky way to "clear" the viewport in high performance mode

	// we've navigated to a new page (not /posts), so we need to update the viewport and glamour renderer.

	// first, sanity check the viewport height (may be 0x0 if we're in high performance mode and have just
	// navigated away from /posts)
	m.viewport.Height = m.cmdHeight - headerHeight - footerHeight

	if needsNewTerm {
		// set up a new renderer
		renderer, err := glamour.NewTermRenderer(
			glamour.WithEnvironmentConfig(),
			glamour.WithWordWrap(m.fitWidth),
		)

		if err != nil {
			fmt.Println(err.Error())
			return m, tea.Quit
		}

		glamourRenderer = renderer

		if m.fitWidth > 76 {
			renderer, err = glamour.NewTermRenderer(
				glamour.WithEnvironmentConfig(),
				glamour.WithWordWrap(m.fitWidth/2-4),
			)

			if err != nil {
				fmt.Println(err.Error())
				return m, tea.Quit
			}
		}

		projRender1, _ = renderer.Render(prog1)
		projRender2, _ = renderer.Render(prog2)
		projRender3, _ = renderer.Render(prog3)
		projRender4, _ = renderer.Render(prog4)

		projRender1 = strings.TrimSuffix(strings.TrimPrefix(projRender1, "\n"), "\n")
		projRender2 = strings.TrimSuffix(strings.TrimPrefix(projRender2, "\n"), "\n")
		projRender3 = strings.TrimSuffix(strings.TrimPrefix(projRender3, "\n"), "\n")
		projRender4 = strings.TrimSuffix(strings.TrimPrefix(projRender4, "\n"), "\n")

		if m.fitWidth < 80 {
			// remove last newlines
			projRender4 = strings.TrimSuffix(projRender4, "\n")

			flex = lipgloss.JoinVertical(lipgloss.Left, projRender1, projRender2, projRender3, projRender4)
		} else {
			// remove last newlines
			projRender3 = strings.TrimSuffix(projRender2, "\n")
			projRender4 = strings.TrimSuffix(projRender4, "\n")

			rowAHeight := max(strings.Count(projRender1, "\n"), strings.Count(projRender2, "\n"))
			gapA := lipgloss.NewStyle().Width(4).Height(rowAHeight)

			rowBHeight := max(strings.Count(projRender3, "\n"), strings.Count(projRender4, "\n"))
			gapB := lipgloss.NewStyle().Width(4).Height(rowBHeight)

			RowA := lipgloss.JoinHorizontal(lipgloss.Left, projRender1, gapA.Render(), projRender2)
			RowB := lipgloss.JoinHorizontal(lipgloss.Left, projRender3, gapB.Render(), projRender4)

			flex = lipgloss.JoinVertical(lipgloss.Left, RowA, RowB)
		}
	}

	topRender, _ := glamourRenderer.Render(top)
	botRender, _ := glamourRenderer.Render(bot)

	topRender = strings.TrimPrefix(topRender, "\n")
	botRender = strings.TrimSuffix(botRender, "\n")

	contactA := lipgloss.Place(8, 1, lipgloss.Left, lipgloss.Center, ApplySubtle("email")) + ApplyMuted("•") + " " + ApplyNormal("max@mmyron.com") + "\n"
	contactB := lipgloss.Place(8, 1, lipgloss.Left, lipgloss.Center, ApplySubtle("twitter")) + ApplyMuted("•") + " " + ApplyNormal("@mmorenthal") + "\n"
	contactC := lipgloss.Place(8, 1, lipgloss.Left, lipgloss.Center, ApplySubtle("github")) + ApplyMuted("•") + " " + ApplyNormal("maxmmyron") + "\n"

	cmb := lipgloss.JoinVertical(lipgloss.Top, topRender, flex, botRender, contactA, contactB, contactC)

	// in high perf mode, View() doesn't seem to render content in *quite* the same way. here, we do some prelim.
	// rendering by placing the rendered post in a container, and setting the viewport's content to that container
	// (otherwise, we have no way of centering the content)
	if useHighPerformanceRenderer {
		container := lipgloss.Place(m.cmdWidth, m.viewport.Height, lipgloss.Center, lipgloss.Top, cmb)
		m.viewport.SetContent(container)
		return m, viewport.Sync(m.viewport)
	}

	m.viewport.SetContent(cmb)
	return m, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

var lastFitWidth = 0

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		lastFitWidth = m.fitWidth
		m.fitWidth = min(msg.Width, maxWidth)
		if m.fitWidth < 80 {
			m.fitWidth = m.fitWidth - 4
		}

		// update the viewport height
		m.viewport.Height = msg.Height - headerHeight - footerHeight

		m.cmdWidth = msg.Width
		m.cmdHeight = msg.Height

		// if we haven't loaded the viewport yet, then load root post
		if !m.loaded {
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.loaded = true
			m, cmd = RerenderContent(m, true, true)
			cmds = append(cmds, cmd)
		}

		// request rerender if best fit width has changed
		if lastFitWidth != m.fitWidth {
			m, cmd = RerenderContent(m, false, true)
			cmds = append(cmds, cmd)
		} else {
			m, cmd = RerenderContent(m, false, false)
			cmds = append(cmds, cmd)
		}

		// resync on resize if using high performance renderer
		if useHighPerformanceRenderer {
			// we have to manually offset the viewport so the header renders correctly
			m.viewport.YPosition = headerHeight + 1
			cmds = append(cmds, viewport.Sync(m.viewport))
		}

		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		default:
			m.viewport, cmd = m.viewport.Update(msg)
		}
	}

	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

// renders our model
func (m model) View() string {
	header := HeaderView(m)
	footer := FooterView(m)

	// default inner content to a bunch of newlines, so we know where the footer goes (remember, the viewport bypasses
	// this View() fn because we're in high perf. render mode)
	var inner = strings.Repeat("\n", max(0, m.viewport.Height-1))

	combinedVp := lipgloss.JoinVertical(lipgloss.Top, header, inner, footer)

	return lipgloss.NewStyle().Width(m.cmdWidth).Height(m.cmdHeight).Align(lipgloss.Center, lipgloss.Top).Render(combinedVp)
}

func HeaderView(m model) string {
	const (
		altLinkWidth = 0
		linkPadding  = 2
	)

	content := ApplyNormal("mmyron.com/")

	// calculate widths for main content
	mainWidth := m.fitWidth - altLinkWidth
	mainContentWidth := mainWidth - 2 - 2*linkPadding // 2 for the border, 4 for the padding

	// truncate if necessary
	if len(content) > mainContentWidth {
		content = content[:mainContentWidth-4] + "..."
	}

	var (
		pathStyle = lipgloss.NewStyle().Width(mainContentWidth).Padding(0, linkPadding).Render
	)

	t := table.New().BorderColumn(true).Width(m.fitWidth).Border(lipgloss.NormalBorder()).BorderStyle(lipgloss.NewStyle().Foreground(tableBorderColor))
	t.Row(pathStyle(content))

	return lipgloss.NewStyle().Width(m.cmdWidth).Height(headerHeight).Align(lipgloss.Center, lipgloss.Top).SetString(t.Render()).Render()
}

func FooterView(m model) string {
	// ▲/▼ scroll  •  q quit
	scrollHelp := fmt.Sprintf("%s %s", ApplyNormal("↑ /↓"), ApplySubtle("scroll"))
	quitHelp := fmt.Sprintf("%s %s", ApplyNormal("q"), ApplySubtle("quit"))
	help := fmt.Sprintf("%s  %s  %s", scrollHelp, ApplyMuted("•"), quitHelp)

	helpSection := lipgloss.Place(m.fitWidth, footerHeight-1, lipgloss.Center, lipgloss.Center, help)
	borderContainer := lipgloss.NewStyle().Width(m.fitWidth).Height(footerHeight-1).Align(lipgloss.Center, lipgloss.Bottom).Border(lipgloss.NormalBorder(), true, false, false).BorderForeground(tableBorderColor).SetString(helpSection).Render()

	return lipgloss.Place(m.cmdWidth, footerHeight, lipgloss.Center, lipgloss.Bottom, borderContainer)
}
