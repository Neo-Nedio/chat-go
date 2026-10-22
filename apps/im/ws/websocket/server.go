package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

type AckType int

const (
	NoAck    AckType = iota // 不进行应答
	OnlyAck                 // 服务端响应一次应答
	RigorAck                // 严格应答，服务端应答后客户端再进行一次应答
)

func (t AckType) ToString() string {
	switch t {
	case OnlyAck:
		return "OnlyAck"
	case RigorAck:
		return "RigorAck"
	}

	return "NoAck"
}

type Server struct {
	sync.RWMutex
	opt *serverOption

	routes         map[string]HandlerFunc
	addr           string
	patten         string
	connToUser     map[*Conn]string
	userToConn     map[string]*Conn
	upgrader       websocket.Upgrader
	authentication Authentication
	logx.Logger    // ← 嵌入 logx.Logger

	// gozero -> core -> threading
	*threading.TaskRunner // ← 嵌入 *threading.TaskRunner
}

func NewServer(addr string, opts ...ServerOption) *Server {
	opt := newOption(opts...)

	return &Server{
		routes:         make(map[string]HandlerFunc),
		opt:            &opt,
		addr:           addr,
		patten:         opt.patten,
		connToUser:     make(map[*Conn]string),
		userToConn:     make(map[string]*Conn),
		upgrader:       websocket.Upgrader{},
		authentication: opt.Authentication,
		Logger:         logx.WithContext(context.Background()),
		TaskRunner:     threading.NewTaskRunner(opt.concurrency),
	}
}

func (s *Server) ServerWs(w http.ResponseWriter, r *http.Request) {
	defer func() {
		// 处理运行过程中可能会抛出的系统性panic，为避免服务崩溃，需要恢复并捕获异常
		if r := recover(); r != nil {
			s.Errorf("server handler ws recover err %v", r)
		}
	}()

	// 获取连接对象
	conn := NewConn(s, w, r)
	if conn == nil {
		return
	}

	if !s.authentication.Auth(w, r) {
		//conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprint("不具备访问权限")))
		s.Send(&Message{FrameType: FrameData, Data: fmt.Sprint("不具备访问权限")}, conn)
		conn.Close()
		return
	}

	// 记录连接
	s.addConn(conn, r)

	// 根据连接对象获取请求，根据请求查找路由并执行
	go s.handlerConn(conn)

}

func (s *Server) addConn(conn *Conn, req *http.Request) {
	uid := s.authentication.UserId(req)

	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()

	// 验证用户是否之前登入过
	if c := s.userToConn[uid]; c != nil {
		// 关闭之前的连接
		c.Close()
	}

	s.connToUser[conn] = uid
	s.userToConn[uid] = conn
}

func (s *Server) GetConn(uid string) *Conn {
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	return s.userToConn[uid]
}

func (s *Server) GetConns(uids ...string) []*Conn {
	if len(uids) == 0 {
		return nil
	}

	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	res := make([]*Conn, 0, len(uids))
	for _, uid := range uids {
		res = append(res, s.userToConn[uid])
	}
	return res
}

func (s *Server) GetUsers(conns ...*Conn) []string {

	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	var res []string
	if len(conns) == 0 {
		// 获取全部
		res = make([]string, 0, len(s.connToUser))
		for _, uid := range s.connToUser {
			res = append(res, uid)
		}
	} else {
		// 获取部分
		res = make([]string, 0, len(conns))
		for _, conn := range conns {
			res = append(res, s.connToUser[conn])
		}
	}

	return res
}

// 读原始消息，入队
func (s *Server) handlerConn(conn *Conn) {

	uids := s.GetUsers(conn)
	conn.Uid = uids[0]

	// 开启一个处理channel管道中任务的协程
	go s.handlerWrite(conn)

	if s.isAck(nil) {
		go s.readAck(conn)
	}

	for {
		// 获取请求消息
		_, msg, err := conn.ReadMessage()
		if err != nil {
			s.Errorf("websocket conn read message err %v", err)
			s.Close(conn)
			return
		}
		// 解析消息
		var message Message
		if err = json.Unmarshal(msg, &message); err != nil {
			s.Errorf("json unmarshal err %v, msg %v", err, string(msg))
			s.Close(conn)
			return
		}

		// todo: 给客户端回复一个ack

		// 根据消息类型处理
		if s.isAck(&message) {
			// 若开启ACK则将消息放到队列中处理
			s.Infof("conn message read ack msg %v", message)
			conn.appendMsgMq(&message)
		} else {
			// 不开启则直接将消息放到连接中
			conn.message <- &message
		}
	}
}

func (s *Server) isAck(message *Message) bool {
	if message == nil {
		return s.opt.ack != NoAck
	}
	return s.opt.ack != NoAck && message.FrameType != FrameNoAck
}

/*
RigorAck 只给"重复处理会造成严重后果"的场景：
转账：重复扣款。
下单：重复创建订单。
库存扣减：超卖。*/

