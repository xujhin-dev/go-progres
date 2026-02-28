#!/bin/bash

# 项目编译错误快速修复脚本

echo "🔧 开始修复编译错误..."

# 1. 修复UserRepository接口问题
echo "修复UserRepository接口..."
cat > /Users/xujing/Desktop/learn/go-progres/internal/domain/user/repository/interface.go << 'EOF'
package repository

import (
	"context"
	"user_crud_jwt/internal/domain/user/model"
)

// UserRepository 用户仓库接口
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByMobile(ctx context.Context, mobile string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*model.User, error)
	GetList(ctx context.Context, limit, offset int) ([]*model.User, error)
	Count(ctx context.Context) (int64, error)
	UpdateMemberStatus(ctx context.Context, id string, status int) error
}
EOF

# 2. 修复CouponRepository接口问题
echo "修复CouponRepository接口..."
cat > /Users/xujing/Desktop/learn/go-progres/internal/domain/coupon/repository/interface.go << 'EOF'
package repository

import (
	"context"
	"user_crud_jwt/internal/domain/coupon/model"
)

// CouponRepository 优惠券仓库接口
type CouponRepository interface {
	// 基础CRUD操作
	Create(ctx context.Context, coupon *model.Coupon) error
	CreateCoupon(ctx context.Context, coupon *model.Coupon) error
	GetByID(ctx context.Context, id string) (*model.Coupon, error)
	GetCouponByID(ctx context.Context, id string) (*model.Coupon, error)

	// 库存管理
	DecreaseStock(ctx context.Context, couponID string) error
	DecreaseCouponStock(ctx context.Context, couponID string) error

	// 用户优惠券管理
	CreateUserCoupon(ctx context.Context, userCoupon *model.UserCoupon) error
	GetUserCoupon(ctx context.Context, userID, couponID string) (*model.UserCoupon, error)
	HasUserClaimed(ctx context.Context, userID, couponID string) (bool, error)
	CountUserCoupons(ctx context.Context, userID, couponID string) (int64, error)
}
EOF

# 3. 创建简化的MomentRepository接口
echo "修复MomentRepository接口..."
cat > /Users/xujing/Desktop/learn/go-progres/internal/domain/moment/repository/interface.go << 'EOF'
package repository

import (
	"context"
	"user_crud_jwt/internal/domain/moment/model"
)

// MomentRepository 动态仓库接口
type MomentRepository interface {
	// 动态相关
	CreatePost(ctx context.Context, post *model.Post) error
	GetPostByID(ctx context.Context, id string) (*model.Post, error)
	GetPosts(ctx context.Context, limit, offset int) ([]*model.Post, error)
	GetPostsByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.Post, error)
	UpdatePost(ctx context.Context, post *model.Post) error
	UpdatePostStatus(ctx context.Context, id string, status string) error
	DeletePost(ctx context.Context, id string) error
	
	// 评论相关
	CreateComment(ctx context.Context, comment *model.Comment) error
	GetCommentByID(ctx context.Context, id string) (*model.Comment, error)
	GetCommentsByPostID(ctx context.Context, postID string, limit, offset int) ([]*model.Comment, error)
	UpdateComment(ctx context.Context, comment *model.Comment) error
	DeleteComment(ctx context.Context, id string) error
	
	// 点赞相关
	CreateLike(ctx context.Context, like *model.Like) error
	DeleteLike(ctx context.Context, userID, targetID string, targetType string) error
	GetLikesByTarget(ctx context.Context, targetID string, targetType string, limit, offset int) ([]*model.Like, error)
	HasLiked(ctx context.Context, userID, targetID string) (bool, error)
	
	// 话题相关
	CreateTopic(ctx context.Context, topic *model.Topic) error
	GetTopicByID(ctx context.Context, id string) (*model.Topic, error)
	GetTopicByName(ctx context.Context, name string) (*model.Topic, error)
	GetTopicsByName(ctx context.Context, name string) ([]*model.Topic, error)
	GetTopics(ctx context.Context, limit, offset int) ([]*model.Topic, error)
	DeleteTopic(ctx context.Context, id string) error
	AssociatePostWithTopic(ctx context.Context, postID, topicID string) error
}
EOF

# 4. 创建简化的PaymentRepository接口
echo "修复PaymentRepository接口..."
cat > /Users/xujing/Desktop/learn/go-progres/internal/domain/payment/repository/interface.go << 'EOF'
package repository

import (
	"context"
	"user_crud_jwt/internal/domain/payment/model"
)

// PaymentRepository 支付仓库接口
type PaymentRepository interface {
	// 订单相关
	CreateOrder(ctx context.Context, order *model.Order) error
	GetOrderByID(ctx context.Context, id string) (*model.Order, error)
	GetOrderByNo(ctx context.Context, orderNo string) (*model.Order, error)
	GetOrdersByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.Order, error)
	UpdateOrder(ctx context.Context, order *model.Order) error
	UpdateOrderStatus(ctx context.Context, id string, status string) error
	
	// 支付相关
	CreatePayment(ctx context.Context, payment *model.Payment) error
	GetPaymentByOrderID(ctx context.Context, orderID string) (*model.Payment, error)
	UpdatePaymentStatus(ctx context.Context, id string, status string) error
	
	// 退款相关
	CreateRefund(ctx context.Context, refund *model.Refund) error
	GetRefundByOrderID(ctx context.Context, orderID string) (*model.Refund, error)
	UpdateRefundStatus(ctx context.Context, id string, status string) error
}
EOF

echo "✅ 接口修复完成！"
echo "🧪 测试编译..."

cd /Users/xujing/Desktop/learn/go-progres
go build ./cmd/main.go

if [ $? -eq 0 ]; then
    echo "🎉 主程序编译成功！"
else
    echo "❌ 主程序编译失败"
fi

echo "📝 修复总结："
echo "- 统一了所有Repository接口"
echo "- 添加了缺失的方法"
echo "- 修复了方法签名不一致问题"
echo "- 项目架构更加清晰"
