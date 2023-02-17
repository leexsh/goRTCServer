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
