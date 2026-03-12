package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// PollFunc is called on each poll interval. Return true if 2FA was approved.
type PollFunc func() (bool, error)

// SpinnerResult is sent when the spinner finishes.
type SpinnerResult struct {
	Approved bool
	Err      error
}

type tickMsg struct{}
type pollResultMsg struct {
	approved bool
	err      error
}

// SpinnerModel is a BubbleTea model for a 2FA wait spinner with countdown.
type SpinnerModel struct {
	spinner  spinner.Model
	total    time.Duration
	interval time.Duration
	elapsed  time.Duration
	poll     PollFunc
	result   *SpinnerResult
	quitting bool
}

// NewSpinner creates a 2FA spinner model.
func NewSpinner(total, interval time.Duration, poll PollFunc) SpinnerModel {
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	s.Style = Title
	return SpinnerModel{
		spinner:  s,
		total:    total,
		interval: interval,
		poll:     poll,
	}
}

// Init starts the spinner animation and the first poll tick.
func (m SpinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.scheduleTick())
}

func (m SpinnerModel) scheduleTick() tea.Cmd {
	return tea.Tick(m.interval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m SpinnerModel) doPoll() tea.Cmd {
	poll := m.poll
	return func() tea.Msg {
		ok, err := poll()
		return pollResultMsg{approved: ok, err: err}
	}
}

// Update handles messages.
func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quitting = true
			m.result = &SpinnerResult{Err: fmt.Errorf("cancelled")}
			return m, tea.Quit
		}

	case tickMsg:
		m.elapsed += m.interval
		if m.elapsed >= m.total {
			m.result = &SpinnerResult{Approved: false}
			m.quitting = true
			return m, tea.Quit
		}
		return m, tea.Batch(m.doPoll(), m.scheduleTick())

	case pollResultMsg:
		if msg.err != nil {
			m.result = &SpinnerResult{Err: msg.err}
			m.quitting = true
			return m, tea.Quit
		}
		if msg.approved {
			m.result = &SpinnerResult{Approved: true}
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the spinner with countdown.
func (m SpinnerModel) View() string {
	if m.quitting && m.result != nil {
		if m.result.Err != nil {
			return ErrorBox.Render(m.result.Err.Error()) + "\n"
		}
		if m.result.Approved {
			return SuccessMessage.Render("2FA approved!") + "\n"
		}
		return ErrorBox.Render("2FA timed out.") + "\n"
	}

	remaining := m.total - m.elapsed
	if remaining < 0 {
		remaining = 0
	}
	secs := int(remaining.Seconds())

	return fmt.Sprintf(
		"%s Waiting for 2FA approval on your phone... (%ds remaining)\n\n%s",
		m.spinner.View(),
		secs,
		HelpText.Render("Press q to cancel"),
	)
}

// Result returns the spinner result after the program exits. Nil if still running.
func (m SpinnerModel) Result() *SpinnerResult {
	return m.result
}
