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

	"github.com/charmbracelet/bubbles/list"
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
	loaded      bool           // whether or not the viewport is loaded
	viewport    viewport.Model // mostly holds glamour output
	currentPath string         // current path we're rendering
	content     string         // currently loaded post
	fitWidth    int            // best fit content width max 80ch
	posts       list.Model
	// terminal dims:
	cmdWidth  int
	cmdHeight int
}

type post struct {
	title, path, desc string
	md                string
}

func (p post) Title() string       { return p.title }
func (p post) Path() string        { return p.path }
func (p post) Description() string { return p.desc }
func (p post) mdContent() string   { return p.md }
func (p post) FilterValue() string { return p.title }

const (
	host                       = "localhost"
	port                       = "23234"
	maxWidth                   = 80
	useHighPerformanceRenderer = true

	headerHeight = 4
	footerHeight = 4
)

var (
	glamourRenderer  *glamour.TermRenderer
	posts            []list.Item
	ApplyNormal      = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#000", Dark: "#D7D7D7"}).Render
	ApplySubtle      = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#666", Dark: "#7C7C7C"}).Render
	ApplyMuted       = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#666", Dark: "#3F3F3F"}).Render
	ApplyHighlight   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#000", Dark: "#F93EFD"}).Render
	tableBorderColor = lipgloss.AdaptiveColor{Light: "#cccccc", Dark: "#3F3F3F"}
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
		viewport:    vp,
		fitWidth:    computedWidth,
		loaded:      false,
		currentPath: "/root",
	}

	// grab posts from fs and build out a new list of posts
	files, err := os.ReadDir("fs/posts")

	if err != nil {
		fmt.Println(err.Error())
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		content, err := os.ReadFile("fs/posts/" + file.Name())

		if err != nil {
			fmt.Println(err.Error())
		}

		// split out frontmatter and markdown content, and build out post object to add to list.
		var post post

		md, fm := SplitFrontmatterMarkdown(string(content))

		post.md = md

		if title, ok := fm["title"]; ok {
			post.title = title.(string)
		}

		if desc, ok := fm["subtitle"]; ok {
			post.desc = desc.(string)
		}

		post.path = "fs/posts" + file.Name()

		posts = append(posts, post)
	}

	m.posts = list.New(posts, list.NewDefaultDelegate(), computedWidth, physHeight-headerHeight-footerHeight)
	m.posts.Title = "Recent Posts"
	m.posts.SetShowHelp(false)

	// FIXME: dont use tea.WithMouseCellMotion() because it seems to break the viewport when scrolling fast
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

