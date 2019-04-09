package genid

import (
	"sync"
	"time"
)

const (
	generateBits  uint8 = 10                        // 每台机器(节点)的ID位数，10位最大可以有2^10=1024个节点
	numberBits    uint8 = 12                        // 表示每个集群下的每个节点，1毫秒内可生成的id序号的二进制位数，即每毫秒可生成 2^12-1=4096 个唯一ID
	generateMax   int64 = -1 ^ (-1 << generateBits) // 节点ID的最大值，用于防止溢出，这里求最大值使用了位运算，-1 的二进制表示为 1 的补码
	numberMax     int64 = -1 ^ (-1 << numberBits)   // 同上，用来表示生成id序号的最大值
	timeShift           = generateBits + numberBits // 时间戳向左的偏移量
	generateShift       = numberBits                // 节点ID向左的偏移量
	epoch         int64 = 1525705533000             // 时间戳(毫秒)，41位字节作为时间戳数值，大约68年就会用完，这个一旦定义且开始生成ID后千万不要改了，不然可能会生成相同的ID
)

/*
定义一个generate工作节点所需要的基本参数
*/
type Generate struct {
	mu         sync.Mutex // 互斥锁
	number     int64      // 当前毫秒已经生成的id序列号(从0开始累加) 1毫秒内最多生成4096个ID
	timestamp  int64      // 记录时间戳
	generateId int64      // 该节点ID，因为snowFlake目的是解决分布式下生成唯一id，所以ID中是包含集群和节点编号在内的
}

/*
并发goroutine进行snowflakeID生成
*/
func GeneratorId(id int64) (c chan int64) {
	g := newGenerate(id)
	c = make(chan int64)
	go func() {
		for {
			c <- g.getId()
		}
	}()
	return
}

// 指定某个节点生成id
func (g *Generate) getId() (ID int64) {
	// 添加互斥锁，确保并发安全
	g.mu.Lock()
	defer g.mu.Unlock()
	// 获取生成时的时间戳
	now := time.Now().UnixNano() / 1e6 // 纳秒转毫秒
	if g.timestamp != now {
		// 如果当前时间与工作节点上一次生成ID的时间不一致，则需要重置工作节点生成ID的序号
		g.number = 0
		g.timestamp = now // 将机器上一次生成ID的时间更新为当前时间
		ID = int64((now-epoch)<<timeShift | (g.generateId << generateShift) | (g.number))
		return
	}
	g.number++
	// 这里要判断，当前工作节点是否在1毫秒内已经生成numberMax个ID
	if g.number <= numberMax {
		ID = int64((now-epoch)<<timeShift | (g.generateId << generateShift) | (g.number))
		return
	}
	// 如果当前工作节点在1毫秒内生成的ID已经超过上限，需要等待1毫秒再继续生成
	for now <= g.timestamp {
		now = time.Now().UnixNano() / 1e6
	}
	ID = int64((now-epoch)<<timeShift | (g.generateId << generateShift) | (g.number))
	return
}

/*
初始化一个节点，设置节点ID并校验
*/
func newGenerate(generateId int64) *Generate {
	// 要先校验generateId是否在上面定义的范围内
	if generateId < 0 || generateId > generateMax {
		return nil
	}
	return &Generate{
		number:     0,
		timestamp:  0,
		generateId: generateId,
	}
}
