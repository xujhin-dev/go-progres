package worker

import (
	"log"
	"time"
	"user_crud_jwt/internal/domain/coupon/model"
	"user_crud_jwt/internal/domain/coupon/repository"
)

type CouponTask struct {
	UserID   uint
	CouponID uint
	Retry    int // 重试次数
}

type WorkerPool struct {
	TaskQueue   chan CouponTask
	RetryQueue  chan CouponTask // 重试队列
	Repo        repository.CouponRepository
	WorkerNum   int
	MaxRetry    int // 最大重试次数
}

func NewWorkerPool(repo repository.CouponRepository, workerNum int, bufferSize int) *WorkerPool {
	return &WorkerPool{
		TaskQueue:  make(chan CouponTask, bufferSize),
		RetryQueue: make(chan CouponTask, bufferSize/2),
		Repo:       repo,
		WorkerNum:  workerNum,
		MaxRetry:   3, // 最多重试3次
	}
}

func (p *WorkerPool) Start() {
	for i := 0; i < p.WorkerNum; i++ {
		go p.worker(i)
	}
	// 启动重试处理协程
	go p.retryWorker()
	log.Printf("Worker pool started with %d workers", p.WorkerNum)
}

func (p *WorkerPool) worker(id int) {
	for task := range p.TaskQueue {
		if err := p.processTask(task); err != nil {
			log.Printf("[Worker %d] Failed to process task (UserID: %d, CouponID: %d): %v",
				id, task.UserID, task.CouponID, err)

			// 如果未达到最大重试次数，加入重试队列
			if task.Retry < p.MaxRetry {
				task.Retry++
				select {
				case p.RetryQueue <- task:
					log.Printf("[Worker %d] Task added to retry queue (attempt %d/%d)",
						id, task.Retry, p.MaxRetry)
				default:
					log.Printf("[Worker %d] Retry queue full, task dropped: %+v", id, task)
					// TODO: 记录到死信队列或持久化存储
					p.logFailedTask(task, err)
				}
			} else {
				log.Printf("[Worker %d] Task exceeded max retries, dropped: %+v", id, task)
				// TODO: 记录到死信队列
				p.logFailedTask(task, err)
			}
		}
	}
}

func (p *WorkerPool) retryWorker() {
	for task := range p.RetryQueue {
		// 延迟重试，避免立即重试
		time.Sleep(time.Duration(task.Retry) * time.Second)

		// 重新加入主队列
		select {
		case p.TaskQueue <- task:
			log.Printf("[RetryWorker] Task re-queued (attempt %d/%d)", task.Retry, p.MaxRetry)
		default:
			log.Printf("[RetryWorker] Main queue full, task dropped: %+v", task)
			p.logFailedTask(task, nil)
		}
	}
}

func (p *WorkerPool) processTask(task CouponTask) error {
	// 执行数据库写入操作
	if err := p.Repo.DecreaseStock(task.CouponID); err != nil {
		return err
	}

	userCoupon := &model.UserCoupon{
		UserID:   task.UserID,
		CouponID: task.CouponID,
		Status:   1, // 1: 未使用
	}

	if err := p.Repo.CreateUserCoupon(userCoupon); err != nil {
		return err
	}

	return nil
}

func (p *WorkerPool) logFailedTask(task CouponTask, err error) {
	// TODO: 实现死信队列或持久化存储
	// 可以写入文件、数据库或消息队列
	log.Printf("[DeadLetter] Task failed permanently: UserID=%d, CouponID=%d, Error=%v",
		task.UserID, task.CouponID, err)
}

func (p *WorkerPool) AddTask(task CouponTask) {
	select {
	case p.TaskQueue <- task:
		// 任务入队成功
	default:
		log.Printf("Worker pool queue full, dropping task: %+v", task)
		// TODO: 记录到死信队列
		p.logFailedTask(task, nil)
	}
}