// readAck：负责 ACK 确认流程，确认通过才投递
func (s *Server) readAck(conn *Conn) {
	for {
		select {
		case <-conn.done:
			s.Infof("close message ack uid %v ", conn.Uid)
			return
		default:
		}

		// 从队列中读取新的消息
		conn.messageMu.Lock()

		// 队列消息为空时
		if len(conn.readMessage) == 0 {
			conn.messageMu.Unlock()
			// 增加睡眠，让任务更好地切换
			time.Sleep(100 * time.Microsecond)
			continue
		}

		// 读取第一条（保证消息顺序）
		message := conn.readMessage[0]

		// 判断ack的方式
		switch s.opt.ack {
		case OnlyAck:
			// 1. 发 ACK 给客户端
			s.Send(&Message{
				FrameType: FrameAck,
				Id:        message.Id,
				AckSeq:    message.AckSeq + 1,
			}, conn)

			// 2. 出队
			conn.readMessage = conn.readMessage[1:]
			conn.messageMu.Unlock()

			// 3. 投 message channel → handlerWrite 处理
			conn.message <- message
		case RigorAck:
			//✨过程一：先回应客户端
			if message.AckSeq == 0 {
				// AckSeq == 0表示还未进行任何ACK的确认
				conn.readMessage[0].AckSeq++
				conn.readMessage[0].ackTime = time.Now()
				s.Send(&Message{
					FrameType: FrameAck,
					Id:        message.Id,
					AckSeq:    message.AckSeq,
				}, conn)
				s.Infof("message ack RigorAck send mid %v, seq %v , time%v", message.Id, message.AckSeq,
					message.ackTime)
				conn.messageMu.Unlock()

				continue // ← 不投递，等客户端回 ACK
			}

			//✨过程二：再进行ACK验证

			//🔥1. 客户端返回确认结果，服务端再一次确认
			// 得到客户端的序号
			msgSeq := conn.readMessageSeq[message.Id]
			if msgSeq.AckSeq > message.AckSeq { // 客户端 ACK 序号前进了
				// 确认
				conn.readMessage = conn.readMessage[1:] // 出队
				conn.messageMu.Unlock()
				conn.message <- message // ← 投递
				s.Infof("message ack RigorAck success mid %v", message.Id)
				continue
			}

			//🔥2. 客户端没有确认，考虑是否超过了ack的确认时间
			val := s.opt.ackTimeout - time.Since(message.ackTime)
			if !message.ackTime.IsZero() && val <= 0 {
				// 2.2 超过结束确认
				delete(conn.readMessageSeq, message.Id) //// 放弃
				conn.readMessage = conn.readMessage[1:]
				conn.messageMu.Unlock()
				continue //// ← 不投递
			}
			//	   2.1 未超过，重新发送
			conn.messageMu.Unlock()
			s.Send(&Message{
				FrameType: FrameAck,
				Id:        message.Id,
				AckSeq:    message.AckSeq,
			}, conn)
			// 睡眠一定的时间
			time.Sleep(3 * time.Second)
		}
	}
}

// handlerWrite：真正的业务处理
func (s *Server) handlerWrite(conn *Conn) {
	for {
		select {
		// 阻塞等待 conn.done channel 输出数据或关闭
		case <-conn.done:
			// 连接关闭
			return
		case message := <-conn.message:
			switch message.FrameType {
			case FramePing:
				s.Send(&Message{FrameType: FramePing}, conn)
			case FrameData:
				// 根据请求的method分发路由并执行
				if handler, ok := s.routes[message.Method]; ok {
					handler(s, conn, message)
				} else {
					s.Send(&Message{FrameType: FrameData, Data: fmt.Sprintf("不存在执行的方法 %v 请检查", message.Method)}, conn)
					//conn.WriteMessage(&Message{}, []byte(fmt.Sprintf("不存在执行的方法 %v 请检查", message.Method)))
				}
			}

			if s.isAck(message) {
				conn.messageMu.Lock()
				delete(conn.readMessageSeq, message.Id) // 处理完清理
				conn.messageMu.Unlock()
			}
		}
	}
}

/*
这里的 ...string 表示：
这个函数的参数 sendIds 是一个“可变数量的 string 参数”。
*/
func (s *Server) SendByUserId(msg interface{}, sendIds ...string) error {
	if len(sendIds) == 0 {
		return nil
	}
	/*
		Go 里的 ... 在调用函数时有一个反向功能：
		如果一个函数接收可变参数，而你已经有一个切片（比如 []string），
		想把切片里的每个元素当作独立参数传进去，就需要加上 ... 展开操作符。
	*/
	return s.Send(msg, s.GetConns(sendIds...)...)
}

// 根据连接对象执行任务处理
func (s *Server) Send(msg interface{}, conns ...*Conn) error {
	if len(conns) == 0 {
		return nil
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	for _, conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) AddRoutes(rs []Route) {
	for _, r := range rs {
		s.routes[r.Method] = r.Handler
	}
}

func (s *Server) Start() {
	http.HandleFunc(s.patten, s.ServerWs)    // 第 1 步：注册路由（只管路径）
	s.Info(http.ListenAndServe(s.addr, nil)) // 第 2 步：监听端口（只管地址）
}

func (s *Server) Stop() {
	fmt.Println("停止服务")
}

func (s *Server) Close(conn *Conn) {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()

	uid := s.connToUser[conn]
	if uid == "" {
		// 已经被关闭
		return
	}

	delete(s.connToUser, conn)
	delete(s.userToConn, uid)

	err := conn.Close()
	if err != nil {
		return
	}
}
