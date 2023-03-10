package conf

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

var (
	cfg = config{}
	// Global 全局设置
	Global = &cfg.Global
	// Etcd Etcd设置
	Etcd = &cfg.Etcd
	// Nats 消息中间件设置
	Nats = &cfg.Nats
	// WebRTC rtc参数
	WebRTC = &cfg.WebRTC
	// Kafka 中间件
	Kafka = &cfg.Kafka
	Ogg   = &cfg.Ogg
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

type ogg struct {
	OPEN bool `mapstructure:"open"`
}

type kafka struct {
	URL string `mapstructure:"url"`
}

type iceserver struct {
	URLS        []string `mapstructure:"urls"`
	Username    string   `mapstructure:"username"`
	Credentiail string   `mapstructure:"credential"`
}

type webrtc struct {
	ICEPortRange []uint16    `mapstructure:"portrange"`
	ICEServers   []iceserver `mapstructure:"iceserver"`
}

type config struct {
	Global  global `mapstructure:"global"`
	Etcd    etcd   `mapstructure:"etcd"`
	Nats    nats   `mapstructure:"nats"`
	WebRTC  webrtc `mapstructure:"webrtc"`
	Kafka   kafka  `mapstructure:"kafka"`
	Ogg     ogg    `mapstructure:"ogg"`
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
		fmt.Printf("config file %s read failed. %v\n", c.CfgFile, err)
		return false
	}
	err = viper.GetViper().UnmarshalExact(c)
	if err != nil {
		fmt.Printf("config file %s loaded failed. %v\n", c.CfgFile, err)
		return false
	}

	if len(c.WebRTC.ICEPortRange) > 2 {
		fmt.Printf("config file %s loaded failed. range port must be [min,max]\n", c.CfgFile)
		return false
	}

	if len(c.WebRTC.ICEPortRange) != 0 && c.WebRTC.ICEPortRange[1]-c.WebRTC.ICEPortRange[0] <= 100 {
		fmt.Printf("config file %s loaded failed. range port must be [min, max] and max - min >= %d\n", c.CfgFile, 100)
		return false
	}

	fmt.Printf("config %s load ok!\n", c.CfgFile)
	return true
}

func (c *config) parse() bool {
	flag.StringVar(&c.CfgFile, "c", "conf/conf.toml", "config file")
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
