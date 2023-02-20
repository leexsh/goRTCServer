package etcd

import "encoding/json"

const (
	NDC   = "NodeDC"
	NID   = "NodeID"
	NNAME = "NodeName"
	NLOAD = "NODEPAYLOAD"
)

type Node struct {
	NodeDC      string // 节点区域
	NodeID      string // 节点id
	Name        string // 节点名称
	NodePayload string // 节点负载
}

// Encode 将map转换为string
func Encode(data map[string]string) string {
	if data == nil {
		str, _ := json.Marshal(data)
		return string(str)
	}
	return ""
}

// Decode 将string转换为map
func Decode(str []byte) map[string]string {
	if len(str) > 0 {
		var data map[string]string
		json.Unmarshal(str, &data)
		return data
	}
	return nil
}

// 获取节点信息
func (n *Node) GetNodeValue() string {
	data := map[string]string{}
	data[NDC] = n.NodeDC
	data[NID] = n.NodeID
	data[NNAME] = n.Name
	data[NLOAD] = n.NodePayload
	return Encode(data)
}

// GetRPCChannel 获取RPC对象string
func GetPRCChannel(n Node) string {
	return "rpc-" + n.NodeID
}

// GetEventChannel 获取广播对象string
func GetEventChannel(n Node) string {
	return "event-" + n.NodeID
}
