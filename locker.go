package distribution_lock

import (
	"context"
	clientV3 "go.etcd.io/etcd/client/v3"
	"log"
	"os"
	"time"
)

type DistributionLocker struct {
	config         clientV3.Config
	client         *clientV3.Client
	lease          clientV3.Lease
	leaseGrantResp *clientV3.LeaseGrantResponse
	leaseId        clientV3.LeaseID
	keepRespChan   <-chan *clientV3.LeaseKeepAliveResponse
	keepResp       *clientV3.LeaseKeepAliveResponse
	txnResp        *clientV3.TxnResponse
	ctx            context.Context
	cancelFunc     context.CancelFunc
	option         Option
}

type Option struct {
	ConnectionTimeout time.Duration // 连接到etcd的超时时间，示例：5*time.Second
	Prefix            string        // 锁前缀
	Debug             bool
}

// New 创建一把锁
//  etcdEndpoints etcd连接信息，示例：[]string{"localhost:2379"}
//  connectionTimeout 连接到etcd的超时时间，示例：5*time.Second
func New(etcdEndpoints []string, option Option) (locker *DistributionLocker, err error) {
	if option.Prefix == "" {
		option.Prefix = "lock:"
	}
	if option.ConnectionTimeout <= 0 {
		option.ConnectionTimeout = 5 * time.Second
	}
	locker = &DistributionLocker{config: clientV3.Config{
		Endpoints:   etcdEndpoints,
		DialTimeout: option.ConnectionTimeout},
		option: option,
	}
	if locker.client, err = clientV3.New(locker.config); err != nil {
		return nil, err
	}

	//上锁并创建租约
	locker.lease = clientV3.NewLease(locker.client)
	// 第2个参数TTL，可以用于控制如果当前进程和etcd连接断开了，持有锁的上下文多长时间失效
	if locker.leaseGrantResp, err = locker.lease.Grant(context.TODO(), 3); err != nil {
		return nil, err
	}
	locker.leaseId = locker.leaseGrantResp.ID
	// 创建一个可取消的租约，主要是为了退出的时候能够释放
	locker.ctx, locker.cancelFunc = context.WithCancel(context.TODO())

	if locker.keepRespChan, err = locker.lease.KeepAlive(locker.ctx, locker.leaseId); err != nil {
		return nil, err
	}
	// 续约应答
	go func() {
		for {
			select {
			case locker.keepResp = <-locker.keepRespChan:
				if locker.keepRespChan == nil || locker.keepResp == nil {
					if locker.option.Debug {
						log.Printf("进程 %+v 的锁 %+v 的租约已经失效了", os.Getpid(), locker.leaseId)
					}
					return
				} else { // 每秒会续租一次, 所以就会收到一次应答
					if locker.option.Debug {
						log.Printf("进程 %+v 收到自动续租应答 %+v", os.Getpid(), locker.keepResp.ID)
					}
				}
			}
		}
	}()
	return locker, nil
}

// GetId 获得当前锁的内部ID
func (locker *DistributionLocker) GetId() int64 {
	return int64(locker.leaseId)
}

func (locker *DistributionLocker) Acquire(lockerId string) (who string, ok bool) {
	var err error
	// 在租约时间内去抢锁（etcd 里面的锁就是一个 key）
	kv := clientV3.NewKV(locker.client)

	// 创建事务
	txn := kv.Txn(context.TODO())
	// 定义锁的Key
	var lockerKey = locker.option.Prefix + lockerId
	// If 不存在 key，Then 设置它，Else 抢锁失败
	txn.If(clientV3.Compare(clientV3.CreateRevision(lockerKey), "=", 0)).
		Then(clientV3.OpPut(lockerKey, lockerId, clientV3.WithLease(locker.leaseId))).
		Else(clientV3.OpGet(lockerKey))
	if locker.txnResp, err = txn.Commit(); err != nil {
		return "", false
	}

	if !locker.txnResp.Succeeded {
		return string(locker.txnResp.Responses[0].GetResponseRange().Kvs[0].Value), false
	}
	return "", true
}

func (locker *DistributionLocker) Release() error {
	locker.cancelFunc()
	if _, err := locker.lease.Revoke(context.TODO(), locker.leaseId); err != nil {
		return err
	}
	return nil
}
