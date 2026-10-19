package websocket

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Conn struct {
	idleMu sync.Mutex

	Uid string // 连接id

	*websocket.Conn
	s *Server

	lastRead          time.Time // 最后一次成功读到消息的时间
	maxConnectionIdle time.Duration

	messageMu      sync.Mutex
	readMessage    []*Message          // 解决"顺序"，只有队首处理完（确认通过），才处理下一条
	readMessageSeq map[string]*Message //  解决"去重 + ACK 进度"
	message        chan *Message       // 确认后的消息，投给 handlerWrite 处理

	// channel 用来通知当Conn停止时相应方法结束
	done chan struct{}
}

func NewConn(s *Server, w http.ResponseWriter, r *http.Request) *Conn {
	c, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.Errorf("upgrade err %v", err)
		return nil
	}

	conn := &Conn{
		Conn:              c,
		s:                 s,
		lastRead:          time.Now(),              // 连接建立即算活跃，给一个完整超时窗口
		maxConnectionIdle: s.opt.maxConnectionIdle, // 从选项取值，修掉写死默认值的 bug
		readMessage:       make([]*Message, 0, 2),
		readMessageSeq:    make(map[string]*Message, 2),
		//✨通道大小设置为1：可以减少数据投递中的阻塞情况，也可以保障这个收和发的执行的顺序问题
		message: make(chan *Message, 1),
		done:    make(chan struct{}),
	}

	go conn.keepalive()
	return conn
}

/*
handlerConn（读循环）          readAck（确认）           handlerWrite（业务）
│                            │                          │
│ 收到消息                    │                          │
├─ 开ACK → appendMsgMq ─────▶ readMessage 队列           │
│                              │ 取队首                  │
│                              ├─ 发ACK/等ACK            │
│                              ├─ 确认通过 → 出队         │
│                              └─ 投 message ───────────▶ │ 从 message 取
│                                                         └─ 路由到 handler
└─ 不开ACK → message channel ────────────────────────────▶*/

// appendMsgMq 是消息去重 + 排序的入队逻辑，处理客户端发来的消息，决定"要不要放进待 ACK 队列"。
func (c *Conn) appendMsgMq(msg *Message) {
	c.messageMu.Lock()
	defer c.messageMu.Unlock()

	// 消息 ID 已存在
	if m, ok := c.readMessageSeq[msg.Id]; ok {
		if len(c.readMessage) == 0 {
			return // 队列空，忽略
		}
		if msg.AckSeq <= m.AckSeq {
			return // ACK 没前进 → 重复，丢弃
		}
		c.readMessageSeq[msg.Id] = msg // ACK 前进 → 更新 map
		return
	}

	// 新消息但是 ACK 帧 → 多余 ACK，丢弃
	if msg.FrameType == FrameAck {
		return
	}

	// 新消息 → 入队 + 记录
	c.readMessage = append(c.readMessage, msg)
	c.readMessageSeq[msg.Id] = msg

}

// ReadMessage 只在成功读到消息时刷新 lastRead
func (c *Conn) ReadMessage() (messageType int, p []byte, err error) {
	messageType, p, err = c.Conn.ReadMessage()
	if err != nil {
		return
	}

	c.idleMu.Lock()
	c.lastRead = time.Now()
	c.idleMu.Unlock()
	return
}

// WriteMessage 只做并发写保护，不参与活跃判断
func (c *Conn) WriteMessage(messageType int, data []byte) error {
	return c.Conn.WriteMessage(messageType, data)
}

func (c *Conn) Close() error {
	select {
	case <-c.done:
		// 已经关过了，什么都不做
	default:
		// close(c.done) 是唯一让 <-c.done 解除阻塞的操作
		//一旦 close，所有等在这个 channel 上的 <-c.done 都会立刻返回（读到零值 struct{}{}）
		//close 一个 channel 后，后续再 close 会 panic，所以前面用 select + default 判断是否已 close，保证幂等。
		close(c.done)
	}

	return c.Conn.Close()
}

// keepalive 只判断"多久没读到消息"，超过 maxConnectionIdle 就关闭连接
func (c *Conn) keepalive() {
	idleTimer := time.NewTimer(c.maxConnectionIdle)
	defer idleTimer.Stop()

	for {
		select {
		case <-idleTimer.C:
			c.idleMu.Lock()
			lastRead := c.lastRead
			c.idleMu.Unlock()

			idleFor := time.Since(lastRead)
			if idleFor >= c.maxConnectionIdle {
				// 超过规定时间没收到任何消息，判定连接死亡，优雅关闭
				c.s.Close(c)
				return
			}
			// 还没超时，按剩余时间重排定时器
			idleTimer.Reset(c.maxConnectionIdle - idleFor)
		case <-c.done:
			return // ← 收到关闭信号，退出 goroutine
		}
	}
}
