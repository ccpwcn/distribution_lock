# 说明
etcd是新一代的服务发现与配置中心，官网 https://etcd.io/ ，同时兼顾有Redis的部分功能，可以以K-V模式存储数据。

基于它（类似于Zookeeper）实现一套稳定、可靠、强一致的分布式锁，在以后的应用开发中大有用处。

# 开发
这个分布式锁组件依赖于`go.etcd.io/etcd/client/v3`。

你使用时，只需要在你的工程是使用下列命令即可：
```shell
go get github.com/ccpwcn/distribution_lock
```

代码调用方法请参见测试用例。

# 实现原理
它的实现原理和基于zookeeper的实现原理本质上是一样的：
- 在分布式多进程环境下，锁不能设在进程中，因为两个服务是两个进程，甚至在两台计算机上
- 和集中式锁相比，分布式锁可以支持并发环境下，尤其是高并发环境下的一些数据竞争问题
- 进程1先与etcd建立连接，基于该连接设置k-v的值和版本号，形成一把锁
- 进程1对该连接中，会不断进行续命，只要当前连接没有断开（调用release方法之前），就一直连着，锁一直有效
- 如果进程1的锁主动释放了（明确调用了release方法），就不再续命
- 在进程1持有锁期间另外的进程2申请锁，与etcd建立连接
- 进程2基于该连接设置k-v的值和版本号，设置失败，获取锁失败
- 如果进程1在锁没有释放之前意外中止了（崩溃了、被kill掉了等等），它的锁不会再续命，连接被释放，其他进程此时可申请到锁

> Tips：无论如何，申请锁和释放锁的操作成对出现是个好习惯，只申请不释放的做法不值得提倡。

# 亲测可用
- 锁名相同，竞争测试通过，只会有一个进程获得锁，不会出现数据不一致的问题
- 锁名不同，各不影响，各走各的业务逻辑

因此可以在不同的业务场景下，分别定义好自己的锁名各自使用，互不影响。

# 本地测试
可以将下面的代码保存为main.go编译成一个exe，然后连续启动多个进行竞争测试。
```go
package main

import (
	"flag"
	"fmt"
	"github.com/ccpwcn/distribution_lock"
	"log"
	"os"
	"time"
)

var ver = "1.0.1"

func main() {
	namePtr := flag.String("n", "", "锁名称")
	mockTaskDurationPtr := flag.Int("m", 10, "模拟任务时长")
	verPtr := flag.Bool("v", false, "版本")
	flag.Parse()
	if *verPtr {
		fmt.Printf("distribution_lock %s\n", ver)
		return
	}
	if *namePtr == "" {
		fmt.Println("必须指定一个锁名称")
		return
	}
	option := distribution_lock.Option{
		ConnectionTimeout: 5 * time.Second,
		Prefix:            "",
		Debug:             true,
	}
	if locker, err := distribution_lock.New([]string{"localhost:2379"}, option); err != nil {
		log.Fatalf("创建锁失败：%+v", err)
	} else if who, ok := locker.Acquire(*namePtr); ok {
		// 抢到锁后执行业务逻辑，没有抢到则退出
		log.Printf("进程 %+v 持有锁 %+v 正在处理任务中...", os.Getpid(), locker.GetId())
		time.Sleep(time.Duration(*mockTaskDurationPtr) * time.Second) // 这是正在做的事情
		log.Printf("进程 %+v 的任务处理完了", os.Getpid())
		// 手动释放锁，在后台应用服务中，也可以通过defer释放
		if err := locker.Release(); err != nil {
			log.Fatalf("释放锁失败：%+v", err)
		}
		time.Sleep(2 * time.Second)
	} else {
		log.Printf("获取锁失败，锁现在在 %+v 手中", who)
	}
}
```