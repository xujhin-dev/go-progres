package service

import (
	"context"
	"testing"
	"time"
	"user_crud_jwt/internal/domain/user/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetByMobile(ctx context.Context, mobile string) (*model.User, error) {
	args := m.Called(ctx, mobile)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetList(ctx context.Context, limit, offset int) ([]*model.User, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.User), args.Error(1)
}

func (m *MockUserRepository) List(ctx context.Context, limit, offset int) ([]*model.User, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.User), args.Error(1)
}

func (m *MockUserRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepository) UpdateMemberStatus(ctx context.Context, userID string, status int) error {
	args := m.Called(ctx, userID, status)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
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
	ctx := context.Background()

	t.Run("New user registration success", func(t *testing.T) {
		mobile := "13800138000"
		code := "123456"

		mockOTP.On("Verify", mobile, code).Return(true)
		mockRepo.On("GetByMobile", ctx, mobile).Return(nil, nil) // 用户不存在
		mockRepo.On("Create", ctx, mock.AnythingOfType("*model.User")).Return(nil)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*model.User")).Return(nil)

		token, err := service.LoginOrRegister(ctx, mobile, code)

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
		mockRepo.On("GetByMobile", ctx, mobile).Return(user, nil)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*model.User")).Return(nil)

		token, err := service.LoginOrRegister(ctx, mobile, code)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		mockOTP.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Invalid verification code", func(t *testing.T) {
		mobile := "13800138002"
		code := "wrongcode"

		mockOTP.On("Verify", mobile, code).Return(false)

		token, err := service.LoginOrRegister(ctx, mobile, code)

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
	ctx := context.Background()

	t.Run("Send OTP success", func(t *testing.T) {
		mobile := "13800138000"
		code := "123456"

		mockOTP.On("Send", mobile).Return(code, nil)

		err := service.SendOTP(ctx, mobile)

		assert.NoError(t, err)
		mockOTP.AssertExpectations(t)
	})
}

func TestGetUsers(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockOTP := new(MockOTPService)
	service := NewUserService(mockRepo, mockOTP)
	ctx := context.Background()

	t.Run("Get users success", func(t *testing.T) {
		page, limit := 1, 10
		offset := (page - 1) * limit // 0
		users := []*model.User{
			createTestUser("user1", "13800138001"),
			createTestUser("user2", "13800138002"),
		}
		total := int64(2)

		mockRepo.On("GetList", ctx, limit, offset).Return(users, nil) // 修正顺序：limit, offset
		mockRepo.On("Count", ctx).Return(total, nil)

		result, totalResult, err := service.GetUsers(ctx, page, limit)

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
	ctx := context.Background()

	t.Run("Get user success", func(t *testing.T) {
		userID := "test-user-id"
		user := createTestUser(userID, "13800138000")

		mockRepo.On("GetByID", ctx, userID).Return(user, nil)

		result, err := service.GetUser(ctx, userID)

		assert.NoError(t, err)
		assert.Equal(t, userID, result.ID)
		mockRepo.AssertExpectations(t)
	})
}

func TestUpdateUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockOTP := new(MockOTPService)
	service := NewUserService(mockRepo, mockOTP)
	ctx := context.Background()

	t.Run("Update user success", func(t *testing.T) {
		userID := "test-user-id"
		user := createTestUser(userID, "13800138000")
		newNickname := "Updated Nickname"

		mockRepo.On("GetByID", ctx, userID).Return(user, nil)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*model.User")).Return(nil)

		result, err := service.UpdateUser(ctx, userID, newNickname, "")

		assert.NoError(t, err)
		assert.Equal(t, newNickname, result.Nickname)
		mockRepo.AssertExpectations(t)
	})
}

func TestDeleteUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockOTP := new(MockOTPService)
	service := NewUserService(mockRepo, mockOTP)
	ctx := context.Background()

	t.Run("Delete user success", func(t *testing.T) {
		userID := "test-user-id"
		user := createTestUser(userID, "13800138000")

		mockRepo.On("GetByID", ctx, userID).Return(user, nil)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*model.User")).Return(nil)

		err := service.DeleteUser(ctx, userID)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestUpgradeMember(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockOTP := new(MockOTPService)
	service := NewUserService(mockRepo, mockOTP)
	ctx := context.Background()

	t.Run("Upgrade member success", func(t *testing.T) {
		userID := "test-user-id"
		user := createTestUser(userID, "13800138000")

		mockRepo.On("GetByID", ctx, userID).Return(user, nil)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*model.User")).Return(nil)

		err := service.UpgradeMember(ctx, userID, 30*24*time.Hour)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestUserStatusChecks(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockOTP := new(MockOTPService)
	service := NewUserService(mockRepo, mockOTP)
	ctx := context.Background()

	t.Run("Banned user login failed", func(t *testing.T) {
		mobile := "13800138000"
		code := "123456"
		user := createTestUser("banned-user", mobile)
		user.Status = model.StatusBanned
		futureTime := time.Now().Add(1 * time.Hour) // 设置为未来时间
		user.BannedUntil = &futureTime

		mockOTP.On("Verify", mobile, code).Return(true)
		mockRepo.On("GetByMobile", ctx, mobile).Return(user, nil)

		token, err := service.LoginOrRegister(ctx, mobile, code)

		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "account is banned")
		mockOTP.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Deleted user login failed", func(t *testing.T) {
		mobile := "13800138001"
		code := "123456"
		user := createTestUser("deleted-user", mobile)
		user.Status = model.StatusDeleted

		mockOTP.On("Verify", mobile, code).Return(true)
		mockRepo.On("GetByMobile", ctx, mobile).Return(user, nil)

		token, err := service.LoginOrRegister(ctx, mobile, code)

		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "account has been deleted")
		mockOTP.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Expired ban user login success", func(t *testing.T) {
		mobile := "13800138002"
		code := "123456"
		user := createTestUser("expired-ban-user", mobile)
		user.Status = model.StatusBanned
		pastTime := time.Now().Add(-1 * time.Hour)
		user.BannedUntil = &pastTime // 设置为过去时间

		mockOTP.On("Verify", mobile, code).Return(true)
		mockRepo.On("GetByMobile", ctx, mobile).Return(user, nil)
		mockRepo.On("Update", ctx, mock.AnythingOfType("*model.User")).Return(nil)

		token, err := service.LoginOrRegister(ctx, mobile, code)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		mockOTP.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})
}
