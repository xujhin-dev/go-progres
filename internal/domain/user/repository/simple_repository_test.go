package repository

import (
	"context"
	"testing"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/pkg/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleUserRepository_Create(t *testing.T) {
	// 创建内存数据库连接（仅用于测试）
	db := &database.DB{} // 简单创建，不需要真实连接
	repo := NewSimpleUserRepository(db)

	ctx := context.Background()
	user := model.NewUser("13800138000", "TestUser")

	// 执行测试
	err := repo.Create(ctx, user)
	assert.NoError(t, err)

	// 验证用户已创建
	foundUser, err := repo.GetByMobile(ctx, user.Mobile)
	assert.NoError(t, err)
	assert.NotNil(t, foundUser)
	assert.Equal(t, user.Mobile, foundUser.Mobile)
	assert.Equal(t, user.Nickname, foundUser.Nickname)
}

func TestSimpleUserRepository_GetByID(t *testing.T) {
	db := &database.DB{}
	repo := NewSimpleUserRepository(db)

	ctx := context.Background()
	user := model.NewUser("13800138000", "TestUser")

	// 先创建用户
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// 通过ID获取用户
	foundUser, err := repo.GetByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.NotNil(t, foundUser)
	assert.Equal(t, user.ID, foundUser.ID)
	assert.Equal(t, user.Mobile, foundUser.Mobile)
}

func TestSimpleUserRepository_GetByID_NotFound(t *testing.T) {
	db := &database.DB{}
	repo := NewSimpleUserRepository(db)

	ctx := context.Background()
	userID := "non-existent-id"

	// 查找不存在的用户
	foundUser, err := repo.GetByID(ctx, userID)
	assert.NoError(t, err)
	assert.Nil(t, foundUser)
}

func TestSimpleUserRepository_GetByMobile(t *testing.T) {
	db := &database.DB{}
	repo := NewSimpleUserRepository(db)

	ctx := context.Background()
	user := model.NewUser("13800138000", "TestUser")

	// 先创建用户
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// 通过手机号获取用户
	foundUser, err := repo.GetByMobile(ctx, user.Mobile)
	assert.NoError(t, err)
	assert.NotNil(t, foundUser)
	assert.Equal(t, user.ID, foundUser.ID)
	assert.Equal(t, user.Mobile, foundUser.Mobile)
}

func TestSimpleUserRepository_Update(t *testing.T) {
	db := &database.DB{}
	repo := NewSimpleUserRepository(db)

	ctx := context.Background()
	user := model.NewUser("13800138000", "TestUser")

	// 先创建用户
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// 更新用户
	user.Nickname = "UpdatedUser"
	err = repo.Update(ctx, user)
	assert.NoError(t, err)

	// 验证更新
	foundUser, err := repo.GetByMobile(ctx, user.Mobile)
	assert.NoError(t, err)
	assert.Equal(t, "UpdatedUser", foundUser.Nickname)
}

func TestSimpleUserRepository_GetList(t *testing.T) {
	db := &database.DB{}
	repo := NewSimpleUserRepository(db)

	ctx := context.Background()

	// 创建多个用户
	user1 := model.NewUser("13800138001", "User1")
	user2 := model.NewUser("13800138002", "User2")

	err := repo.Create(ctx, user1)
	require.NoError(t, err)
	err = repo.Create(ctx, user2)
	require.NoError(t, err)

	// 获取用户列表
	users, err := repo.GetList(ctx, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestSimpleUserRepository_Count(t *testing.T) {
	db := &database.DB{}
	repo := NewSimpleUserRepository(db)

	ctx := context.Background()

	// 创建用户
	user := model.NewUser("13800138000", "TestUser")
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// 统计用户数量
	count, err := repo.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestSimpleUserRepository_UpdateMemberStatus(t *testing.T) {
	db := &database.DB{}
	repo := NewSimpleUserRepository(db)

	ctx := context.Background()
	user := model.NewUser("13800138000", "TestUser")

	// 先创建用户
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// 更新会员状态
	err = repo.UpdateMemberStatus(ctx, user.ID, model.RoleAdmin)
	assert.NoError(t, err)

	// 验证更新
	foundUser, err := repo.GetByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.Equal(t, model.RoleAdmin, foundUser.Role)
}

func TestSimpleUserRepository_Delete(t *testing.T) {
	db := &database.DB{}
	repo := NewSimpleUserRepository(db)

	ctx := context.Background()
	user := model.NewUser("13800138000", "TestUser")

	// 先创建用户
	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// 删除用户
	err = repo.Delete(ctx, user.ID)
	assert.NoError(t, err)

	// 验证删除
	foundUser, err := repo.GetByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.Nil(t, foundUser)
}
