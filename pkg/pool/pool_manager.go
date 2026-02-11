package pool

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WorkerPool 工作池接口
type WorkerPool interface {
	Submit(task Task) error
	SubmitWithTimeout(task Task, timeout time.Duration) error
	Close() error
	Stats() PoolStats
}

// Task 任务接口
type Task interface {
	Execute(ctx context.Context) error
	ID() string
}

// PoolStats 池统计信息
type PoolStats struct {
	WorkerCount     int           `json:"worker_count"`
	ActiveWorkers   int           `json:"active_workers"`
	QueuedTasks     int           `json:"queued_tasks"`
	CompletedTasks  int64         `json:"completed_tasks"`
	FailedTasks     int64         `json:"failed_tasks"`
	AverageTaskTime time.Duration `json:"average_task_time"`
}

// GoRoutinePool Go 协程池实现
type GoRoutinePool struct {
	workerCount   int
	taskQueue     chan Task
	quit          chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
	stats         PoolStats
	taskStartTime map[string]time.Time
}

// NewGoRoutinePool 创建协程池
func NewGoRoutinePool(workerCount int, queueSize int) WorkerPool {
	pool := &GoRoutinePool{
		workerCount:   workerCount,
		taskQueue:     make(chan Task, queueSize),
		quit:          make(chan struct{}),
		taskStartTime: make(map[string]time.Time),
	}

	// 启动工作协程
	for i := 0; i < workerCount; i++ {
		pool.wg.Add(1)
		go pool.worker(i)
	}

	return pool
}

// worker 工作协程
func (p *GoRoutinePool) worker(id int) {
	defer p.wg.Done()
	
	for {
		select {
		case task := <-p.taskQueue:
			p.executeTask(task, id)
		case <-p.quit:
			return
		}
	}
}

// executeTask 执行任务
func (p *GoRoutinePool) executeTask(task Task, workerID int) {
	startTime := time.Now()
	taskID := task.ID()
	
	// 记录任务开始时间
	p.mu.Lock()
	p.taskStartTime[taskID] = startTime
	p.stats.ActiveWorkers++
	p.mu.Unlock()
	
	// 执行任务
	err := task.Execute(context.Background())
	
	// 更新统计信息
	p.mu.Lock()
	delete(p.taskStartTime, taskID)
	p.stats.ActiveWorkers--
	
	if err != nil {
		p.stats.FailedTasks++
	} else {
		p.stats.CompletedTasks++
	}
	
	// 计算平均任务时间
	taskDuration := time.Since(startTime)
	if p.stats.CompletedTasks > 0 {
		totalTime := time.Duration(p.stats.CompletedTasks-1) * p.stats.AverageTaskTime + taskDuration
		p.stats.AverageTaskTime = totalTime / time.Duration(p.stats.CompletedTasks)
	}
	
	p.mu.Unlock()
}

// Submit 提交任务
func (p *GoRoutinePool) Submit(task Task) error {
	select {
	case p.taskQueue <- task:
		p.mu.Lock()
		p.stats.QueuedTasks = len(p.taskQueue)
		p.mu.Unlock()
		return nil
	default:
		return fmt.Errorf("task queue is full")
	}
}

// SubmitWithTimeout 带超时的任务提交
func (p *GoRoutinePool) SubmitWithTimeout(task Task, timeout time.Duration) error {
	select {
	case p.taskQueue <- task:
		p.mu.Lock()
		p.stats.QueuedTasks = len(p.taskQueue)
		p.mu.Unlock()
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("task submission timeout after %v", timeout)
	}
}

// Close 关闭协程池
func (p *GoRoutinePool) Close() error {
	close(p.quit)
	p.wg.Wait()
	return nil
}

// Stats 获取统计信息
func (p *GoRoutinePool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	stats := p.stats
	stats.QueuedTasks = len(p.taskQueue)
	return stats
}

// SemaphorePool 信号量池（用于限制并发数）
type SemaphorePool struct {
	semaphore chan struct{}
	mu        sync.RWMutex
	stats     PoolStats
}

// NewSemaphorePool 创建信号量池
func NewSemaphorePool(maxConcurrency int) *SemaphorePool {
	return &SemaphorePool{
		semaphore: make(chan struct{}, maxConcurrency),
	}
}

// Acquire 获取信号量
func (p *SemaphorePool) Acquire(ctx context.Context) error {
	select {
	case p.semaphore <- struct{}{}:
		p.mu.Lock()
		p.stats.ActiveWorkers++
		p.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release 释放信号量
func (p *SemaphorePool) Release() {
	select {
	case <-p.semaphore:
		p.mu.Lock()
		p.stats.ActiveWorkers--
		p.mu.Unlock()
	default:
		// 信号量已满，忽略
	}
}

// Execute 在信号量保护下执行函数
func (p *SemaphorePool) Execute(ctx context.Context, fn func() error) error {
	if err := p.Acquire(ctx); err != nil {
		return err
	}
	defer p.Release()
	
	return fn()
}

// Stats 获取统计信息
func (p *SemaphorePool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return PoolStats{
		WorkerCount:   cap(p.semaphore),
		ActiveWorkers: p.stats.ActiveWorkers,
	}
}

// RateLimiter 限流器
type RateLimiter struct {
	rate     float64
	capacity int
	tokens   int
	lastTime time.Time
	mu       sync.Mutex
}

// NewRateLimiter 创建限流器
func NewRateLimiter(rate float64, capacity int) *RateLimiter {
	return &RateLimiter{
		rate:     rate,
		capacity: capacity,
		tokens:   capacity,
		lastTime: time.Now(),
	}
}

// Allow 检查是否允许请求
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(r.lastTime).Seconds()
	r.lastTime = now
	
	// 添加令牌
	r.tokens += int(elapsed * r.rate)
	if r.tokens > r.capacity {
		r.tokens = r.capacity
	}
	
	// 检查是否有足够令牌
	if r.tokens > 0 {
		r.tokens--
		return true
	}
	
	return false
}

// Wait 等待直到可以执行
func (r *RateLimiter) Wait(ctx context.Context) error {
	for !r.Allow() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond * 10):
			// 继续等待
		}
	}
	return nil
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	maxFailures  int
	resetTimeout  time.Duration
	failures      int
	lastFailTime  time.Time
	state         CircuitState
	mu            sync.RWMutex
}

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures: maxFailures,
		resetTimeout: resetTimeout,
		state:       StateClosed,
	}
}

// Call 执行函数调用
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	// 检查状态
	if cb.state == StateOpen {
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}
	
	// 执行函数
	err := fn()
	
	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()
		
		if cb.failures >= cb.maxFailures {
			cb.state = StateOpen
		}
		
		return err
	}
	
	// 成功时重置
	cb.failures = 0
	cb.state = StateClosed
	
	return nil
}

// State 获取当前状态
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}
