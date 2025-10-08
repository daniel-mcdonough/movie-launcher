package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

var (
	videoDir    = os.Getenv("VIDEO_DIR")
	videoPlayer = os.Getenv("VIDEO_PLAYER")
	videoExts     = []string{".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v", ".mpg", ".mpeg", ".3gp", ".ogv"}
	selectedStyle = lipgloss.NewStyle().Reverse(true)
)

type model struct {
	allVideos    []string
	videos       []string
	cursor       int
	viewportTop  int
	viewportSize int
	selected     string
	quitting     bool
	searchMode   bool
	searchInput  textinput.Model
}

func isVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, videoExt := range videoExts {
		if ext == videoExt {
			return true
		}
	}
	return false
}

func searchVideos(keywords []string) ([]string, error) {
	var results []string
	lowerKeywords := make([]string, len(keywords))
	for i, k := range keywords {
		lowerKeywords[i] = strings.ToLower(k)
	}

	err := filepath.WalkDir(videoDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Only process video files
		if !isVideoFile(path) {
			return nil
		}

		lowerPath := strings.ToLower(path)
		matched := true
		for _, keyword := range lowerKeywords {
			if !strings.Contains(lowerPath, keyword) {
				matched = false
				break
			}
		}

		if matched {
			results = append(results, path)
		}
		return nil
	})

	return results, err
}

func initialModel(videos []string) model {
	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.CharLimit = 100

	return model{
		allVideos:    videos,
		videos:       videos,
		cursor:       0,
		viewportTop:  0,
		viewportSize: 20,
		searchMode:   false,
		searchInput:  ti,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func filterVideos(videos []string, filter string) []string {
	if filter == "" {
		return videos
	}

	lowerFilter := strings.ToLower(filter)
	var filtered []string
	for _, video := range videos {
		if strings.Contains(strings.ToLower(video), lowerFilter) {
			filtered = append(filtered, video)
		}
	}
	return filtered
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewportSize = msg.Height - 5
		if m.viewportSize < 5 {
			m.viewportSize = 5
		}
	case tea.KeyMsg:
		if m.searchMode {
			switch msg.String() {
			case "enter":
				m.searchMode = false
				m.videos = filterVideos(m.allVideos, m.searchInput.Value())
				m.cursor = 0
				m.viewportTop = 0
				m.searchInput.Blur()
				return m, nil
			case "esc", "ctrl+c":
				m.searchMode = false
				m.searchInput.SetValue("")
				m.searchInput.Blur()
				return m, nil
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}
		} else {
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "/":
				m.searchMode = true
				m.searchInput.Focus()
				return m, textinput.Blink
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
					if m.cursor < m.viewportTop {
						m.viewportTop = m.cursor
					}
				}
			case "down", "j":
				if m.cursor < len(m.videos)-1 {
					m.cursor++
					if m.cursor >= m.viewportTop+m.viewportSize {
						m.viewportTop = m.cursor - m.viewportSize + 1
					}
				}
			case "pgup":
				m.cursor -= m.viewportSize
				if m.cursor < 0 {
					m.cursor = 0
				}
				m.viewportTop = m.cursor
			case "pgdown":
				m.cursor += m.viewportSize
				if m.cursor >= len(m.videos) {
					m.cursor = len(m.videos) - 1
				}
				if m.cursor >= m.viewportTop+m.viewportSize {
					m.viewportTop = m.cursor - m.viewportSize + 1
				}
			case "home", "g":
				m.cursor = 0
				m.viewportTop = 0
			case "end", "G":
				m.cursor = len(m.videos) - 1
				m.viewportTop = m.cursor - m.viewportSize + 1
				if m.viewportTop < 0 {
					m.viewportTop = 0
				}
			case "enter":
				if len(m.videos) > 0 {
					m.selected = m.videos[m.cursor]
					m.quitting = true
					return m, tea.Quit
				}
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	s := "Video Browser - arrows/jk, PgUp/PgDn, g/G (top/bottom), / to filter, Enter to play, q to quit\n"
	s += fmt.Sprintf("Found %d videos (showing %d-%d)\n",
		len(m.videos),
		m.viewportTop+1,
		min(m.viewportTop+m.viewportSize, len(m.videos)))

	if m.searchMode {
		s += "/" + m.searchInput.View() + "\n"
	} else {
		s += "\n"
	}

	viewportEnd := min(m.viewportTop+m.viewportSize, len(m.videos))
	for i := m.viewportTop; i < viewportEnd; i++ {
		video := m.videos[i]
		relPath, _ := filepath.Rel(videoDir, video)
		if m.cursor == i {
			s += selectedStyle.Render(relPath) + "\n"
		} else {
			s += relPath + "\n"
		}
	}

	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	if videoDir == "" {
		fmt.Println("VIDEO_DIR environment variable is required")
		os.Exit(1)
	}
	if videoPlayer == "" {
		videoPlayer = "mpv"
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: movie-launcher <search keywords...>")
		fmt.Println("Example: movie-launcher matrix 1999")
		os.Exit(1)
	}

	keywords := os.Args[1:]
	fmt.Printf("Searching for videos matching: %s\n", strings.Join(keywords, " "))

	videos, err := searchVideos(keywords)
	if err != nil {
		fmt.Printf("Error searching videos: %v\n", err)
		os.Exit(1)
	}

	if len(videos) == 0 {
		fmt.Println("No videos found matching your search.")
		os.Exit(0)
	}

	p := tea.NewProgram(initialModel(videos), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		fmt.Printf("Error running UI: %v\n", err)
		os.Exit(1)
	}

	finalModel := m.(model)
	if finalModel.selected != "" {
		fmt.Printf("Playing: %s\n", finalModel.selected)
		cmd := exec.Command(videoPlayer, finalModel.selected)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error playing video: %v\n", err)
			os.Exit(1)
		}
	}
}
