package LoadBalance

import (
	"errors"
	"strconv"
)

// NewWeightRoundRobinBalance 创建一个加权轮询负载均衡器
func NewWeightRoundRobinBalance() LoadBalance {
	return &WeightRoundRobinBalance{}
}

type WeightRoundRobinBalance struct {
	curIndex int
	rss      []*WeightNode
	rsw      []int

	//观察主体
	conf loadBalanceConf
}

// loadBalanceConf 配置主题
type loadBalanceConf interface {
	GetConf() []string
	WatchConf()
	UpdateConf(conf []string)
}

type WeightNode struct {
	addr            string // 服务器地址
	weight          int    //权重值
	currentWeight   int    //节点当前权重
	effectiveWeight int    //有效权重
}

func (r *WeightRoundRobinBalance) Add(params ...string) error {
	if len(params) != 2 {
		return errors.New("param len need 2")
	}
	parInt, err := strconv.ParseInt(params[1], 10, 64)
	if err != nil {
		return err
	}
	node := &WeightNode{addr: params[0], weight: int(parInt)}
	node.effectiveWeight = node.weight
	r.rss = append(r.rss, node)
	return nil
}

func (r *WeightRoundRobinBalance) Next() string {
	total := 0
	var best *WeightNode
	for i := 0; i < len(r.rss); i++ {
		w := r.rss[i]
		//step 1 统计所有有效权重之和
		total += w.effectiveWeight

		//step 2 变更节点临时权重为的节点临时权重+节点有效权重
		w.currentWeight += w.effectiveWeight

		//step 3 有效权重默认与权重相同，通讯异常时-1, 通讯成功+1，直到恢复到weight大小
		if w.effectiveWeight < w.weight {
			w.effectiveWeight++
		}
		//step 4 选择最大临时权重点节点
		if best == nil || w.currentWeight > best.currentWeight {
			best = w
		}
	}
	if best == nil {
		return ""
	}
	//step 5 变更临时权重为 临时权重-有效权重之和
	best.currentWeight -= total
	return best.addr
}

func (r *WeightRoundRobinBalance) Get(key string) (string, error) {
	return r.Next(), nil
}

func (r *WeightRoundRobinBalance) SetConf(conf loadBalanceConf) {
	r.conf = conf
}

func (r *WeightRoundRobinBalance) Delete(addr string) error {
	index := -1
	for i, node := range r.rss {
		if node.addr == addr {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("node not found")
	}
	r.rss = append(r.rss[:index], r.rss[index+1:]...)
	return nil
}
