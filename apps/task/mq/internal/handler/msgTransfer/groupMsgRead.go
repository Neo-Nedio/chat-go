package msgTransfer

import (
	"chat-go/apps/im/ws/ws"
	"chat-go/pkg/constants"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

/*
这个缓冲器（groupMsgRead + groupMsgs）的作用是：
把群聊里短时间内的多条已读回执，合并成一条推送发出去，从而减少 WebSocket 推送次数、降低服务端和客户端的压力。
*/
/*群聊已读事件 → Consume → mergePush 累加进 g.push
↓
groupMsgRead.transfer() 循环
├─ 攒够 DelayCount 条 → 发 g.push 到 m.push
├─ 超时 DelayTime → 发 g.push 到 m.push
└─ 空闲 → 发空 push 到 m.push（通知回收）
↓
MsgReadTransfer.transfer() 从 m.push 取
├─ 真正 Transfer 推送（空 push 因无 RecvId 被跳过）
└─ IsIdle 则 Clear + delete → close(done) → 上面的循环退出*/
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

	go m.transfer()

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

// 在消息合并后进行检查，与websocket中的心跳检测机制相似
func (g *groupMsgRead) transfer() {
	// 超时发送
	timer := time.NewTimer(GroupMsgReadRecordDelayTime / 2)
	defer timer.Stop()
	for {
		select {
		case <-g.done: // 会话缓冲被 Clear，退出
			return
		case <-timer.C: // 定时检查：是否超时
			g.mu.Lock()
			// 获取上一次推送时间
			lastPushTime := g.pushTime
			// *2，防止在推送时刻由于阻塞未推送
			val := GroupMsgReadRecordDelayTime*2 - time.Since(lastPushTime)
			// 得到待推送的数据
			push := g.push
			// 当前没有超时且没有超过最大计数，或数据为空
			if val > 0 && g.count < GroupMsgReadRecordDelayCount || push == nil {
				// 没到时间，重新定时
				if val > 0 {
					timer.Reset(val)
				}
				// 未达标
				g.mu.Unlock()
				continue
			}
			// 达标：清空状态，推送
			g.pushTime = time.Now()
			g.push = nil
			g.count = 0
			timer.Reset(GroupMsgReadRecordDelayTime / 2)
			g.mu.Unlock()
			// 推送
			logx.Infof("merge push delay time condition reached, push: %v ", push)
			g.pushChan <- push
		default: // 非阻塞分支：每轮都检查条数/空闲
			g.mu.Lock()
			// 消息数量达到最大值
			if g.count >= GroupMsgReadRecordDelayCount {
				// 达标，推送
				// 得到待推送的数据
				push := g.push
				g.push = nil
				g.count = 0
				g.mu.Unlock()
				// 推送
				logx.Infof("merge push max delay count condition reached, push: %v ", push)
				g.pushChan <- push
				continue
			}
			// 空闲时发"空 push"让上层回收
			if g.isIdle() {
				g.mu.Unlock()
				// 使得 msgReadTransfer 释放
				g.pushChan <- &ws.Push{
					ChatType:       constants.GroupChatType,
					ConversationId: g.conversationId,
				}
				continue
			}
			// 默认检测时，发现合并后的消息没有问题，且当前状态为活跃状态
			// 则设置默认的延迟时间缓一下
			g.mu.Unlock()
			// 活跃但没达标，短暂睡眠
			tempDelay := GroupMsgReadRecordDelayTime / 4
			if tempDelay > time.Second {
				tempDelay = time.Second
			}
			time.Sleep(tempDelay)
		}
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
