package security

// PermissionWrite 权限写入操作
type PermissionWrite struct {
	rbac *RBAC
}

// NewPermissionWrite 创建权限写入实例
func NewPermissionWrite(rbac *RBAC) *PermissionWrite {
	return &PermissionWrite{rbac: rbac}
}

// WritePermission 写入权限
func (pw *PermissionWrite) WritePermission(userID string, permission Permission) error {
	pw.rbac.mu.Lock()
	defer pw.rbac.mu.Unlock()
	
	if pw.rbac.userPermissions == nil {
		pw.rbac.userPermissions = make(map[string][]Permission)
	}
	
	// 检查权限是否已存在
	for _, p := range pw.rbac.userPermissions[userID] {
		if p == permission {
			return nil // 权限已存在
		}
	}
	
	// 添加新权限
	pw.rbac.userPermissions[userID] = append(pw.rbac.userPermissions[userID], permission)
	return nil
}
