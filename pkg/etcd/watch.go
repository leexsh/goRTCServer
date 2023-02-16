package etcd

import (
	"log"
	"strconv"
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	// ServerUp 服务存活
	ServerUp = 0
	// ServerDown 服务死亡
	ServerDown = 1
)

// ServiceWatchCallback 定义服务节点状态改变的回调
type ServiceWatchCallback func(status int32, node Node)

// ServiceWatcher 服务发现对象
type ServiceWatcher struct {
	etcd     *Etcd
	bstop    bool
	nodes    map[string]Node
	nodeLock sync.Mutex
	callback ServiceWatchCallback
}

// NewServiceWatcher 新建一个服务发现对象
func NewServiceWatcher(endpoints []string) *ServiceWatcher {
	etcd, err := NewEtcd(endpoints)
	if err != nil {
		log.Printf("NewServiceWatcher err, err is %v", err)
		return nil
	}
	return &ServiceWatcher{
		etcd:     etcd,
		bstop:    false,
		nodes:    make(map[string]Node),
		nodeLock: sync.Mutex{},
		callback: nil,
	}
}

// Close 关闭资源
func (s *ServiceWatcher) Close() {
	s.bstop = true
	if s.etcd != nil {
		s.etcd.Close()
	}
}

// GetNodes 根据服务名称获取所有该服务节点的所有对象
func (s *ServiceWatcher) GetNodes(serviceName string) (map[string]Node, bool) {
	s.nodeLock.Lock()
	defer s.nodeLock.Unlock()
	mapNodes := make(map[string]Node)
	for _, node := range s.nodes {
		if node.Name == serviceName {
			mapNodes[node.NodeID] = node
		}
	}
	if len(mapNodes) > 0 {
		return mapNodes, true
	}
	return mapNodes, false
}

// GetNodeByID 根据服务节点的ID获取服务节点对象
func (s *ServiceWatcher) GetNodeByID(nid string) (*Node, bool) {
	s.nodeLock.Lock()
	defer s.nodeLock.Unlock()
	node, ok := s.nodes[nid]
	if ok {
		return &node, true
	}
	return nil, false
}

// GetNodeByPayload 获取指定区域内指定服务节点负载最低的节点
func (s *ServiceWatcher) GetNodeByPayload(dc, name string) (*Node, bool) {
	var nodePtr *Node = nil
	var payload int = 65535
	s.nodeLock.Lock()
	defer s.nodeLock.Unlock()
	for _, node := range s.nodes {
		if node.NodeDC == dc && node.NodeDC == name {
			pay, _ := strconv.Atoi(node.NodePayload)
			if pay < payload {
				nodePtr = &node
				payload = pay
			}
		}
	}
	if nodePtr != nil {
		return nodePtr, true
	}
	return nil, false
}

// DeleteNodeByID 删除指定id的服务节点
func (s *ServiceWatcher) DeleteNodeByID(nid string) bool {
	s.nodeLock.Lock()
	defer s.nodeLock.Unlock()
	_, ok := s.nodes[nid]
	if ok {
		delete(s.nodes, nid)
		return true
	}
	return false
}

// WatchNode 监控服务节点的状态改变
func (s *ServiceWatcher) WatchNode(ch clientv3.WatchChan) {
	go func() {
		for {
			if s.bstop {
				return
			}
			msg := <-ch
			for _, ev := range msg.Events {
				if ev.Type == clientv3.EventTypePut {
					nid := string(ev.Kv.Key)
					mpNode := Decode(ev.Kv.Value)
					if mpNode["NodeID"] != "" && mpNode["NodeID"] == nid {
						node := Node{
							NodeDC:      mpNode[NDC],
							NodeID:      mpNode[NID],
							Name:        mpNode[NNAME],
							NodePayload: mpNode[NLOAD],
						}
						s.nodeLock.Lock()
						s.nodes[nid] = node
						s.nodeLock.Unlock()

						log.Printf("Node update, ID is [%s]", nid)
						if s.callback != nil {
							s.callback(ServerUp, node)
						}
					}
				} else if ev.Type == clientv3.EventTypeDelete {
					nid := string(ev.Kv.Key)
					node, ok := s.GetNodeByID(nid)
					if ok {
						log.Printf("Node delete, ID is [%s]", nid)
						if s.callback != nil {
							s.callback(ServerDown, *node)
						}
						s.DeleteNodeByID(nid)
					}
				}
			}
		}
	}()
}

// WatchServiceNode 监控指定服务名称的所有服务节点的状态
func (s *ServiceWatcher) WatchServiceNode(prefix string, callback ServiceWatchCallback) {
	s.callback = callback
	s.GetServiceNodes(prefix)
	s.etcd.Watch(prefix, s.WatchNode, true)
}

// GetServiceNodes 获取已经存在的节点
func (s *ServiceWatcher) GetServiceNodes(prefix string) {
	resp, err := s.etcd.GetResponseByPrefix(prefix)
	if err != nil {
		log.Println(err.Error())
	}

	for _, kv := range resp.Kvs {
		mpNode := Decode(kv.Value)
		if mpNode["NodeID"] != "" {
			node := Node{
				mpNode[NDC],
				mpNode[NID],
				mpNode[NNAME],
				mpNode[NLOAD],
			}
			s.nodeLock.Lock()
			s.nodes[node.NodeID] = node
			s.nodeLock.Unlock()

			log.Printf("Find Node , nodeID is [%s]", node.NodeID)
			if s.callback != nil {
				s.callback(ServerUp, node)
			}
		}
	}
}
