# 用户功能单元测试文档

## 📋 测试概述

本文档描述了用户模块的单元测试实现，包括测试覆盖范围、测试用例设计和运行方法。

## 🧪 测试文件结构

```
internal/domain/user/service/
├── user_service.go           # 用户服务实现
├── user_service_test.go      # 用户服务单元测试
└── cached_user_service.go    # 缓存用户服务实现
```

## 🎯 测试覆盖范围

### 1. 用户服务层测试 (`user_service_test.go`)

#### 登录注册功能测试 (`TestLoginOrRegister`)
- **新用户注册成功**: 测试首次登录时自动创建用户账号
- **现有用户登录成功**: 测试已存在用户的登录流程
- **验证码无效**: 测试错误验证码的处理

#### OTP验证码功能测试 (`TestSendOTP`)
- **发送验证码成功**: 测试验证码发送功能

#### 用户管理功能测试
- **获取用户列表** (`TestGetUsers`): 测试分页获取用户列表
- **获取单个用户** (`TestGetUser`): 测试根据ID获取用户信息
- **更新用户信息** (`TestUpdateUser`): 测试用户信息更新
- **删除用户** (`TestDeleteUser`): 测试用户软删除功能

## 🔧 Mock设计

### MockUserRepository
模拟用户仓库接口，用于隔离数据库依赖：
- `Create`: 创建用户
- `GetByID`: 根据ID获取用户
- `GetByMobile`: 根据手机号获取用户
- `GetList`: 获取用户列表（分页）
- `Update`: 更新用户信息
- `UpdateMemberStatus`: 更新会员状态
- `Delete`: 删除用户

### MockOTPService
模拟OTP验证码服务，用于隔离短信服务依赖：
- `Send`: 发送验证码
- `Verify`: 验证验证码

## 📊 测试用例详情

### 登录注册测试用例

```go
// 新用户注册成功测试
t.Run("New user registration success", func(t *testing.T) {
    mobile := "13800138000"
    code := "123456"
    
    // Mock设置
    mockOTP.On("Verify", mobile, code).Return(true)
    mockRepo.On("GetByMobile", mobile).Return(nil, gorm.ErrRecordNotFound)
    mockRepo.On("Create", mock.AnythingOfType("*model.User")).Return(nil)
    mockRepo.On("Update", mock.AnythingOfType("*model.User")).Return(nil)

    // 执行测试
    token, err := service.LoginOrRegister(mobile, code)
    
    // 验证结果
    assert.NoError(t, err)
    assert.NotEmpty(t, token)
})
```

### 测试数据创建

```go
func createTestUser(id, mobile string) *model.User {
    return &model.User{
        ID:       id,
        Mobile:   mobile,
        Nickname: "TestUser",
        Role:     model.RoleUser,
        Status:   model.StatusNormal,
    }
}
```

## 🚀 运行测试

### 运行所有用户服务测试
```bash
go test ./internal/domain/user/service -v
```

### 运行特定测试
```bash
# 只运行登录注册测试
go test ./internal/domain/user/service -v -run TestLoginOrRegister

# 只运行OTP测试
go test ./internal/domain/user/service -v -run TestSendOTP
```

### 生成测试覆盖率报告
```bash
go test ./internal/domain/user/service -cover
```

### 生成详细覆盖率报告
```bash
go test ./internal/domain/user/service -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## 📈 测试覆盖率

当前测试覆盖率：**19.8%**

主要覆盖的功能：
- ✅ 用户登录/注册逻辑
- ✅ OTP验证码处理
- ✅ 基本CRUD操作
- ✅ 错误处理逻辑

## 🎯 测试最佳实践

### 1. 隔离外部依赖
- 使用Mock对象隔离数据库和外部服务
- 确保测试的独立性和可重复性

### 2. 测试数据管理
- 使用工厂函数创建测试数据
- 避免硬编码测试数据

### 3. 断言验证
- 使用明确的断言验证预期结果
- 验证错误消息内容

### 4. Mock期望设置
- 为每个测试用例创建独立的Mock实例
- 明确设置Mock的期望行为和返回值

## 🔮 未来扩展

### 计划添加的测试用例

1. **边界条件测试**
   - 空手机号处理
   - 超长手机号处理
   - 特殊字符处理

2. **并发测试**
   - 多用户同时注册
   - 并发登录测试

3. **性能测试**
   - 大量用户查询性能
   - Token生成性能

4. **集成测试**
   - 端到端登录流程
   - 与数据库的集成测试

5. **处理器层测试**
   - HTTP请求处理
   - 参数验证
   - 响应格式验证

## 🐛 常见问题

### 1. Mock期望冲突
**问题**: 多个测试用例使用同一个Mock实例导致期望冲突
**解决**: 为每个测试用例创建新的Mock实例

### 2. 时间相关测试
**问题**: 使用`time.Time{}`导致空指针异常
**解决**: 使用`time.Now().Add()`创建有效的时间值

### 3. 类型断言错误
**问题**: Mock返回值类型不匹配
**解决**: 确保Mock方法的返回值类型与接口定义一致

## 📝 总结

用户模块的单元测试已经建立，覆盖了核心业务逻辑的主要功能。测试使用Mock技术隔离了外部依赖，确保了测试的稳定性和可重复性。未来将继续扩展测试覆盖率，添加更多边界条件和性能测试。
