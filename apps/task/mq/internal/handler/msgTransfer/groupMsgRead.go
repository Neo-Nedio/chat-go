package msgTransfer

import (
	"chat-go/apps/im/ws/ws"
	"sync"
	"time"
)

/*
这个缓冲器（groupMsgRead + groupMsgs）的作用是：
把群聊里短时间内的多条已读回执，合并成一条推送发出去，从而减少 WebSocket 推送次数、降低服务端和客户端的压力。
*/
type groupMsgRead struct {
	mu sync.Mutex

	conversationId string // 会话ID

	push     *ws.Push      // 用于记录消息
	pushChan chan *ws.Push // 推送用户合并消息的通道

	count    int       // 计数
	pushTime time.Time // 上次推送时间

	done chan struct{}
}

func newGroupMsgRead(push *ws.Push, pushChan chan *ws.Push) *groupMsgRead {
	m := &groupMsgRead{
		conversationId: push.ConversationId,
		push:           push,
		pushChan:       pushChan,
		count:          1,
		pushTime:       time.Now(),
		done:           make(chan struct{}),
	}

	return m
}

// mergePush 合并消息
func (g *groupMsgRead) mergePush(push *ws.Push) {
	g.mu.Lock()
	defer g.mu.Unlock()
	// 说明已经被清理，重新设置
	if g.push == nil {
		g.push = push
	}

	g.count++
	for msgId, read := range push.ReadRecords {
		// 如果存在相同消息，进行替换即可
		// 原因：msgReadTransfer中对于已读消息的处理，是在原来的基础上进行更改，所以我们只需要记录就可以覆盖了
		g.push.ReadRecords[msgId] = read
	}
}

// IsIdle 判断是否为活跃状态
func (g *groupMsgRead) IsIdle() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.isIdle()
}

func (g *groupMsgRead) isIdle() bool {
	// 获取上一次推送时间
	lastPushTime := g.pushTime
	// *2的原因：防止在推送时刻由于阻塞未推送，给消费者延长时间，作为检测的空闲时间
	val := GroupMsgReadRecordDelayTime*2 - time.Since(lastPushTime)

	if val <= 0 && g.push == nil && g.count == 0 {
		return true
	}
	return false
}

func (m *groupMsgRead) Clear() {
	select {
	case <-m.done:
	default:
		close(m.done)
	}

	m.push = nil
}
