package etcd

import (
	"errors"
	"log"
	"strconv"
	"time"
)

// ServiceNode 服务注册对象
type ServiceNode struct {
	etcd *Etcd
	node Node
}

// NewServiceNode 新建一个服务注册对象
func NewServiceNode(endpoints []string, dc, nid, name string) *ServiceNode {
	etcd, err := NewEtcd(endpoints)
	if err != nil {
		log.Printf("NewServiceNode err, err = %v", err)
		return nil
	}
	return &ServiceNode{
		etcd: etcd,
		node: Node{
			NodeDC:      dc,
			NodeID:      nid,
			Name:        name,
			NodePayload: "0",
		},
	}
}

// Close 关闭资源
func (s *ServiceNode) Close() {
	if s.etcd != nil {
		s.etcd.Close()
	}
}

// NodeInfo 返回服务节点信息
func (s *ServiceNode) NodeInfo() Node {
	return s.node
}

// GetRPCChannel 获取RPC对象string
func (s *ServiceNode) GetRPCChannel() string {
	return GetPRCChannel(s.node)
}

// GetEventChannel 获取广播对象string
func (s *ServiceNode) GetEventChannel() string {
	return GetEventChannel(s.node)
}

// RegisterNode 注册服务节点
func (s *ServiceNode) RegisterNode() error {
	if s.node.NodeDC == "" || s.node.NodeID == "" || s.node.Name == "" {
		return errors.New("Node dc id or name must be non empty")
	}
	go s.keepRegistered(s.node)
	return nil
}

// UpdateNodePayload 更新节点负载
func (s *ServiceNode) UpdateNodePayload(payload int) error {
	if s.node.NodePayload != strconv.Itoa(payload) {
		s.node.NodePayload = strconv.Itoa(payload)
		go s.updateRegistered(s.node)
	}
	return nil
}

// keepRegister 注册一个服务节点到etcd服务上
func (s *ServiceNode) keepRegistered(node Node) {
	for {
		err := s.etcd.Keep(node.NodeID, node.GetNodeValue())
		if err != nil {
			log.Printf("keeyRegistered node err, err is %v", err)
			time.Sleep(5 * time.Second)
		} else {
			log.Printf("Node[%s] keepRegistered succes!", node.NodeID)
			return
		}
	}
}

// updateRegistered 更新一个服务节点到etcd服务上
func (s *ServiceNode) updateRegistered(node Node) {
	for {
		err := s.etcd.Update(node.NodeID, node.GetNodeValue())
		if err != nil {
			log.Printf("updateRegistered node err, err is %v", err)
			time.Sleep(5 * time.Second)
		} else {
			log.Printf("Node[%s] updateRegistered success!", node.NodeID)
			return
		}
	}
}
