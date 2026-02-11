package security

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"user_crud_jwt/pkg/cache"

	"github.com/gin-gonic/gin"
)

// Permission 权限定义
type Permission string

const (
	// 用户权限
	PermissionUserRead    Permission = "user:read"
	PermissionUserWrite   Permission = "user:write"
	PermissionUserDelete  Permission = "user:delete"
	
	// 优惠券权限
	PermissionCouponRead   Permission = "coupon:read"
	PermissionCouponWrite  Permission = "coupon:write"
	PermissionCouponDelete Permission = "coupon:delete"
	
	// 动态权限
	PermissionMomentRead   Permission = "moment:read"
	PermissionMomentWrite  Permission = "moment:write"
	PermissionMomentDelete Permission = "moment:delete"
	
	// 支付权限
	PermissionPaymentRead  Permission = "payment:read"
	PermissionPaymentWrite Permission = "payment:write"
	
	// 管理员权限
	PermissionAdminRead    Permission = "admin:read"
	PermissionAdminWrite   Permission = "admin:write"
	PermissionAdminDelete  Permission = "admin:delete"
	PermissionAdminSystem  Permission = "admin:system"
	
	// 系统权限
	PermissionSystemMonitor Permission = "system:monitor"
	PermissionSystemConfig Permission = "system:config"
)

// Role 角色定义
type Role string

const (
	RoleUser      Role = "user"
	RoleModerator Role = "moderator"
	RoleAdmin     Role = "admin"
	RoleSuperAdmin Role = "super_admin"
)

// PermissionChecker 权限检查器接口
type PermissionChecker interface {
	HasPermission(userID string, permission Permission) (bool, error)
	HasRole(userID string, role Role) (bool, error)
	HasAnyPermission(userID string, permissions []Permission) (bool, error)
	HasAllPermissions(userID string, permissions []Permission) (bool, error)
	GetUserPermissions(userID string) ([]Permission, error)
	GetUserRole(userID string) (Role, error)
}

// RBAC 基于角色的访问控制
type RBAC struct {
	cache          cache.CacheService
	rolePermissions map[Role][]Permission
	userRoles      map[string]Role
	userPermissions map[string][]Permission
	mu             sync.RWMutex
}

// NewRBAC 创建 RBAC 实例
func NewRBAC(cache cache.CacheService) *RBAC {
	rbac := &RBAC{
		cache:          cache,
		rolePermissions: make(map[Role][]Permission),
		userRoles:      make(map[string]Role),
		userPermissions: make(map[string][]Permission),
	}

	// 初始化角色权限映射
	rbac.initDefaultRoles()
	
	return rbac
}

// initDefaultRoles 初始化默认角色
func (rbac *RBAC) initDefaultRoles() {
	// 普通用户权限
	rbac.rolePermissions[RoleUser] = []Permission{
		PermissionUserRead,
		PermissionUserWrite,
		PermissionCouponRead,
	PermissionMomentRead,
		PermissionMomentWrite,
	}

	// 管理员权限
	rbac.rolePermissions[RoleModerator] = []Permission{
		PermissionUserRead,
		PermissionUserWrite,
		PermissionUserDelete,
		PermissionCouponRead,
		PermissionCouponWrite,
		PermissionCouponDelete,
		PermissionMomentRead,
		PermissionMomentWrite,
		PermissionMomentDelete,
		PermissionPaymentRead,
	}

	// 管理员权限
	rbac.rolePermissions[RoleAdmin] = []Permission{
		PermissionUserRead,
		PermissionUserWrite,
		PermissionUserDelete,
		PermissionCouponRead,
		PermissionCouponWrite,
		PermissionCouponDelete,
		PermissionMomentRead,
		PermissionWrite,
		PermissionMomentDelete,
		PermissionPaymentRead,
		PermissionPaymentWrite,
		PermissionAdminRead,
		PermissionAdminWrite,
		PermissionAdminDelete,
	}

	// 超级管理员权限
	rbac.rolePermissions[RoleSuperAdmin] = []Permission{
		PermissionUserRead,
		PermissionUserWrite,
		PermissionUserDelete,
		PermissionCouponRead,
		PermissionCouponWrite,
		PermissionCouponDelete,
	PermissionMomentRead,
		PermissionWrite,
		PermissionMomentDelete,
		PermissionPaymentRead,
		PermissionPaymentWrite,
		PermissionAdminRead,
		PermissionAdminWrite,
		PermissionAdminDelete,
		PermissionAdminSystem,
		PermissionSystemMonitor,
		PermissionSystemConfig,
	}
}

