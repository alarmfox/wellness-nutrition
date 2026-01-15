package testutil

import (
	"database/sql"
	"sync"

	"github.com/alarmfox/wellness-nutrition/app/models"
)

// MockUserRepository is a mock implementation of UserRepository for testing
type MockUserRepository struct {
	mu    sync.RWMutex
	users map[string]*models.User
	Error error // If set, methods will return this error
}

// NewMockUserRepository creates a new mock user repository
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[string]*models.User),
	}
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *MockUserRepository) GetByID(id string) (*models.User, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.users[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return user, nil
}

func (m *MockUserRepository) GetByVerificationToken(token string) (*models.User, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.users {
		if user.VerificationToken.Valid && user.VerificationToken.String == token {
			return user, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *MockUserRepository) GetAll() ([]*models.User, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var users []*models.User
	for _, user := range m.users {
		if user.Role == models.RoleUser {
			users = append(users, user)
		}
	}
	return users, nil
}

func (m *MockUserRepository) Create(user *models.User) error {
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) Update(user *models.User) error {
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.users[user.ID]; !ok {
		return sql.ErrNoRows
	}

	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) DecrementAccesses(userID string) error {
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[userID]
	if !ok {
		return sql.ErrNoRows
	}

	if user.RemainingAccesses > 0 {
		user.RemainingAccesses--
	}
	return nil
}

func (m *MockUserRepository) IncrementAccesses(userID string) error {
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[userID]
	if !ok {
		return sql.ErrNoRows
	}

	user.RemainingAccesses++
	return nil
}

func (m *MockUserRepository) Delete(ids []string) error {
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, id := range ids {
		delete(m.users, id)
	}
	return nil
}

// Reset clears all users and errors
func (m *MockUserRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users = make(map[string]*models.User)
	m.Error = nil
}

// AddUser adds a user to the mock repository (for test setup)
func (m *MockUserRepository) AddUser(user *models.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = user
}

// GetUserCount returns the number of users in the mock repository
func (m *MockUserRepository) GetUserCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.users)
}
