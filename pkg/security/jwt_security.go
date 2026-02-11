package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"
	"user_crud_jwt/pkg/cache"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSecurity JWT 安全管理器
type JWTSecurity struct {
	secretKey       []byte
	issuer          string
	cache           cache.CacheService
	tokenBlacklist  map[string]bool
	refreshTokens   map[string]*RefreshTokenInfo
	mu              sync.RWMutex
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// RefreshTokenInfo 刷新令牌信息
type RefreshTokenInfo struct {
	UserID    string
	TokenID   string
	ExpiresAt time.Time
	Revoked   bool
	Used      bool
}

// Claims JWT 声明
type Claims struct {
	UserID      string   `json:"user_id"`
	TokenID     string   `json:"token_id"`
	Type        string   `json:"type"` // access or refresh
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// GetJWTID 获取 JWT ID
func (c *Claims) GetJWTID() string {
	return c.TokenID
}

// NewJWTSecurity 创建 JWT 安全管理器
func NewJWTSecurity(secretKey, issuer string, cache cache.CacheService) *JWTSecurity {
	return &JWTSecurity{
		secretKey:       []byte(secretKey),
		issuer:          issuer,
		cache:           cache,
		tokenBlacklist:  make(map[string]bool),
		refreshTokens:   make(map[string]*RefreshTokenInfo),
		accessTokenTTL:  time.Hour * 24,
		refreshTokenTTL: time.Hour * 24 * 7, // 7 days
	}
}

// GenerateTokenPair 生成令牌对
func (js *JWTSecurity) GenerateTokenPair(userID, role string, permissions []string) (accessToken, refreshToken string, err error) {
	// 生成访问令牌
	accessToken, err = js.generateAccessToken(userID, role, permissions)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// 生成刷新令牌
	refreshToken, err = js.generateRefreshToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// generateAccessToken 生成访问令牌
func (js *JWTSecurity) generateAccessToken(userID, role string, permissions []string) (string, error) {
	tokenID, err := generateTokenID()
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := &Claims{
		UserID:      userID,
		TokenID:     tokenID,
		Type:        "access",
		Role:        role,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    js.issuer,
			Subject:   userID,
			Audience:  []string{"api"},
			ExpiresAt: jwt.NewNumericDate(now.Add(js.accessTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        tokenID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(js.secretKey)
}

// generateRefreshToken 生成刷新令牌
func (js *JWTSecurity) generateRefreshToken(userID string) (string, error) {
	tokenID, err := generateTokenID()
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := &Claims{
		UserID:  userID,
		TokenID: tokenID,
		Type:    "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    js.issuer,
			Subject:   userID,
			Audience:  []string{"refresh"},
			ExpiresAt: jwt.NewNumericDate(now.Add(js.refreshTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        tokenID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(js.secretKey)
	if err != nil {
		return "", err
	}

	// 存储刷新令牌信息
	js.mu.Lock()
	js.refreshTokens[tokenID] = &RefreshTokenInfo{
		UserID:    userID,
		TokenID:   tokenID,
		ExpiresAt: now.Add(js.refreshTokenTTL),
		Revoked:   false,
		Used:      false,
	}
	js.mu.Unlock()

	return tokenString, nil
}

// ValidateToken 验证令牌
func (js *JWTSecurity) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return js.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// 检查令牌是否在黑名单中
	if js.isTokenBlacklisted(claims.GetJWTID()) {
		return nil, fmt.Errorf("token is blacklisted")
	}

	// 检查刷新令牌是否被撤销
	if claims.Type == "refresh" {
		js.mu.RLock()
		refreshInfo, exists := js.refreshTokens[claims.GetJWTID()]
		js.mu.RUnlock()

		if !exists || refreshInfo.Revoked || refreshInfo.Used {
			return nil, fmt.Errorf("refresh token is invalid")
		}
	}

	return claims, nil
}

// RefreshToken 刷新令牌
func (js *JWTSecurity) RefreshToken(refreshTokenString string) (string, string, error) {
	// 验证刷新令牌
	claims, err := js.ValidateToken(refreshTokenString)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.Type != "refresh" {
		return "", "", fmt.Errorf("token is not a refresh token")
	}

	// 标记刷新令牌为已使用
	js.mu.Lock()
	if refreshInfo, exists := js.refreshTokens[claims.GetJWTID()]; exists {
		refreshInfo.Used = true
	}
	js.mu.Unlock()

	// 将旧的刷新令牌加入黑名单
	js.addToBlacklist(claims.GetJWTID(), claims.ExpiresAt.Time)

	// 生成新的令牌对
	newAccessToken, newRefreshToken, err := js.GenerateTokenPair(claims.UserID, claims.Role, claims.Permissions)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate new tokens: %w", err)
	}

	return newAccessToken, newRefreshToken, nil
}

// RevokeToken 撤销令牌
func (js *JWTSecurity) RevokeToken(tokenString string) error {
	claims, err := js.ValidateToken(tokenString)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// 将令牌加入黑名单
	js.addToBlacklist(claims.GetJWTID(), claims.ExpiresAt.Time)

	// 如果是刷新令牌，标记为已撤销
	if claims.Type == "refresh" {
		js.mu.Lock()
		if refreshInfo, exists := js.refreshTokens[claims.GetJWTID()]; exists {
			refreshInfo.Revoked = true
		}
		js.mu.Unlock()
	}

	return nil
}

// RevokeUserTokens 撤销用户的所有令牌
func (js *JWTSecurity) RevokeUserTokens(userID string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	// 撤销所有刷新令牌
	for tokenID, refreshInfo := range js.refreshTokens {
		if refreshInfo.UserID == userID {
			refreshInfo.Revoked = true
			js.addToBlacklist(tokenID, refreshInfo.ExpiresAt)
		}
	}

	return nil
}

// isTokenBlacklisted 检查令牌是否在黑名单中
func (js *JWTSecurity) isTokenBlacklisted(tokenID string) bool {
	js.mu.RLock()
	defer js.mu.RUnlock()

	// 检查内存黑名单
	if _, exists := js.tokenBlacklist[tokenID]; exists {
		return true
	}

	// 检查缓存黑名单
	cacheKey := fmt.Sprintf("token_blacklist:%s", tokenID)
	var blacklisted bool
	if err := js.cache.Get(context.Background(), cacheKey, &blacklisted); err == nil && blacklisted {
		return true
	}

	return false
}

// addToBlacklist 将令牌加入黑名单
func (js *JWTSecurity) addToBlacklist(tokenID string, expiresAt time.Time) {
	js.mu.Lock()
	js.tokenBlacklist[tokenID] = true
	js.mu.Unlock()

	// 添加到缓存
	cacheKey := fmt.Sprintf("token_blacklist:%s", tokenID)
	ttl := time.Until(expiresAt)
	if ttl > 0 {
		js.cache.Set(context.Background(), cacheKey, true, ttl)
	}
}

// CleanupExpiredTokens 清理过期令牌
func (js *JWTSecurity) CleanupExpiredTokens() {
	js.mu.Lock()
	defer js.mu.Unlock()

	now := time.Now()

	// 清理过期的刷新令牌
	for tokenID, refreshInfo := range js.refreshTokens {
		if now.After(refreshInfo.ExpiresAt) {
			delete(js.refreshTokens, tokenID)
			delete(js.tokenBlacklist, tokenID)
		}
	}
}

// GetTokenInfo 获取令牌信息
func (js *JWTSecurity) GetTokenInfo(tokenString string) (*TokenInfo, error) {
	claims, err := js.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	return &TokenInfo{
		UserID:      claims.UserID,
		TokenID:     claims.TokenID,
		Type:        claims.Type,
		Role:        claims.Role,
		Permissions: claims.Permissions,
		ExpiresAt:   claims.ExpiresAt.Time,
		IssuedAt:    claims.IssuedAt.Time,
		Issuer:      claims.Issuer,
	}, nil
}

// TokenInfo 令牌信息
type TokenInfo struct {
	UserID      string    `json:"user_id"`
	TokenID     string    `json:"token_id"`
	Type        string    `json:"type"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions"`
	ExpiresAt   time.Time `json:"expires_at"`
	IssuedAt    time.Time `json:"issued_at"`
	Issuer      string    `json:"issuer"`
}

// generateTokenID 生成令牌 ID
func generateTokenID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// TokenMiddleware 令牌中间件
type TokenMiddleware struct {
	jwtSecurity *JWTSecurity
	skipPaths   []string
}

// NewTokenMiddleware 创建令牌中间件
func NewTokenMiddleware(jwtSecurity *JWTSecurity, skipPaths []string) *TokenMiddleware {
	return &TokenMiddleware{
		jwtSecurity: jwtSecurity,
		skipPaths:   skipPaths,
	}
}

// ExtractToken 从请求中提取令牌
func (tm *TokenMiddleware) ExtractToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is required")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("authorization header format must be Bearer {token}")
	}

	return parts[1], nil
}

// ValidateRequest 验证请求令牌
func (tm *TokenMiddleware) ValidateRequest(authHeader string) (*Claims, error) {
	token, err := tm.ExtractToken(authHeader)
	if err != nil {
		return nil, err
	}

	return tm.jwtSecurity.ValidateToken(token)
}

// ShouldSkipPath 检查是否跳过路径
func (tm *TokenMiddleware) ShouldSkipPath(path string) bool {
	for _, skipPath := range tm.skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

// JWTClaimsValidator JWT 声明验证器
type JWTClaimsValidator struct {
	requiredRoles []string
	requiredPerms []string
	issuer        string
	audience      []string
}

// NewJWTClaimsValidator 创建 JWT 声明验证器
func NewJWTClaimsValidator(issuer string) *JWTClaimsValidator {
	return &JWTClaimsValidator{
		issuer:   issuer,
		audience: []string{"api"},
	}
}

// Validate 验证 JWT 声明
func (jcv *JWTClaimsValidator) Validate(claims *Claims) error {
	// 验证发行者
	if claims.Issuer != jcv.issuer {
		return fmt.Errorf("invalid issuer")
	}

	// 验证受众
	if len(jcv.audience) > 0 {
		validAudience := false
		for _, audience := range jcv.audience {
			for _, claimAudience := range claims.Audience {
				if claimAudience == audience {
					validAudience = true
					break
				}
			}
			if validAudience {
				break
			}
		}
		if !validAudience {
			return fmt.Errorf("invalid audience")
		}
	}

	// 验证角色
	if len(jcv.requiredRoles) > 0 {
		validRole := false
		for _, requiredRole := range jcv.requiredRoles {
			if claims.Role == requiredRole {
				validRole = true
				break
			}
		}
		if !validRole {
			return fmt.Errorf("insufficient role")
		}
	}

	// 验证权限
	if len(jcv.requiredPerms) > 0 {
		for _, requiredPerm := range jcv.requiredPerms {
			hasPermission := false
			for _, userPerm := range claims.Permissions {
				if userPerm == requiredPerm {
					hasPermission = true
					break
				}
			}
			if !hasPermission {
				return fmt.Errorf("missing required permission: %s", requiredPerm)
			}
		}
	}

	return nil
}

// AddRequiredRole 添加必需角色
func (jcv *JWTClaimsValidator) AddRequiredRole(role string) {
	jcv.requiredRoles = append(jcv.requiredRoles, role)
}

// AddRequiredPermission 添加必需权限
func (jcv *JWTClaimsValidator) AddRequiredPermission(perm string) {
	jcv.requiredPerms = append(jcv.requiredPerms, perm)
}