// HasPermission 检查用户是否有指定权限
func (rbac *RBAC) HasPermission(userID string, permission Permission) (bool, error) {
	// 首先检查缓存
	cacheKey := fmt.Sprintf("user_permission:%s:%s", userID, permission)
	var hasPermission bool
	if err := rbac.cache.Get(context.Background(), cacheKey, &hasPermission); err == nil {
		return hasPermission, nil
	}

	// 检查用户权限
	rbac.mu.RLock()
	userPerms, exists := rbac.userPermissions[userID]
	rbac.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("user not found: %s", userID)
	}

	for _, perm := range userPerms {
		if perm == permission {
			// 缓存结果
			rbac.cache.Set(context.Background(), cacheKey, true, time.Minute*30)
			return true, nil
		}
	}

	// 缓存结果
	rbac.cache.Set(context.Background(), cacheKey, false, time.Minute*30)
	return false, nil
}

// HasRole 检查用户是否有指定角色
func (rbac *RBAC) HasRole(userID string, role Role) (bool, error) {
	// 首先检查缓存
	cacheKey := fmt.Sprintf("user_role:%s:%s", userID, role)
	var hasRole bool
	if err := rbac.cache.Get(context.Background(), cacheKey, &hasRole); err == nil {
		return hasRole, nil
	}

	// 检查用户角色
	rbac.mu.RLock()
	userRole, exists := rbac.userRoles[userID]
	rbac.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("user not found: %s", userID)
	}

	hasRole = (userRole == role)
	
	// 缓存结果
	rbac.cache.Set(context.Background(), cacheKey, hasRole, time.Minute*30)
	return hasRole, nil
}

// HasAnyPermission 检查用户是否有任意一个权限
func (rbac *RBAC) HasAnyPermission(userID string, permissions []Permission) (bool, error) {
	for _, permission := range permissions {
		if has, err := rbac.HasPermission(userID, permission); err != nil {
			return false, err
		} else if has {
			return true, nil
		}
	}
	return false, nil
}

// HasAllPermissions 检查用户是否拥有所有权限
func (rbac *RBAC) HasAllPermissions(userID string, permissions []Permission) (bool, error) {
	for _, permission := range permissions {
		if has, err := rbac.HasPermission(userID, permission); err != nil {
			return false, err
		} else if !has {
			return false, nil
		}
	}
	return true, nil
}

// GetUserPermissions 获取用户所有权限
func (rbac *RBAC) GetUserPermissions(userID string) ([]Permission, error) {
	// 首先检查缓存
	cacheKey := fmt.Sprintf("user_permissions:%s", userID)
	var permissions []Permission
	if err := rbac.cache.Get(context.Background(), cacheKey, &permissions); err == nil {
		return permissions, nil
	}

	// 获取用户权限
	rbac.mu.RLock()
	userPerms, exists := rbac.userPermissions[userID]
	rbac.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	// 缓存结果
	rbac.cache.Set(context.Background(), cacheKey, userPerms, time.Minute*30)
	return userPerms, nil
}

// GetUserRole 获取用户角色
func (rbac *RBAC) GetUserRole(userID string) (Role, error) {
	// 首先检查缓存
	cacheKey := fmt.Sprintf("user_role:%s", userID)
	var role Role
	if err := rbac.cache.Get(context.Background(), cacheKey, &role); err == nil {
		return role, nil
	}

	// 获取用户角色
	rbac.mu.RLock()
	userRole, exists := rbac.userRoles[userID]
	rbac.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("user not found: %s", userID)
	}

	// 缓存结果
	rbac.cache.Set(context.Background(), cacheKey, userRole, time.Minute*30)
	return userRole, nil
}

