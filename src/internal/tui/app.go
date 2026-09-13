package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"

	"github.com/nerdexecutive/ne-image-sorter/internal/domain"
	"github.com/nerdexecutive/ne-image-sorter/internal/repository"
	"github.com/nerdexecutive/ne-image-sorter/internal/sorter"
)

// saveConfigMsg asks the root model to persist the configuration a screen
// changed. Screens never touch the repository themselves, so persistence
// stays in one place.
type saveConfigMsg struct{ cfg domain.Config }

// saveConfig returns a command that persists cfg.
func saveConfig(cfg domain.Config) tea.Cmd {
	return func() tea.Msg { return saveConfigMsg{cfg: cfg} }
}

// App is the root Bubbletea model. It owns the configuration and routes
// between screens.
type App struct {
	cfg     domain.Config
	configs repository.Config
	svc     *sorter.Service
	log     zerolog.Logger

	active  Screen
	menu    menuModel
	folders foldersModel
	rules   rulesModel
	sort    sortModel

	width  int
	height int
}

// NewApp builds the root model from a loaded configuration.
func NewApp(cfg domain.Config, configs repository.Config, svc *sorter.Service, log zerolog.Logger) App {
	return App{
		cfg:     cfg,
		configs: configs,
		svc:     svc,
		log:     log,
		active:  ScreenMenu,
		menu:    newMenuModel(cfg),
	}
}

// Init implements tea.Model.
func (a App) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// ctrl+c quits from anywhere, on every screen, always.
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.menu.width = msg.Width
		a.folders.width = msg.Width
		a.rules.width = msg.Width
		a.sort.width, a.sort.height = msg.Width, msg.Height
		return a, nil

	case saveConfigMsg:
		a.cfg = msg.cfg
		a.menu.cfg = msg.cfg
		if err := a.configs.Save(msg.cfg); err != nil {
			a.log.Error().Err(err).Msg("save config failed")
			return a, notify("Could not save the configuration. See the log.", true)
		}
		a.log.Info().Msg("configuration saved")
		return a, nil

	case statusMsg:
		a.menu.status, a.menu.isErr = msg.text, msg.isErr
		return a, nil

	case switchMsg:
		return a.open(msg.screen)
	}

	return a.route(msg)
}

// open switches to a screen, building it fresh from the current config so it
// never shows stale state.
func (a App) open(s Screen) (tea.Model, tea.Cmd) {
	a.active = s
	switch s {
	case ScreenFolders:
		a.folders = newFoldersModel(a.cfg)
		a.folders.width = a.width
	case ScreenRules:
		a.rules = newRulesModel(a.cfg)
		a.rules.width = a.width
	case ScreenSort:
		a.sort = newSortModel(a.svc, a.cfg)
		a.sort.width, a.sort.height = a.width, a.height
		return a, a.sort.Init()
	}
	return a, nil
}

// route forwards a message to the active screen.
func (a App) route(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch a.active {
	case ScreenMenu:
		a.menu, cmd = a.menu.Update(msg)
	case ScreenFolders:
		a.folders, cmd = a.folders.Update(msg)
	case ScreenRules:
		a.rules, cmd = a.rules.Update(msg)
	case ScreenSort:
		a.sort, cmd = a.sort.Update(msg)
	}
	return a, cmd
}

// View implements tea.Model.
func (a App) View() string {
	switch a.active {
	case ScreenFolders:
		return a.folders.View()
	case ScreenRules:
		return a.rules.View()
	case ScreenSort:
		return a.sort.View()
	default:
		return a.menu.View()
	}
}
