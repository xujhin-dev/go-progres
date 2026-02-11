package response

// 业务状态码
const (
	CodeSuccess = 0
	CodeError   = 1

	// 用户模块错误 100xx
	ErrUserExists    = 10001
	ErrUserNotFound  = 10002
	ErrAuthFailed    = 10003
	ErrTokenInvalid  = 10004
	ErrNoPermission  = 10005

	// 优惠券模块错误 200xx
	ErrCouponNotFound   = 20001
	ErrCouponOutOfStock = 20002
	ErrCouponClaimed    = 20003

	// 系统错误 500xx
	ErrServerInternal = 50001
	ErrInvalidParam   = 50002
	ErrTooManyRequests = 50003
)
