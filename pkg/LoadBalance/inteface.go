package LoadBalance

type LoadBalance interface {
	// Add 添加节点,参数1为节点地址,参数2为节点权重
	Add(...string) error
	// Next 轮询获取下一个节点
	Next() string
	// Delete 获取所有节点
	Delete(addr string) error
}
