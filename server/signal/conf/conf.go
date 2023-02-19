package conf

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
)

var (
	cfg = config{}
	// 全局配置
	Global = &cfg.Global
	// Etc设置
	Etcd = &cfg.Etcd
	// 信令服务设置
	Signal = &cfg.Signal
	// Nats设置
	Nats = &cfg.Nats
	// kafka中间件设置
	Kafka = &cfg.Kafka
)

func init() {
	if !cfg.parse() {
		showHelp()
		os.Exit(-1)
	}
}

type global struct {
	Pprof  string `mapstructure:"pprof"`
	NodeDC string `mapstructure:"dc"`
	Name   string `mapstructure:"name"`
	NodeID string `mapstructure:"id"`
}

type etcd struct {
	Adds []string `mapstructure:"addrs"`
}

type nats struct {
	URL string `mapstructure:"url"`
}

type signal struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
	Cert string `mapstructure:"cert"`
	Key  string `mapstructure:"key"`
}

type kafka struct {
	URL string `mapstructure:"url"`
}

type config struct {
	Global  global `mapstructure:"global"`
	Etcd    etcd   `mapstructure:"etcd"`
	Nats    nats   `mapstructure:"nats"`
	Signal  signal `mapstructure:"signal"`
	Kafka   kafka  `mapstructure:"kafka"`
	CfgFile string
}

func showHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -c {config file}")
	fmt.Println("      -h (show help info)")
}

func (c *config) load() bool {
	_, err := os.Stat(c.CfgFile)
	if err != nil {
		return false
	}

	viper.SetConfigFile(c.CfgFile)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		log.Printf("config file %s read failed. err is %v\n", c.CfgFile, err)
		return false
	}
	err = viper.GetViper().UnmarshalExact(c)
	if err != nil {
		log.Printf("config file %s load failed. err is %v\n", c.CfgFile, err)
		return false
	}
	fmt.Printf("config %s load successed\n", c.CfgFile)
	return true
}

func (c *config) parse() bool {
	flag.StringVar(&c.CfgFile, "c", "cfg/conf.toml", "config file")
	help := flag.Bool("h", false, "help info")
	flag.Parse()
	if !c.load() {
		return false
	}
	if *help {
		showHelp()
		return false
	}
	return true
}
