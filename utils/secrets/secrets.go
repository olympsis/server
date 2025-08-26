package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Manager struct {
	secretsPath string
	cache       map[string]string
	mu          sync.RWMutex
}

func New() *Manager {
	return &Manager{
		secretsPath: "/run/secrets",
		cache:       make(map[string]string),
	}
}

// Get retrieves a secret by name
func (m *Manager) Get(name string) (string, error) {
	m.mu.RLock()
	if cached, exists := m.cache[name]; exists {
		m.mu.RUnlock()
		return cached, nil
	}
	m.mu.RUnlock()

	// Read from filesystem
	secret, err := m.readSecret(name)
	if err != nil {
		return "", err
	}

	// Cache the secret
	m.mu.Lock()
	m.cache[name] = secret
	m.mu.Unlock()

	return secret, nil
}

func (m *Manager) readSecret(name string) (string, error) {
	secretPath := filepath.Join(m.secretsPath, name)

	data, err := os.ReadFile(secretPath)
	if err != nil {
		if os.IsNotExist(err) {
			value, err := m.readEnvironment(name)
			if err != nil {
				return "", fmt.Errorf("secret '%s' not found", name)
			}
			return strings.TrimSpace(string(value)), nil
		}
		return "", fmt.Errorf("failed to read secret '%s': %w", name, err)
	}

	return strings.TrimSpace(string(data)), nil
}

func (m *Manager) readEnvironment(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("environment '%s' not found", name)
	}

	return value, nil
}

// GetRequired retrieves a secret and panics if not found (for critical secrets)
func (m *Manager) GetRequired(name string) string {
	secret, err := m.Get(name)
	if err != nil {
		panic(fmt.Sprintf("Required secret '%s' not available: %v", name, err))
	}
	return secret
}

// Exists checks if a secret exists
func (m *Manager) Exists(name string) bool {
	_, err := m.Get(name)
	return err == nil
}
