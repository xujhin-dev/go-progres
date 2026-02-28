package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"
	"user_crud_jwt/internal/domain/user/model"
	"user_crud_jwt/pkg/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Create(t *testing.T) {
	// 创建模拟数据库连接
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	dbWrapper := &database.DB{DB: sqlxDB}
	userRepo := NewSQLCUserRepository(dbWrapper)

	ctx := context.Background()
	user := model.NewUser("13800138000", "TestUser")

	// 设置期望的 SQL 查询
	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(user.ID, user.CreatedAt, user.UpdatedAt, user.Username, user.Password, user.Email, user.Mobile, user.Nickname, user.AvatarURL, user.Role, user.IsMember, user.MemberExpireAt, user.Status, user.BannedUntil, user.Token, user.TokenExpireAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 执行测试
	err = userRepo.Create(ctx, user)
	assert.NoError(t, err)

	// 验证所有期望都被满足
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	dbWrapper := &database.DB{DB: sqlxDB}
	userRepo := NewSQLCUserRepository(dbWrapper)

	ctx := context.Background()
	userID := "test-user-id"
	expectedUser := &model.User{
		ID:       userID,
		Mobile:   "13800138000",
		Nickname: "TestUser",
		Role:     model.RoleUser,
		Status:   model.StatusNormal,
	}

	// 设置期望的 SQL 查询
	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "username", "password", "email", "mobile", "nickname", "avatar_url", "role", "is_member", "member_expire_at", "status", "banned_until", "token", "token_expire_at"}).
		AddRow(expectedUser.ID, time.Now(), time.Now(), nil, expectedUser.Username, expectedUser.Password, expectedUser.Email, expectedUser.Mobile, expectedUser.Nickname, expectedUser.AvatarURL, expectedUser.Role, expectedUser.IsMember, expectedUser.MemberExpireAt, expectedUser.Status, expectedUser.BannedUntil, expectedUser.Token, expectedUser.TokenExpireAt)

	mock.ExpectQuery(`SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile, nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, token, token_expire_at FROM users WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(userID).
		WillReturnRows(rows)

	// 执行测试
	user, err := userRepo.GetByID(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Mobile, user.Mobile)

	// 验证所有期望都被满足
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	dbWrapper := &database.DB{DB: sqlxDB}
	userRepo := NewSQLCUserRepository(dbWrapper)

	ctx := context.Background()
	userID := "non-existent-id"

	// 设置期望的 SQL 查询返回空结果
	mock.ExpectQuery(`SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile, nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, token, token_expire_at FROM users WHERE id = \$1 AND deleted_at IS NULL`).
		WithArgs(userID).
		WillReturnError(sql.ErrNoRows)

	// 执行测试
	user, err := userRepo.GetByID(ctx, userID)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Equal(t, "user not found", err.Error())

	// 验证所有期望都被满足
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByMobile(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	dbWrapper := &database.DB{DB: sqlxDB}
	userRepo := NewSQLCUserRepository(dbWrapper)

	ctx := context.Background()
	mobile := "13800138000"
	expectedUser := &model.User{
		ID:       "test-user-id",
		Mobile:   mobile,
		Nickname: "TestUser",
		Role:     model.RoleUser,
		Status:   model.StatusNormal,
	}

	// 设置期望的 SQL 查询
	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "username", "password", "email", "mobile", "nickname", "avatar_url", "role", "is_member", "member_expire_at", "status", "banned_until", "token", "token_expire_at"}).
		AddRow(expectedUser.ID, time.Now(), time.Now(), nil, expectedUser.Username, expectedUser.Password, expectedUser.Email, expectedUser.Mobile, expectedUser.Nickname, expectedUser.AvatarURL, expectedUser.Role, expectedUser.IsMember, expectedUser.MemberExpireAt, expectedUser.Status, expectedUser.BannedUntil, expectedUser.Token, expectedUser.TokenExpireAt)

	mock.ExpectQuery(`SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile, nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, token, token_expire_at FROM users WHERE mobile = \$1 AND deleted_at IS NULL`).
		WithArgs(mobile).
		WillReturnRows(rows)

	// 执行测试
	user, err := userRepo.GetByMobile(ctx, mobile)
	assert.NoError(t, err)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Mobile, user.Mobile)

	// 验证所有期望都被满足
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	dbWrapper := &database.DB{DB: sqlxDB}
	userRepo := NewSQLCUserRepository(dbWrapper)

	ctx := context.Background()
	user := model.NewUser("13800138000", "TestUser")
	user.Nickname = "UpdatedUser"

	// 设置期望的 SQL 查询
	mock.ExpectExec(`UPDATE users SET updated_at = \$1, username = \$2, password = \$3, email = \$4, mobile = \$5, nickname = \$6, avatar_url = \$7, role = \$8, is_member = \$9, member_expire_at = \$10, status = \$11, banned_until = \$12, token = \$13, token_expire_at = \$14 WHERE id = \$15 AND deleted_at IS NULL`).
		WithArgs(sqlmock.AnyArg(), user.Username, user.Password, user.Email, user.Mobile, user.Nickname, user.AvatarURL, user.Role, user.IsMember, user.MemberExpireAt, user.Status, user.BannedUntil, user.Token, user.TokenExpireAt, user.ID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 执行测试
	err = userRepo.Update(ctx, user)
	assert.NoError(t, err)

	// 验证所有期望都被满足
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetList(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	dbWrapper := &database.DB{DB: sqlxDB}
	userRepo := NewSQLCUserRepository(dbWrapper)

	ctx := context.Background()
	offset := 0
	limit := 10

	// 设置期望的 SQL 查询
	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "username", "password", "email", "mobile", "nickname", "avatar_url", "role", "is_member", "member_expire_at", "status", "banned_until", "token", "token_expire_at"}).
		AddRow("user1", time.Now(), time.Now(), nil, "user1", "", "", "13800138001", "User1", "", model.RoleUser, false, nil, model.StatusNormal, nil, "", nil).
		AddRow("user2", time.Now(), time.Now(), nil, "user2", "", "", "13800138002", "User2", "", model.RoleUser, false, nil, model.StatusNormal, nil, "", nil)

	// 设置总数查询
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE deleted_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// 设置列表查询
	mock.ExpectQuery(`SELECT id, created_at, updated_at, deleted_at, username, password, email, mobile, nickname, avatar_url, role, is_member, member_expire_at, status, banned_until, token, token_expire_at FROM users WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT \$1 OFFSET \$2`).
		WithArgs(limit, offset).
		WillReturnRows(rows)

	// 执行测试
	users, err := userRepo.GetList(ctx, limit, offset)
	assert.NoError(t, err)
	assert.Len(t, users, 2)

	// 验证所有期望都被满足
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_UpdateMemberStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	dbWrapper := &database.DB{DB: sqlxDB}
	userRepo := NewSQLCUserRepository(dbWrapper)

	ctx := context.Background()
	userID := "test-user-id"
	status := 1 // 会员状态

	// 设置期望的 SQL 查询
	mock.ExpectExec(`UPDATE users SET updated_at = \$1, username = \$2, password = \$3, email = \$4, mobile = \$5, nickname = \$6, avatar_url = \$7, role = \$8, is_member = \$9, member_expire_at = \$10, status = \$11, banned_until = \$12, token = \$13, token_expire_at = \$14 WHERE id = \$15 AND deleted_at IS NULL`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), status, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), userID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 执行测试
	err = userRepo.UpdateMemberStatus(ctx, userID, status)
	assert.NoError(t, err)

	// 验证所有期望都被满足
	assert.NoError(t, mock.ExpectationsWereMet())
}