// AssignRole 为用户分配角色
func (rbac *RBAC) AssignRole(userID string, role Role) error {
	rbac.mu.Lock()
	defer rbac.Unlock()

	// 更新用户角色
	rbac.userRoles[userID] = role

	// 更新用户权限
	rbac.userPermissions[userID] = rbac.rolePermissions[role]

	// 清除相关缓存
	rbac.clearUserCache(userID)

	return nil
}

// AddPermissionToRole 为角色添加权限
func (rbac *RBAC) AddPermissionToRole(role Role, permission Permission) error {
	rbac.mu.Lock()
	defer rbac.Unlock()

	// 更新角色权限
	permissions := rbac.rolePermissions[role]
	for _, perm := range permissions {
		if perm == permission {
			return fmt.Errorf("permission already exists for role %s: %s", role, permission)
		}
	}
	rbac.rolePermissions[role] = append(permissions, permission)

	// 更新拥有该角色的用户权限
	for userID, userRole := range rbac.userRoles {
		if userRole == role {
			rbac.userPermissions[userID] = rbac.rolePermissions[role]
			rbac.clearUserCache(userID)
		}
	}

	return nil
}

// RemovePermissionFromRole 从角色移除权限
func (rbac *RBAC) RemovePermissionFromRole(role Role, permission Permission) error {
	rbac.mu.Lock()
	defer rbac.Unlock()

	// 更新角色权限
	permissions := rbac.rolePermissions[role]
	newPermissions := make([]Permission, 0, len(permissions))
	found := false

	for _, perm := range permissions {
		if perm != permission {
			newPermissions = append(newPermissions, perm)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("permission not found for role %s: %s", role, permission)
	}

	rbac.rolePermissions[role] = newPermissions

	// 更新拥有该角色的用户权限
	for userID, userRole := range rbac.userRoles {
		if userRole == role {
			rbac.userPermissions[userID] = newPermissions
			rbac.clearUserCache(userID)
		}
	}

	return nil
}

// clearUserCache 清除用户相关缓存
func (rbac *RBAC) clearUserCache(userID string) {
	// 清除角色缓存
	rbac.cache.Delete(context.Background(), fmt.Sprintf("user_role:%s", userID))
	
	// 清除权限缓存
	for _, perm := range rbac.rolePermissions[rbac.userRoles[userID]] {
		rbac.cache.Delete(context.Background(), fmt.Sprintf("user_permission:%s:%s", userID, perm))
	}
	
	// 清除权限列表缓存
	rbac.cache.Delete(context.Background(), fmt.Sprintf("user_permissions:%s", userID))
}

// GetRolePermissions 获取角色权限
func (rbac *RBAC) GetRolePermissions(role Role) []Permission {
	rbac.mu.RLock()
	defer rbac.RUnlock()
	return rbac.rolePermissions[role]
}

// GetUsersByRole 获取拥有指定角色的用户
func (rbac *RBAC) GetUsersByRole(role Role) []string {
	rbac.mu.RLock()
	defer rbac.RUnlock()

	var users []string
	for userID, userRole := range rbac.userRoles {
		if userRole == role {
			users = append(users, userID)
		}
	}

	return users
}

// PermissionMiddleware 权限检查中间件
type PermissionMiddleware struct {
	rbac     *RBAC
	required Permission
}

// NewPermissionMiddleware 创建权限检查中间件
func NewPermissionMiddleware(rbac *RBAC, required Permission) *PermissionMiddleware {
	return &PermissionMiddleware{
		rbac:     rbac,
		required: required,
	}
}

// Middleware 返回中间件
func (pm *PermissionMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			c.Abort()
			return
		}

		// 检查权限
		hasPermission, err := pm.rbac.HasPermission(userID.(string), pm.required)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "permission check failed",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("permission denied: %s", pm.required),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RoleMiddleware 角色检查中间件
type RoleMiddleware struct {
	rbac     *RBAC
	required Role
}

// NewRoleMiddleware 创建角色检查中间件
func NewRoleMiddleware(rbac *RBAC, required Role) *RoleMiddleware {
	return &RoleMiddleware{
		rbac:     rbac,
		required: required,
	}
}

// Middleware 返回中间件
func (rm *RoleMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			c.Abort()
			return
		}

		// 检查角色
		hasRole, err := rm.rbac.HasRole(userID.(string), rm.required)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "role check failed",
			})
			c.Abort()
			return
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("role required: %s", rm.required),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// MultiPermissionMiddleware 多权限检查中间件
type MultiPermissionMiddleware struct {
	rbac      *RBAC
	required []Permission
	requireAll bool // true: 需要所有权限，false: 需要任意权限
}

