package distribution_lock

type GenerateDistributionLocker interface {
	GetId() int64
	// Acquire 获得锁
	// lockerId 锁ID，推荐使用UUID或雪花算法，确保唯一性，防止复杂业务+大量数据的情况下发生锁冲撞
	// 返回值：who 如果获得锁失败，此ID可以标示锁现在在谁手中
	Acquire(lockerId string) (who string, ok bool)
	Release() error
}
