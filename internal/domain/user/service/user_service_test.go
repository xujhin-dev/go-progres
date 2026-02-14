package service

import (
	"testing"
	"time"
	"user_crud_jwt/internal/domain/user/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
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

func (m *MockUserRepository) GetByMobile(mobile string) (*model.User, error) {
	args := m.Called(mobile)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetList(offset, limit int) ([]model.User, int64, error) {
	args := m.Called(offset, limit)
	return args.Get(0).([]model.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) Update(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateMemberStatus(userID string, expireAt time.Time) error {
	args := m.Called(userID, expireAt)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

// MockOTPService is a mock of OTPService
type MockOTPService struct {
	mock.Mock
}

func (m *MockOTPService) Send(mobile string) (string, error) {
	args := m.Called(mobile)
	return args.String(0), args.Error(1)
}

func (m *MockOTPService) Verify(mobile, code string) bool {
	args := m.Called(mobile, code)
	return args.Bool(0)
}

func createTestUser(id, mobile string) *model.User {
	return &model.User{
		ID:       id,
		Mobile:   mobile,
		Nickname: "TestUser",
		Role:     model.RoleUser,
		Status:   model.StatusNormal,
	}
}

func TestLoginOrRegister(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockOTP := new(MockOTPService)
	service := NewUserService(mockRepo, mockOTP)

	t.Run("New user registration success", func(t *testing.T) {
		mobile := "13800138000"
		code := "123456"

		mockOTP.On("Verify", mobile, code).Return(true)
		mockRepo.On("GetByMobile", mobile).Return(nil, gorm.ErrRecordNotFound)
		mockRepo.On("Create", mock.AnythingOfType("*model.User")).Return(nil)
		mockRepo.On("Update", mock.AnythingOfType("*model.User")).Return(nil)

		token, err := service.LoginOrRegister(mobile, code)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		mockOTP.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Existing user login success", func(t *testing.T) {
		mobile := "13800138001"
		code := "123456"
		user := createTestUser("existing-user-id", mobile)

		mockOTP.On("Verify", mobile, code).Return(true)
		mockRepo.On("GetByMobile", mobile).Return(user, nil)
		mockRepo.On("Update", mock.AnythingOfType("*model.User")).Return(nil)

		token, err := service.LoginOrRegister(mobile, code)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		mockOTP.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Invalid verification code", func(t *testing.T) {
		mobile := "13800138002"
		code := "wrongcode"

		mockOTP.On("Verify", mobile, code).Return(false)

		token, err := service.LoginOrRegister(mobile, code)

		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "invalid verification code")
		mockOTP.AssertExpectations(t)
	})
}

func TestSendOTP(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockOTP := new(MockOTPService)
	service := NewUserService(mockRepo, mockOTP)

	t.Run("Send OTP success", func(t *testing.T) {
		mobile := "13800138000"
		code := "123456"

		mockOTP.On("Send", mobile).Return(code, nil)

		err := service.SendOTP(mobile)

		assert.NoError(t, err)
		mockOTP.AssertExpectations(t)
	})
}

func TestGetUsers(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockOTP := new(MockOTPService)
	service := NewUserService(mockRepo, mockOTP)

	t.Run("Get users success", func(t *testing.T) {
		page, limit := 1, 10
		offset := 0
		users := []model.User{
			*createTestUser("user1", "13800138001"),
			*createTestUser("user2", "13800138002"),
		}
		total := int64(2)

		mockRepo.On("GetList", offset, limit).Return(users, total, nil)

		result, totalResult, err := service.GetUsers(page, limit)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, total, totalResult)
		mockRepo.AssertExpectations(t)
	})
}

func TestGetUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockOTP := new(MockOTPService)
	service := NewUserService(mockRepo, mockOTP)

	t.Run("Get user success", func(t *testing.T) {
		userID := "test-user-id"
		user := createTestUser(userID, "13800138000")

		mockRepo.On("GetByID", userID).Return(user, nil)

		result, err := service.GetUser(userID)

		assert.NoError(t, err)
		assert.Equal(t, userID, result.ID)
		mockRepo.AssertExpectations(t)
	})
}

func TestUpdateUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockOTP := new(MockOTPService)
	service := NewUserService(mockRepo, mockOTP)

	t.Run("Update user success", func(t *testing.T) {
		userID := "test-user-id"
		user := createTestUser(userID, "13800138000")
		newNickname := "Updated Nickname"

		mockRepo.On("GetByID", userID).Return(user, nil)
		mockRepo.On("Update", mock.AnythingOfType("*model.User")).Return(nil)

		result, err := service.UpdateUser(userID, newNickname, "")

		assert.NoError(t, err)
		assert.Equal(t, newNickname, result.Nickname)
		mockRepo.AssertExpectations(t)
	})
}

func TestDeleteUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockOTP := new(MockOTPService)
	service := NewUserService(mockRepo, mockOTP)

	t.Run("Delete user success", func(t *testing.T) {
		userID := "test-user-id"
		user := createTestUser(userID, "13800138000")

		mockRepo.On("GetByID", userID).Return(user, nil)
		mockRepo.On("Update", mock.AnythingOfType("*model.User")).Return(nil)

		err := service.DeleteUser(userID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}