// NewMultiPermissionMiddleware 创建多权限检查中间件
func NewMultiPermissionMiddleware(rbac *RBAC, required []Permission, requireAll bool) *MultiPermissionMiddleware {
	return &MultiPermissionMiddleware{
		rbac:      rbac,
		required: required,
		requireAll: requireAll,
	}
}

// Middleware 返回中间件
func (mpm *MultiPermissionMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			c.Abort()
			return
		}

		var hasPermission bool
		var err error

		if mpm.requireAll {
			// 需要所有权限
			hasPermission, err = mpm.rbac.HasAllPermissions(userID.(string), mpm.required)
		} else {
			// 需要任意权限
			hasPermission, err = mpm.rbac.HasAnyPermission(userID.(string), mpm.required)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "permission check failed",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "permission denied",
				"required_permissions": mpm.required,
				"require_all":       mpm.requireAll,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OwnershipMiddleware 所有权检查中间件
type OwnershipMiddleware struct {
	rbac *RBAC
}

// NewOwnershipMiddleware 创建所有权检查中间件
func NewOwnershipMiddleware(rbac *RBAC) *OwnershipMiddleware {
	return &OwnershipMiddleware{rbac: rbac}
}

// Middleware 返回中间件
func (om *OwnershipMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			c.Abort()
			return
		}

		// 获取资源ID
		resourceID := c.Param("id")
		if resourceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "resource ID required",
			})
			c.Abort()
			return
		}

		// 检查所有权（这里简化处理，实际应该检查数据库）
		if !om.checkOwnership(userID.(string), resourceID, c.Request.URL.Path) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "access denied: you don't own this resource",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkOwnership 检查所有权
func (om *OwnershipMiddleware) checkOwnership(userID, resourceID, path string) bool {
	// 简化实现：用户只能访问自己的资源
	// 实际项目中应该查询数据库验证所有权
	
	// 如果是管理员，可以访问所有资源
	if role, err := om.rbac.GetUserRole(userID); err == nil {
		if role == RoleAdmin || role == RoleSuperAdmin {
			return true
		}
	}

	// 检查资源路径和用户ID匹配
	return strings.Contains(path, "/users/") && resourceID == userID
}

// ResourceOwner 资源所有者接口
type ResourceOwner interface {
	GetOwnerID() string
}

// CheckResourceOwnership 检查资源所有权
func CheckResourceOwnership(userID string, resource ResourceOwner) bool {
	return resource.GetOwnerID() == userID
}

// PolicyEngine 策略引擎
type PolicyEngine struct {
	policies map[string]Policy
	rbac     *RBAC
}

// Policy 策略接口
type Policy interface {
	Evaluate(ctx context.Context, request PolicyRequest) (PolicyDecision, error)
}

// PolicyRequest 策略请求
type PolicyRequest struct {
	UserID     string
	Resource   string
	Action     string
	Context    map[string]interface{}
}

// PolicyDecision 策略决定
type PolicyDecision int

const (
	DecisionAllow PolicyDecision = iota
	DecisionDeny
	DecisionNotApplicable
)

// NewPolicyEngine 创建策略引擎
func NewPolicyEngine(rbac *RBAC) *PolicyEngine {
	return &PolicyEngine{
		policies: make(map[string]Policy),
		rbac:     rbac,
	}
}

// AddPolicy 添加策略
func (pe *PolicyEngine) AddPolicy(name string, policy Policy) {
	pe.policies[name] = policy
}

