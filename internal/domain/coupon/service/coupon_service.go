package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
	"user_crud_jwt/internal/domain/coupon/model"
	"user_crud_jwt/internal/domain/coupon/repository"
	"user_crud_jwt/internal/pkg/worker"

	"github.com/redis/go-redis/v9"
)

type CouponService interface {
	CreateCoupon(name string, total int, amount float64, startTime, endTime time.Time) (*model.Coupon, error)
	ClaimCoupon(userID, couponID string) error
	SendCouponToUser(userID, couponID string) error
}

type couponService struct {
	repo       repository.CouponRepository
	rdb        *redis.Client
	soldOutMap sync.Map // 本地缓存：记录已售罄的 CouponID
	workerPool *worker.WorkerPool
}

func NewCouponService(repo repository.CouponRepository, rdb *redis.Client) CouponService {
	// 初始化 Worker Pool (5个 Worker，缓冲队列 1000)
	pool := worker.NewWorkerPool(repo, 5, 1000)
	pool.Start()

	return &couponService{
		repo:       repo,
		rdb:        rdb,
		workerPool: pool,
	}
}

func (s *couponService) CreateCoupon(name string, total int, amount float64, startTime, endTime time.Time) (*model.Coupon, error) {
	coupon := &model.Coupon{
		Name:      name,
		Total:     total,
		Stock:     total,
		Amount:    amount,
		StartTime: startTime,
		EndTime:   endTime,
	}

	if err := s.repo.Create(coupon); err != nil {
		return nil, err
	}

	// 预热缓存：将库存写入 Redis
	stockKey := fmt.Sprintf("coupon:stock:%s", coupon.ID)
	s.rdb.Set(context.Background(), stockKey, total, 0)

	return coupon, nil
}

// Lua 脚本：检查用户是否已领 + 检查库存 + 扣减库存 + 记录用户已领
var claimScript = redis.NewScript(`
	local user_key = KEYS[1]
	local stock_key = KEYS[2]
	local user_id = ARGV[1]

	-- 1. 检查用户是否已领取
	if redis.call("SISMEMBER", user_key, user_id) == 1 then
		return -1 -- 已领取
	end

	-- 2. 检查库存
	local stock = tonumber(redis.call("GET", stock_key))
	if stock <= 0 then
		return -2 -- 库存不足
	end

	-- 3. 扣减库存
	redis.call("DECR", stock_key)
	-- 4. 记录用户已领取
	redis.call("SADD", user_key, user_id)

	return 1 -- 成功
`)

func (s *couponService) ClaimCoupon(userID, couponID string) error {
	// 0. 本地缓存校验 (极高性能，无需网络 IO)
	if _, ok := s.soldOutMap.Load(couponID); ok {
		return errors.New("coupon out of stock (local cache)")
	}

	ctx := context.Background()
	userKey := fmt.Sprintf("coupon:users:%s", couponID)
	stockKey := fmt.Sprintf("coupon:stock:%s", couponID)

	// 1. 执行 Lua 脚本进行预扣减
	result, err := claimScript.Run(ctx, s.rdb, []string{userKey, stockKey}, userID).Int()
	if err != nil {
		return fmt.Errorf("redis error: %v", err)
	}

	if result == -1 {
		return errors.New("you have already claimed this coupon")
	}
	if result == -2 {
		// 标记本地缓存为已售罄
		s.soldOutMap.Store(couponID, true)
		return errors.New("coupon out of stock")
	}

	// 2. Redis 扣减成功后，异步写入数据库 (通过 Worker Pool)
	// 相比之前的同步写入，这里的吞吐量将极大提升
	s.workerPool.AddTask(worker.CouponTask{
		UserID:   userID,
		CouponID: couponID,
	})

	return nil
}

// SendCouponToUser 管理员给用户发券 (复用 ClaimCoupon 逻辑，或实现特定逻辑)
func (s *couponService) SendCouponToUser(userID, couponID string) error {
	// 管理员发券本质上也是扣减库存并增加用户券记录
	// 这里直接复用 ClaimCoupon 逻辑，保证库存一致性
	// 如果需要绕过"每个用户限领一张"的限制，可以单独写 Lua 脚本或逻辑
	// 假设需求是管理员可以给用户发多张，或者也受限制，这里默认受限制
	return s.ClaimCoupon(userID, couponID)
}
