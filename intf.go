package distribution_lock

import "context"

type GenerateDistributionLocker interface {
	// GetId 获得锁的身份标识符
	// 只有持有锁的时候调用此方法才有意义
	GetId() int64
	// Acquire 获得锁
	// lockerId 锁ID，推荐使用UUID或雪花算法，确保唯一性，防止复杂业务+大量数据的情况下发生锁冲撞
	// 返回值：who 如果获得锁失败，此ID可以标示锁现在在谁手中（这个谁，来自于GetId()的返回值
	//        换句话说，A进程获得锁之后，可以通过GetId知道自己的ID是多少，此时B进程获得锁失败，可以通过who返回值知道锁在A手中
	Acquire(lockerId string) (who int64, ok bool)
	// TryAcquire 可以添加一个上下文，用于控制超时，如果超出指定时间了还没有获得锁，返回失败
	TryAcquire(ctx context.Context, lockerId string) (who int64, ok bool)
	Release() error
}