// Evaluate 评估策略
func (pe *PolicyEngine) Evaluate(ctx context.Context, request PolicyRequest) (PolicyDecision, error) {
	// 优先检查 RBAC
	if request.Action != "" {
		// 将动作映射为权限
		permission := mapActionToPermission(request.Action)
		if permission != "" {
			hasPermission, err := pe.rbac.HasPermission(request.UserID, permission)
			if err != nil {
				return DecisionDeny, err
			}
			if !hasPermission {
				return DecisionDeny, nil
			}
		}
	}

	// 检查自定义策略
	for _, policy := range pe.policies {
		decision, err := policy.Evaluate(ctx, request)
		if err != nil {
			return DecisionDeny, err
		}
		if decision == DecisionDeny {
			return DecisionDeny, nil
		}
		if decision == DecisionAllow {
			return DecisionAllow, nil
		}
	}

	return DecisionAllow, nil
}

// mapActionToPermission 将动作映射为权限
func mapActionToPermission(action string) Permission {
	actionPermissionMap := map[string]Permission{
		"read":   PermissionUserRead,
		"write":  PermissionUserWrite,
		"delete": PermissionUserDelete,
		"create": PermissionUserWrite,
		"update": PermissionUserWrite,
	}

	if perm, exists := actionPermissionMap[action]; exists {
		return perm
	}

	return ""
}

// TimeBasedPolicy 基于时间的策略
type TimeBasedPolicy struct {
	startTime time.Time
	endTime   time.Time
	days      []time.Weekday
	hours     []int
}

// NewTimeBasedPolicy 创建基于时间的策略
func NewTimeBasedPolicy(startTime, endTime time.Time, days []time.Weekday, hours []int) *TimeBasedPolicy {
	return &TimeBasedPolicy{
		startTime: startTime,
		endTime:   endTime,
		days:      days,
		hours:     hours,
	}
}

// Evaluate 评估策略
func (tbp *TimeBasedPolicy) Evaluate(ctx context.Context, request PolicyRequest) (PolicyDecision, error) {
	now := time.Now()

	// 检查时间范围
	if now.Before(tbp.startTime) || now.After(tbp.endTime) {
		return DecisionNotApplicable, nil
	}

	// 检查星期
	if len(tbp.days) > 0 {
		weekday := now.Weekday()
		allowed := false
		for _, day := range tbp.days {
			if weekday == day {
				allowed = true
				break
			}
		}
		if !allowed {
			return DecisionDeny, nil
		}
	}

	// 检查小时
	if len(tbp.hours) > 0 {
		hour := now.Hour()
		allowed := false
		for _, h := range tbp.hours {
			if hour == h {
				allowed = true
				break
			}
	}
		if !allowed {
			return DecisionDeny, nil
		}
	}

	return DecisionAllow, nil
}

// LocationPolicy 基于位置的策略
type LocationPolicy struct {
	allowedCountries []string
	allowedIPs       []string
	blockedIPs       []string
}

// NewLocationPolicy 创建基于位置的策略
func NewLocationPolicy(allowedCountries, allowedIPs, blockedIPs []string) *LocationPolicy {
	return &LocationPolicy{
		allowedCountries: allowedCountries,
		allowedIPs:       allowedIPs,
		blockedIPs:       blockedIPs,
	}
}

// Evaluate 评估策略
func (lp *LocationPolicy) Evaluate(ctx context.Context, request PolicyRequest) (PolicyDecision, error) {
	// 这里可以实现实际的地理位置检查
	// 为了简化，我们只检查 IP
	
	// 检查是否被阻止
	for _, blockedIP := range lp.blockedIPs {
		if request.Context["ip"] == blockedIP {
			return DecisionDeny, nil
		}
	}

	// 检查是否允许
	if len(lp.allowedIPs) > 0 {
		allowed := false
		for _, allowedIP := range lp.allowedIPs {
			if request.Context["ip"] == allowedIP {
				allowed = true
				break
			}
		}
		if !allowed {
			return DecisionDeny, nil
		}
	}

	return DecisionAllow, nil
}
