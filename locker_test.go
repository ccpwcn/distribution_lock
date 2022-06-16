package distribution_lock

import (
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var etcdEndpoint = []string{"localhost:2379"}

// 一把锁，开调试
func TestDistributionLockerOneAsDebug(t *testing.T) {
	option := Option{
		ConnectionTimeout: 5 * time.Second,
		Prefix:            "",
		Debug:             true,
	}
	if locker, err := New(etcdEndpoint, option); err != nil {
		log.Fatalf("创建锁失败：%+v", err)
	} else if who, ok := locker.Acquire("DistributionLockerOneAsDebug"); ok {
		// 抢到锁后执行业务逻辑，没有抢到则退出
		t.Logf("进程 %+v 持有锁 %+v 正在处理任务中...", os.Getpid(), locker.GetId())
		time.Sleep(8 * time.Second) // 这是正在做的事情，假定耗时8秒
		t.Logf("进程 %+v 的任务处理完了", os.Getpid())
		// 手动释放锁，在后台应用服务中，也可以通过defer释放
		if err := locker.Release(); err != nil {
			log.Fatalf("释放锁失败：%+v", err)
		}
		time.Sleep(2 * time.Second)
	} else {
		t.Logf("获取锁失败，锁现在在 %+v 手中", who)
	}
}

// 一把锁，不开调试带前缀
func TestDistributionLockerOneNoneDebugAndPrefix(t *testing.T) {
	option := Option{
		ConnectionTimeout: 3 * time.Second,
		Prefix:            "MyDistributionLocker",
		Debug:             false,
	}
	if locker, err := New(etcdEndpoint, option); err != nil {
		log.Fatalf("创建锁失败：%+v", err)
	} else if who, ok := locker.Acquire("DistributionLockerOneNoneDebugAndPrefix"); ok {
		// 抢到锁后执行业务逻辑，没有抢到则退出
		t.Logf("进程 %+v 持有锁 %+v 正在处理任务中...", os.Getpid(), locker.GetId())
		time.Sleep(10 * time.Second) // 这是正在做的事情，假定耗时10秒
		t.Logf("进程 %+v 的任务处理完了", os.Getpid())
		// 手动释放锁，在后台应用服务中，也可以通过defer释放
		if err := locker.Release(); err != nil {
			log.Fatalf("释放锁失败：%+v", err)
		}
		time.Sleep(1 * time.Second)
	} else {
		t.Logf("获取锁失败，锁现在在 %+v 手中", who)
	}
}

// 一把锁，多任务（多请求）竞争锁，
// 此测试用例还可以通过命令 go test -run="TestDistributionLockerMultiTask" 开多个进程进行并行竞争测试
// 多进程测试时的结果验证方法，条件：多个测试只要有一个未完成，预期结果是：获取锁失败，successCount的值就是0
func TestDistributionLockerMultiTask(t *testing.T) {
	const taskCount = 5
	option := Option{
		ConnectionTimeout: 3 * time.Second,
		Prefix:            "MyDistributionLocker",
		Debug:             false,
	}
	var successCount int64 = 0
	var wg sync.WaitGroup
	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		go func(taskId int) {
			defer wg.Done()
			if locker, err := New(etcdEndpoint, option); err != nil {
				log.Fatalf("[%+v]创建锁失败：%+v", taskId, err)
			} else if who, ok := locker.Acquire("DistributionLockerMulti"); ok {
				// 抢到锁后执行业务逻辑，没有抢到则退出
				t.Logf("[%+v]进程 %+v 持有锁 %+v 正在处理任务中...", taskId, os.Getpid(), locker.GetId())
				atomic.AddInt64(&successCount, 1)
				time.Sleep(10 * time.Second) // 这是正在做的事情，假定耗时10秒
				t.Logf("[%+v]进程 %+v 的任务处理完了", taskId, os.Getpid())
				// 手动释放锁，在后台应用服务中，也可以通过defer释放
				if err := locker.Release(); err != nil {
					log.Fatalf("[%+v]释放锁失败：%+v", taskId, err)
				}
				time.Sleep(1 * time.Second)
			} else {
				t.Logf("[%+v]获取锁失败，锁现在在 %+v 手中", taskId, who)
			}
		}(i)
	}
	wg.Wait()
	if successCount != 1 {
		t.Fatalf("进程 %+v 的分布式锁功能存在BUG", os.Getpid())
	}
}

// 多把锁，多任务（多请求），各有各的锁
func TestDistributionLockerMultiBusinessMultiLocker(t *testing.T) {
	const taskCount = 5
	option := Option{
		ConnectionTimeout: 3 * time.Second,
		Prefix:            "MyDistributionLocker",
		Debug:             false,
	}
	var successCount int64 = 0
	var wg sync.WaitGroup
	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		go func(taskId int) {
			defer wg.Done()
			if locker, err := New(etcdEndpoint, option); err != nil {
				log.Fatalf("[%+v]创建锁失败：%+v", taskId, err)
			} else if who, ok := locker.Acquire(fmt.Sprintf("DistributionLockerMulti_%d", taskId)); ok {
				// 抢到锁后执行业务逻辑，没有抢到则退出
				t.Logf("[%+v]进程 %+v 持有锁 %+v 正在处理任务中...", taskId, os.Getpid(), locker.GetId())
				atomic.AddInt64(&successCount, 1)
				time.Sleep(10 * time.Second) // 这是正在做的事情，假定耗时10秒
				t.Logf("[%+v]进程 %+v 的任务处理完了", taskId, os.Getpid())
				// 手动释放锁，在后台应用服务中，也可以通过defer释放
				if err := locker.Release(); err != nil {
					log.Fatalf("[%+v]释放锁失败：%+v", taskId, err)
				}
				time.Sleep(1 * time.Second)
			} else {
				t.Logf("[%+v]获取锁失败，锁现在在 %+v 手中", taskId, who)
			}
		}(i)
	}
	wg.Wait()
	if successCount != taskCount {
		t.Fatalf("进程 %+v 的分布式锁功能存在BUG", os.Getpid())
	}
}

func TestDistributionLocker_GetId(t *testing.T) {
	option := Option{
		ConnectionTimeout: 3 * time.Second,
		Prefix:            "DistributionLocker_GetId",
		Debug:             false,
	}
	if locker, err := New(etcdEndpoint, option); err != nil {
		log.Fatalf("创建锁失败：%+v", err)
	} else if who, ok := locker.Acquire("DistributionLocker_GetId"); ok {
		// 抢到锁后执行业务逻辑，没有抢到则退出
		t.Logf("进程 %+v 持有锁 %+v 正在处理任务中...", os.Getpid(), locker.GetId())
		time.Sleep(5 * time.Second) // 这是正在做的事情，假定耗时5秒
		t.Logf("进程 %+v 的任务处理完了", os.Getpid())
		// 手动释放锁，在后台应用服务中，也可以通过defer释放
		if err := locker.Release(); err != nil {
			log.Fatalf("释放锁失败：%+v", err)
		}
		time.Sleep(1 * time.Second)
	} else {
		t.Logf("获取锁失败，锁现在在 %+v 手中", who)
	}
}
