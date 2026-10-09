package agent

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"sync"
	"time"
)

const sessionCookieName = "cuebook_agent_session"

type Message struct {
	Role    string
	Content string
	Sources []string
}

type chatSession struct {
	mu       sync.Mutex
	messages []Message
	lastUsed time.Time
}

type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]*chatSession
	max      int
	ttl      time.Duration
}

func newSessionStore(max int, ttl time.Duration) *sessionStore {
	return &sessionStore{
		sessions: make(map[string]*chatSession),
		max:      max,
		ttl:      ttl,
	}
}

func (store *sessionStore) get(id string, now time.Time) (string, *chatSession, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	for sessionID, session := range store.sessions {
		if now.Sub(session.lastUsed) > store.ttl {
			delete(store.sessions, sessionID)
		}
	}
	if session, exists := store.sessions[id]; id != "" && exists {
		session.lastUsed = now
		return id, session, false, nil
	}

	id, err := randomSessionID()
	if err != nil {
		return "", nil, false, err
	}
	if len(store.sessions) >= store.max {
		var oldestID string
		var oldest time.Time
		for sessionID, session := range store.sessions {
			if oldestID == "" || session.lastUsed.Before(oldest) {
				oldestID, oldest = sessionID, session.lastUsed
			}
		}
		delete(store.sessions, oldestID)
	}
	session := &chatSession{lastUsed: now}
	store.sessions[id] = session
	return id, session, true, nil
}

func (store *sessionStore) touch(id string, session *chatSession, now time.Time) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.sessions[id] == session {
		session.lastUsed = now
	}
}

func (store *sessionStore) count() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return len(store.sessions)
}

func randomSessionID() (string, error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(random[:]), nil
}

func appendTurn(history []Message, user, assistant Message, maxMessages int) []Message {
	history = append(history, user, assistant)
	if len(history) <= maxMessages {
		return history
	}
	remove := len(history) - maxMessages
	if remove%2 != 0 {
		remove++
	}
	if remove >= len(history) {
		return history[len(history)-2:]
	}
	return append([]Message(nil), history[remove:]...)
}

func transcript(messages []Message) []Message {
	copyOfMessages := make([]Message, len(messages))
	for index, message := range messages {
		copyOfMessages[index] = message
		copyOfMessages[index].Sources = append([]string(nil), message.Sources...)
	}
	return copyOfMessages
}

func sortedSources(sources []string) []string {
	unique := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		unique[source] = struct{}{}
	}
	result := make([]string, 0, len(unique))
	for source := range unique {
		result = append(result, source)
	}
	sort.Strings(result)
	return result
}
