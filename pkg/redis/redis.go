package redis

import (
	"context"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// Config Redis配置对象
type Config struct {
	Addrs []string
	Pwd   string
	DB    int
}

// Redis Redis对象
type Redis struct {
	cluster      *redis.ClusterClient
	singleClient *redis.Client
	clusterMode  bool
}

func NewRedis(c Config) *Redis {
	if len(c.Addrs) == 0 {
		return nil
	}
	r := &Redis{}
	if len(c.Addrs) == 1 {
		// 单个对象
		r.clusterMode = false
		r.singleClient = redis.NewClient(
			&redis.Options{
				Addr:         c.Addrs[0],
				Password:     c.Pwd,
				DB:           c.DB,
				DialTimeout:  3 * time.Second,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 5 * time.Second,
			})
		err := r.singleClient.Ping(context.Background()).Err()
		if err != nil {
			log.Printf(err.Error())
			return nil
		}
		return r
	} else {
		// redis 集群
		r.clusterMode = true
		r.cluster = redis.NewClusterClient(
			&redis.ClusterOptions{
				Addrs:        c.Addrs,
				Password:     c.Pwd,
				DialTimeout:  3 * time.Second,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 5 * time.Second,
			})
		err := r.cluster.Ping(context.Background()).Err()
		if err != nil {
			log.Println(err.Error())
			return nil
		}
	}
	return r
}

// Exists redis中 查找key是否存在
func (r *Redis) Exists(k string) int64 {
	if r.clusterMode {
		return r.cluster.Exists(context.Background(), k).Val()
	}
	return r.singleClient.Exists(context.Background(), k).Val()
}

// Keys redis 查找所有符合给定模式的所有key
func (r *Redis) Keys(k string) []string {
	if r.clusterMode {
		return r.cluster.Keys(context.Background(), k).Val()
	}
	return r.singleClient.Keys(context.Background(), k).Val()
}

// Del redis删除所有指定key的所有数据
func (r *Redis) Del(k string) error {
	if r.clusterMode {
		return r.cluster.Del(context.Background(), k).Err()
	}
	return r.singleClient.Del(context.Background(), k).Err()
}

// Exprie redis设置key过期时间
func (r *Redis) Expire(k string, t time.Duration) error {
	if r.clusterMode {
		return r.cluster.Expire(context.Background(), k, t).Err()
	}
	return r.singleClient.Expire(context.Background(), k, t).Err()
}

// Set redis用string的格式存储key值
func (r *Redis) Set(k, v string, t time.Duration) error {
	if r.clusterMode {
		return r.cluster.Set(context.Background(), k, v, t).Err()
	}
	return r.singleClient.Set(context.Background(), k, v, t).Err()
}

// SetNx 不存在则写入
func (r *Redis) SetNx(k, v string, t time.Duration) error {
	if r.clusterMode {
		return r.cluster.SetNX(context.Background(), k, v, t).Err()
	}
	return r.singleClient.SetNX(context.Background(), k, v, t).Err()
}

// Get redis用string的方式获取key
func (r *Redis) Get(k string) string {
	if r.clusterMode {
		return r.cluster.Get(context.Background(), k).Val()
	}
	return r.singleClient.Get(context.Background(), k).Val()
}

// HSet redis用hash的方式存储key的field字段
func (r *Redis) HSet(k, field string, value interface{}) error {
	if r.clusterMode {
		return r.cluster.HSet(context.Background(), k, field, value).Err()
	}
	return r.singleClient.HSet(context.Background(), k, field, value).Err()
}

// HGet redis用hash的形式读取key的field字段
func (r *Redis) HGet(k, field string) string {
	if r.clusterMode {
		return r.cluster.HGet(context.Background(), k, field).Val()
	}
	return r.singleClient.HGet(context.Background(), k, field).Val()
}

// HMSet redis用hash的方式存储key的field字段
func (r *Redis) HMSet(k, field string, value interface{}) error {
	if r.clusterMode {
		return r.cluster.HMSet(context.Background(), k, field, value).Err()
	}
	return r.singleClient.HMSet(context.Background(), k, field, value).Err()
}

// HMGet redis用hash的形式读取key的field字段
func (r *Redis) HMGet(k, field string) []interface{} {
	if r.clusterMode {
		return r.cluster.HMGet(context.Background(), k, field).Val()
	}
	return r.singleClient.HMGet(context.Background(), k, field).Val()
}

// HDel redis删除hash散列表key的field字段
func (r *Redis) HDel(k, field string) error {
	if r.clusterMode {
		return r.cluster.HDel(context.Background(), k, field).Err()
	}
	return r.singleClient.HDel(context.Background(), k, field).Err()
}

// HGetAll redis读取hash散列表key值对应的全部字段数据
func (r *Redis) HGetAll(k string) map[string]string {
	if r.clusterMode {
		return r.cluster.HGetAll(context.Background(), k).Val()
	}
	return r.singleClient.HGetAll(context.Background(), k).Val()
}
