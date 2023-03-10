package etcd

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	defaultDialTimeout      = time.Second * 5
	defaultGrantTimeout     = 5
	defaultOperationTimeout = time.Second * 5
)

// watchCallback watch 回调
type WatchCallback func(watchChan clientv3.WatchChan)

type Etcd struct {
	client        *clientv3.Client            // etcd 客户端
	liveKeyID     map[string]clientv3.LeaseID // 租约map
	liveKeyIDLock sync.RWMutex                // map租约锁
	stop          bool                        // 停止开关
}

func NewEtcd(endpoints []string) (*Etcd, error) {
	cfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: defaultDialTimeout,
	}
	cli, err := clientv3.New(cfg)
	if err != nil {
		log.Printf("NewEtcd err = %v", err)
		return nil, err
	}
	etcd := new(Etcd)
	etcd.client = cli
	etcd.liveKeyID = make(map[string]clientv3.LeaseID)
	etcd.stop = false
	return etcd, nil
}

// keep 写入key-value并且保活
func (e *Etcd) Keep(key, value string) error {
	resp, err := e.client.Grant(context.TODO(), defaultGrantTimeout)
	if err != nil {
		log.Printf("Etcd Grant err, key is %s, err is %v", key, err)
		return err
	}
	_, err = e.client.Put(context.TODO(), key, value, clientv3.WithLease(resp.ID))
	if err != nil {
		log.Printf("Etcd put err, key is %s, err is %v", key, err)
		return err
	}
	ch, err := e.client.KeepAlive(context.TODO(), resp.ID)
	if err != nil {
		log.Printf("Etcd KeepAlive err, key is %s, err is %v", key, err)
		return err
	}
	// 读chan
	go func() {
		for {
			if e.stop {
				return
			}
			<-ch
		}
	}()

	// 加入map
	e.liveKeyIDLock.Lock()
	e.liveKeyID[key] = resp.ID
	e.liveKeyIDLock.Unlock()
	log.Printf("Etcd Keep ok, key is %s, value is %s", key, value)
	return nil
}

// update 更新k-v
func (e *Etcd) Update(key, value string) error {
	// 查询原来的id
	e.liveKeyIDLock.Lock()
	id := e.liveKeyID[key]
	e.liveKeyIDLock.Unlock()

	// 更新写入
	_, err := e.client.Put(context.TODO(), key, value, clientv3.WithLease(id))
	if err != nil {
		// 出错就重新keep
		err = e.Keep(key, value)
		if err != nil {
			log.Printf("Etcd update err, key is %s, err is %v", key, err)
		}
	}
	return err
}

// Delete 删除key, prefix是否前缀
func (e *Etcd) Delete(key string, prefix bool) error {
	var err error
	if prefix {
		for k := range e.liveKeyID {
			if strings.HasPrefix(k, key) {
				delete(e.liveKeyID, k)
			}
		}
		_, err = e.client.Delete(context.TODO(), key, clientv3.WithPrefix())
	} else {
		e.liveKeyIDLock.Lock()
		delete(e.liveKeyID, key)
		e.liveKeyIDLock.Unlock()
		_, err = e.client.Delete(context.TODO(), key)
	}
	return err
}

// watch 观察指定的key, 有改变通过watchFunc回调告知
func (e *Etcd) Watch(key string, watchFunc WatchCallback, prefix bool) error {
	if watchFunc == nil {
		return errors.New("watchFunc is nil")
	}
	if prefix {
		watchFunc(e.client.Watch(context.Background(), key, clientv3.WithPrefix()))
	} else {
		watchFunc(e.client.Watch(context.Background(), key))
	}
	return nil
}

// close 关闭etcd对象
func (e *Etcd) Close() error {
	if e.stop {
		return errors.New("Etcd aleady close")
	}
	e.stop = true
	e.liveKeyIDLock.Lock()
	defer e.liveKeyIDLock.Unlock()
	for k, _ := range e.liveKeyID {
		delete(e.liveKeyID, k)
		e.client.Delete(context.TODO(), k)
	}
	return e.client.Close()
}

// GetValue 获取指定key的值
func (e *Etcd) GetValue(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	resp, err := e.client.Get(ctx, key)
	if err != nil {
		cancel()
		return "", err
	}
	var val string
	for _, ev := range resp.Kvs {
		val = string(ev.Value)
	}
	cancel()
	return val, err
}

// GetByPrefix 获取指定前缀的key对应的值,map格式返回
func (e *Etcd) GetByPrefix(key string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	resp, err := e.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		cancel()
		return nil, err
	}
	data := make(map[string]string)
	for _, kv := range resp.Kvs {
		data[string(kv.Key)] = string(kv.Value)
	}
	cancel()
	return data, err
}

// GetResponseByPrefix 获取指定前缀的key对应的值,对象格式返回
func (e *Etcd) GetResponseByPrefix(key string) (*clientv3.GetResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	resp, err := e.client.Get(ctx, key, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
	if err != nil {
		cancel()
		return nil, err
	}
	cancel()
	return resp, nil
}