// this function runs in two cases:
//  1. we've navigated to a new file (needsNewFile = true)
//  2. the best fit width has changed (needsNewFile = false)
//
// and handles updating viewport/glamour/list logic
func Rerender(m model, needsNewFile bool) (model, tea.Cmd) {
	// we've navigated to the posts "page", which *does not* use the viewport for rendering
	// in this case, we set the viewport to 0x0. This is a hacky way to "clear" the viewport in high performance mode
	if m.currentPath == "/posts" {
		m.viewport.SetYOffset(0)
		m.viewport.Width = 0
		m.viewport.Height = 0
		m.viewport.SetContent("")

		if useHighPerformanceRenderer {
			return m, viewport.Sync(m.viewport)
		}
		return m, nil
	}

	// we've navigated to a new page (not /posts), so we need to update the viewport and glamour renderer.

	// first, sanity check the viewport height (may be 0x0 if we're in high performance mode and have just
	// navigated away from /posts)
	m.viewport.Height = m.cmdHeight - headerHeight - footerHeight

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

	// grab a new file if we need to
	if needsNewFile {
		file, err := os.ReadFile("fs" + m.currentPath + ".md")

		if err != nil {
			fmt.Println(err.Error())
			return m, tea.Quit
		}

		m.content = string(file)

		// when navigating to a new page, update offset so we don't load a new page scrolled down
		m.viewport.SetYOffset(0)
	}

	// render glamour content
	render, err := glamourRenderer.Render(m.content)

	if err != nil {
		fmt.Println(err.Error())
		return m, tea.Quit
	}

	// in high perf mode, View() doesn't seem to render content in *quite* the same way. here, we do some prelim.
	// rendering by placing the rendered post in a container, and setting the viewport's content to that container
	// (otherwise, we have no way of centering the content)
	if useHighPerformanceRenderer {
		container := lipgloss.Place(m.cmdWidth, m.viewport.Height, lipgloss.Center, lipgloss.Top, render)
		m.viewport.SetContent(container)
		return m, viewport.Sync(m.viewport)
	}

	m.viewport.SetContent(render)
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

		// if we're not on the posts page, then update the viewport height
		if m.currentPath != "/posts" {
			m.viewport.Height = msg.Height - headerHeight - footerHeight
		}

		m.cmdWidth = msg.Width
		m.cmdHeight = msg.Height

		m.posts.SetWidth(m.fitWidth)
		m.posts.SetHeight(msg.Height - headerHeight - footerHeight)

		// if we haven't loaded the viewport yet, then load root post
		if !m.loaded {
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.loaded = true
			m, cmd = Rerender(m, true)
			cmds = append(cmds, cmd)
		}

		// request rerender if best fit width has changed
		if lastFitWidth != m.fitWidth {
			m, cmd = Rerender(m, false)
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
		case "q":
			return m, tea.Quit
			// FIXME: remove and improve
		case "p":
			// switch to posts page iff we're @ root
			if m.currentPath == "/root" {
				m.currentPath = "/posts"
				m, cmd = Rerender(m, false)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		case "b":
			// switch to root page, if we can
			if m.currentPath != "/root" {
				m.currentPath = "/root"
				m, cmd = Rerender(m, true)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		case "enter":
			// if on post, load selected post
			if m.currentPath == "/posts" {
				newPath := m.posts.SelectedItem().(post).Path()
				m.currentPath = newPath
				m, cmd = Rerender(m, true)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		default:
			if m.currentPath != "/posts" {
				m.viewport, cmd = m.viewport.Update(msg)
			} else {
				m.posts, cmd = m.posts.Update(msg)
			}
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

	// if we're on the posts page, render that as our "inner" content
	if m.currentPath == "/posts" {
		listContainer := lipgloss.NewStyle().Width(m.fitWidth).Height(m.posts.Height()).Align(lipgloss.Left, lipgloss.Top).SetString(m.posts.View()).Render()
		inner = lipgloss.Place(m.cmdWidth, m.posts.Height(), lipgloss.Center, lipgloss.Top, listContainer)
	}

	combinedVp := lipgloss.JoinVertical(lipgloss.Top, header, inner, footer)

	return lipgloss.NewStyle().Width(m.cmdWidth).Height(m.cmdHeight).Align(lipgloss.Center, lipgloss.Top).Render(combinedVp)
}

func HeaderView(m model) string {
	const (
		altLinkWidth = 11
		linkPadding  = 2
	)

	// build out text
	content := m.currentPath
	sideContent := fmt.Sprintf("%s %s", ApplyHighlight("p"), ApplySubtle("posts"))
	if content != "/root" {
		sideContent = fmt.Sprintf("%s %s", ApplyHighlight("b"), ApplySubtle("back"))
	} else {
		content = "/"
	}
	content = ApplyNormal("mmyron.com" + content)

	// calculate widths for main content
	mainWidth := m.fitWidth - altLinkWidth
	mainContentWidth := mainWidth - 2 - 2*linkPadding // 2 for the border, 4 for the padding

	// truncate if necessary
	if len(content) > mainContentWidth {
		content = content[:mainContentWidth-4] + "..."
	}

	var (
		pathStyle = lipgloss.NewStyle().Width(mainContentWidth).Padding(0, linkPadding).Render
		altStyle  = lipgloss.NewStyle().Width(altLinkWidth).Padding(0, linkPadding).Render
	)

	t := table.New().BorderColumn(true).Width(m.fitWidth).Border(lipgloss.NormalBorder()).BorderStyle(lipgloss.NewStyle().Foreground(tableBorderColor))
	t.Row(pathStyle(content), altStyle(sideContent))

	return lipgloss.NewStyle().Width(m.cmdWidth).Height(headerHeight).Align(lipgloss.Center, lipgloss.Top).SetString(t.Render()).Render()
}

func FooterView(m model) string {
	// ▲/▼ scroll  •  q quit
	scrollHelp := fmt.Sprintf("%s %s", ApplyHighlight("▲/▼"), ApplySubtle("scroll"))
	quitHelp := fmt.Sprintf("%s %s", ApplyHighlight("q"), ApplySubtle("quit"))
	help := fmt.Sprintf("%s  %s  %s", scrollHelp, ApplyMuted("•"), quitHelp)

	helpSection := lipgloss.Place(m.fitWidth, footerHeight-1, lipgloss.Center, lipgloss.Center, help)
	borderContainer := lipgloss.NewStyle().Width(m.fitWidth).Height(footerHeight-1).Align(lipgloss.Center, lipgloss.Bottom).Border(lipgloss.NormalBorder(), true, false, false).BorderForeground(tableBorderColor).SetString(helpSection).Render()

	return lipgloss.Place(m.cmdWidth, footerHeight, lipgloss.Center, lipgloss.Bottom, borderContainer)
}
