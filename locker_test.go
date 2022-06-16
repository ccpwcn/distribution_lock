package distribution_lock

import (
	"log"
	"os"
	"testing"
	"time"
)

func TestDistributionLockerOneAsDebug(t *testing.T) {
	option := Option{
		ConnectionTimeout: 5 * time.Second,
		Prefix:            "",
		Debug:             true,
	}
	if locker, err := New([]string{"localhost:2379"}, option); err != nil {
		log.Fatalf("创建锁失败：%+v", err)
	} else if who, ok := locker.Acquire("TestDistributionLockerOne"); ok {
		// 抢到锁后执行业务逻辑，没有抢到则退出
		t.Logf("进程 %+v 持有锁 %+v 正在处理任务中...", os.Getpid(), locker.GetId())
		time.Sleep(8 * time.Second) // 这是正在做的事情，假定耗时20秒
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

func TestDistributionLockerOneNoneDebugAndPrefix(t *testing.T) {
	option := Option{
		ConnectionTimeout: 3 * time.Second,
		Prefix:            "MyDistributionLocker",
		Debug:             false,
	}
	if locker, err := New([]string{"localhost:2379"}, option); err != nil {
		log.Fatalf("创建锁失败：%+v", err)
	} else if who, ok := locker.Acquire("TestDistributionLockerOne"); ok {
		// 抢到锁后执行业务逻辑，没有抢到则退出
		t.Logf("进程 %+v 持有锁 %+v 正在处理任务中...", os.Getpid(), locker.GetId())
		time.Sleep(10 * time.Second) // 这是正在做的事情，假定耗时20秒
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

func TestDistributionLockerMulti(t *testing.T) {

}
