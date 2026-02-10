package service

import (
	"errors"
	"testing"
	"user_crud_jwt/internal/domain/user/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(id string) (*model.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(username string) (*model.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetAll() ([]model.User, error) {
	args := m.Called()
	return args.Get(0).([]model.User), args.Error(1)
}

func (m *MockUserRepository) Update(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func TestRegister(t *testing.T) {
	mockRepo := new(MockUserRepository)
	service := NewUserService(mockRepo)

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Create", mock.AnythingOfType("*model.User")).Return(nil)

		err := service.Register("testuser", "password", "test@example.com")
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Failure", func(t *testing.T) {
		// Create a new mock for the second test to avoid expectation conflicts
		mockRepo2 := new(MockUserRepository)
		service2 := NewUserService(mockRepo2)
		mockRepo2.On("Create", mock.AnythingOfType("*model.User")).Return(errors.New("db error"))

		err := service2.Register("testuser", "password", "test@example.com")
		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		mockRepo2.AssertExpectations(t)
	})
}
