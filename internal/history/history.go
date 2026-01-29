// Package history provides session management for persistent chat history.
package history

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopus/internal/openai"

	"github.com/google/uuid"
)

// Message represents a single chat message in a session.
type Message struct {
	Role    openai.ChatCompletionRequestMessageRole `json:"role"`
	Content string                                  `json:"content"`
}

// Session represents a chat session with its history.
type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  []Message `json:"messages"`
}

// Manager handles session lifecycle and persistence.
type Manager struct {
	sessionsDir string
	current     *Session
}

// DefaultSessionsDir returns the default directory for storing sessions.
func DefaultSessionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".gopus", "sessions"), nil
}

// NewManager creates a new session manager with the specified sessions directory.
// If sessionsDir is empty, it uses the default directory (~/.gopus/sessions/).
func NewManager(sessionsDir string) (*Manager, error) {
	if sessionsDir == "" {
		var err error
		sessionsDir, err = DefaultSessionsDir()
		if err != nil {
			return nil, err
		}
	}

	// Ensure the sessions directory exists
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	return &Manager{
		sessionsDir: sessionsDir,
	}, nil
}

// NewSession creates a new session with a generated ID.
func (m *Manager) NewSession() *Session {
	now := time.Now()
	session := &Session{
		ID:        uuid.New().String(),
		Name:      "", // Will be set from first message
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  []Message{},
	}
	m.current = session
	return session
}

// Current returns the currently active session.
func (m *Manager) Current() *Session {
	return m.current
}

// SetCurrent sets the current session.
func (m *Manager) SetCurrent(session *Session) {
	m.current = session
}

// ListSessions returns all available sessions sorted by last updated (most recent first).
func (m *Manager) ListSessions() ([]*Session, error) {
	entries, err := os.ReadDir(m.sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Session{}, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessions []*Session
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		sessionPath := filepath.Join(m.sessionsDir, entry.Name())
		session, err := loadSession(sessionPath)
		if err != nil {
			// Skip corrupted session files
			continue
		}
		sessions = append(sessions, session)
	}

	// Sort by UpdatedAt descending (most recent first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// LoadSessionByID loads a session by its ID.
func (m *Manager) LoadSessionByID(id string) (*Session, error) {
	sessionPath := filepath.Join(m.sessionsDir, id+".json")
	session, err := loadSession(sessionPath)
	if err != nil {
		return nil, err
	}
	m.current = session
	return session, nil
}

// SaveCurrent saves the current session to disk.
func (m *Manager) SaveCurrent() error {
	if m.current == nil {
		return fmt.Errorf("no current session to save")
	}
	return m.Save(m.current)
}

// Save saves a session to disk.
func (m *Manager) Save(session *Session) error {
	session.UpdatedAt = time.Now()
	sessionPath := filepath.Join(m.sessionsDir, session.ID+".json")
	return saveSession(sessionPath, session)
}

// DeleteSession deletes a session by its ID.
func (m *Manager) DeleteSession(id string) error {
	sessionPath := filepath.Join(m.sessionsDir, id+".json")
	if err := os.Remove(sessionPath); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Clear current if it was the deleted session
	if m.current != nil && m.current.ID == id {
		m.current = nil
	}

	return nil
}

// AddMessage adds a message to the current session and saves it.
func (m *Manager) AddMessage(role openai.ChatCompletionRequestMessageRole, content string) error {
	if m.current == nil {
		return fmt.Errorf("no current session")
	}

	m.current.Messages = append(m.current.Messages, Message{
		Role:    role,
		Content: content,
	})

	// Set session name from first user message if not set
	if m.current.Name == "" && role == openai.RoleUser {
		m.current.Name = generateSessionName(content)
	}

	// Auto-save after each message
	return m.SaveCurrent()
}

// generateSessionName creates a session name from the first user message.
// It truncates to a reasonable length and adds ellipsis if needed.
func generateSessionName(content string) string {
	const maxLength = 50

	// Clean up the content
	name := strings.TrimSpace(content)
	name = strings.ReplaceAll(name, "\n", " ")
	name = strings.ReplaceAll(name, "\r", "")

	// Truncate if too long
	if len(name) > maxLength {
		name = name[:maxLength-3] + "..."
	}

	return name
}

// SessionsDir returns the sessions directory path.
func (m *Manager) SessionsDir() string {
	return m.sessionsDir
}

// ConvertSessionMessages converts session messages to OpenAI chat format.
func ConvertSessionMessages(session *Session) []openai.ChatCompletionRequestMessage {
	if session == nil {
		return nil
	}

	messages := make([]openai.ChatCompletionRequestMessage, 0, len(session.Messages))
	for _, msg := range session.Messages {
		messages = append(messages, openai.ChatCompletionRequestMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	return messages
}
