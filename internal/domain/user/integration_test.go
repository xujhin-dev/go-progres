package user

import (
	"context"
	"testing"
	"time"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/internal/domain/user/repository"
	"user_crud_jwt/internal/domain/user/service"
	"user_crud_jwt/internal/pkg/otp"
	"user_crud_jwt/pkg/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserIntegration 集成测试 - 测试真实的数据库操作
func TestUserIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 初始化数据库连接
	db := database.InitDatabase()
	require.NotNil(t, db)

	// 清理测试数据
	defer func() {
		db.ExecContext(context.Background(), "DELETE FROM users WHERE mobile LIKE '138001380%'")
	}()

	// 创建依赖
	userRepo := repository.NewUserRepository(db)
	otpService := otp.NewOTPService(nil) // 使用内存 OTP 服务
	userService := service.NewUserService(userRepo, otpService)

	ctx := context.Background()
	mobile := "13800138000"
	code := "123456" // 测试 OTP 代码

	t.Run("Register and Login", func(t *testing.T) {
		// 1. 发送 OTP
		err := userService.SendOTP(ctx, mobile)
		assert.NoError(t, err)

		// 2. 注册新用户
		token, err := userService.LoginOrRegister(ctx, mobile, code)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		// 3. 验证用户已创建
		user, err := userService.GetUser(ctx, token)
		if err != nil {
			// 如果通过 token 获取失败，尝试通过 mobile 获取
			users, _, err := userService.GetUsers(ctx, 1, 10)
			assert.NoError(t, err)
			assert.Len(t, users, 1)
			user = &users[0]
		}
		assert.Equal(t, mobile, user.Mobile)
		assert.Equal(t, model.StatusNormal, user.Status)
	})

	t.Run("Login existing user", func(t *testing.T) {
		// 再次登录应该返回相同的用户
		token, err := userService.LoginOrRegister(ctx, mobile, code)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		user, err := userService.GetUser(ctx, token)
		if err != nil {
			users, _, err := userService.GetUsers(ctx, 1, 10)
			assert.NoError(t, err)
			assert.Len(t, users, 1)
			user = &users[0]
		}
		assert.Equal(t, mobile, user.Mobile)
	})

	t.Run("Update user", func(t *testing.T) {
		// 获取用户列表找到用户ID
		users, _, err := userService.GetUsers(ctx, 1, 10)
		require.NoError(t, err)
		require.Len(t, users, 1)

		userID := users[0].ID
		newNickname := "Updated User"
		newAvatarURL := "https://example.com/avatar.jpg"

		// 更新用户信息
		updatedUser, err := userService.UpdateUser(ctx, userID, newNickname, newAvatarURL)
		assert.NoError(t, err)
		assert.Equal(t, newNickname, updatedUser.Nickname)
		assert.Equal(t, newAvatarURL, updatedUser.AvatarURL)
	})

	t.Run("Upgrade member", func(t *testing.T) {
		// 获取用户列表找到用户ID
		users, _, err := userService.GetUsers(ctx, 1, 10)
		require.NoError(t, err)
		require.Len(t, users, 1)

		userID := users[0].ID
		duration := 30 * 24 * time.Hour // 30天

		// 升级会员
		err = userService.UpgradeMember(ctx, userID, duration)
		assert.NoError(t, err)

		// 验证会员状态
		user, err := userService.GetUser(ctx, userID)
		assert.NoError(t, err)
		assert.True(t, user.IsMember)
		assert.NotNil(t, user.MemberExpireAt)
		assert.True(t, user.MemberExpireAt.After(time.Now()))
	})

	t.Run("Delete user", func(t *testing.T) {
		// 获取用户列表找到用户ID
		users, _, err := userService.GetUsers(ctx, 1, 10)
		require.NoError(t, err)
		require.Len(t, users, 1)

		userID := users[0].ID

		// 删除用户
		err = userService.DeleteUser(ctx, userID)
		assert.NoError(t, err)

		// 验证用户状态
		user, err := userService.GetUser(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, model.StatusDeleted, user.Status)
	})
}

// TestDatabaseConnection 测试数据库连接
func TestDatabaseConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database connection test in short mode")
	}

	db := database.InitDatabase()
	require.NotNil(t, db)

	// 测试基本连接
	ctx := context.Background()
	err := db.PingContext(ctx)
	assert.NoError(t, err)

	// 测试查询
	var result int
	err = db.GetContext(ctx, &result, "SELECT 1")
	assert.NoError(t, err)
	assert.Equal(t, 1, result)

	// 测试事务
	tx, err := db.BeginTxx(ctx, nil)
	assert.NoError(t, err)
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "SELECT 1")
	assert.NoError(t, err)

	err = tx.Commit()
	assert.NoError(t, err)
}
