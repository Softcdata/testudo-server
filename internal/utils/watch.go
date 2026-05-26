package utils

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/websocket"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/transport"
	"k8s.io/apimachinery/pkg/watch"
)

// WatchEventDTO defines the standard structure for watch events in the data field of Envelope
type WatchEventDTO struct {
	Type   string      `json:"type"`   // ADDED, MODIFIED, DELETED, ERROR
	Object interface{} `json:"object"` // The DTO of the resource
}

// WatchOptions watch 的配置选项
type WatchOptions struct {
	Timeout      time.Duration          // 超时时间，默认 30 分钟
	SendInterval time.Duration          // 心跳间隔，默认 30 秒
	FilterFunc   func(watch.Event) bool // 事件过滤函数
}

// DefaultWatchOptions 返回默认的 watch 配置
func DefaultWatchOptions() *WatchOptions {
	return &WatchOptions{
		Timeout:      30 * time.Minute,
		SendInterval: 30 * time.Second,
		FilterFunc:   nil, // 默认不过滤
	}
}

// WatcherFunc 定义 watcher 创建函数类型
// 用于创建 Kubernetes watch.Interface
type WatcherFunc func(ctx context.Context) (watch.Interface, error)

// WebSocket 升级器配置
var upgrader = websocket.HertzUpgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(ctx *app.RequestContext) bool {
		// 允许所有来源（生产环境应该限制）
		return true
	},
}

// WatchMessage WebSocket 消息结构
// StreamWatch 通用的 watch 处理函数，使用 WebSocket 实时推送资源变化
// 参数：
//   - c: 上下文
//   - ctx: Hertz 请求上下文
//   - watcherFunc: 创建 watcher 的函数
//   - converter: 将原始 k8s 对象转换为 DTO 的函数
//   - opts: watch 配置选项（可选）
func StreamWatch(c context.Context, ctx *app.RequestContext, watcherFunc WatcherFunc, converter func(interface{}) interface{}, opts ...*WatchOptions) {
	// 获取配置选项
	var opt *WatchOptions
	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	} else {
		opt = DefaultWatchOptions()
	}

	// 升级到 WebSocket
	err := upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		defer conn.Close()

		// 互斥锁保护 WebSocket 写入操作，防止并发写入 panic
		var writeMu sync.Mutex

		// 线程安全的写入辅助函数
		safeWrite := func(data interface{}, meta interface{}) error {
			writeMu.Lock()
			defer writeMu.Unlock()
			return sendSuccess(ctx, conn, data, meta)
		}

		safeWriteErrorKey := func(code int, key string, args map[string]any) error {
			writeMu.Lock()
			defer writeMu.Unlock()
			return sendErrorKey(ctx, conn, code, key, args)
		}

		// 创建 watcher
		watcher, err := watcherFunc(c)
		if err != nil {
			safeWriteErrorKey(transport.CodeInternalServerError, i18n.KeyWebSocketWatcherCreateFailed, map[string]any{"error": err})
			return
		}
		defer watcher.Stop()

		// 发送连接成功消息
		safeWrite(map[string]string{"status": "connected"}, map[string]string{"type": "connected"})

		// 创建一个 context 用于优雅关闭
		watchCtx, cancel := context.WithTimeout(c, opt.Timeout)
		defer cancel()

		// 启动心跳 goroutine
		heartbeatDone := make(chan struct{})
		go func() {
			ticker := time.NewTicker(opt.SendInterval)
			defer ticker.Stop()
			defer close(heartbeatDone)

			for {
				select {
				case <-ticker.C:
					if err := safeWrite(nil, map[string]string{"type": "heartbeat"}); err != nil {
						return
					}
				case <-watchCtx.Done():
					return
				}
			}
		}()

		// 启动读取 goroutine（处理客户端消息，如心跳响应或控制命令）
		readDone := make(chan struct{})
		go func() {
			defer close(readDone)
			for {
				// 读取客户端消息（保持连接活跃）
				_, _, err := conn.ReadMessage()
				if err != nil {
					cancel() // 客户端断开，取消 context
					return
				}
			}
		}()

		// 监听 Kubernetes 事件
		for {
			select {
			case event, ok := <-watcher.ResultChan():
				if !ok {
					safeWrite(nil, map[string]string{"type": "closed", "reason": "watcher closed"})
					return
				}

				// 如果设置了过滤函数，应用过滤
				if opt.FilterFunc != nil && !opt.FilterFunc(event) {
					continue
				}

				// 转换对象为 DTO
				var dto interface{}
				if converter != nil && event.Object != nil {
					dto = converter(event.Object)
				} else {
					dto = event.Object
				}

				// 构造并发送事件
				watchEvent := WatchEventDTO{
					Type:   string(event.Type),
					Object: dto,
				}

				if err := safeWrite(watchEvent, nil); err != nil {
					return
				}

			case <-watchCtx.Done():
				// 超时或客户端断开
				if watchCtx.Err() == context.DeadlineExceeded {
					safeWrite(nil, map[string]string{"type": "timeout", "reason": "connection timeout"})
				}
				return

			case <-readDone:
				// 客户端断开连接
				return
			}
		}
	})

	if err != nil {
		transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyWebSocketUpgradeFailed, map[string]any{"error": err}, nil)
	}
}

func sendSuccess(ctx *app.RequestContext, conn *websocket.Conn, data interface{}, meta interface{}) error {
	envelope := transport.Success(ctx, data, meta)
	return conn.WriteJSON(envelope)
}

func sendError(ctx *app.RequestContext, conn *websocket.Conn, code int, message string) error {
	envelope := transport.Error(ctx, code, message, nil)
	return conn.WriteJSON(envelope)
}

func sendErrorKey(ctx *app.RequestContext, conn *websocket.Conn, code int, key string, args map[string]any) error {
	envelope := transport.ErrorKey(ctx, code, key, args, nil)
	return conn.WriteJSON(envelope)
}

func WithFilter(filterFunc func(watch.Event) bool) *WatchOptions {
	opts := DefaultWatchOptions()
	opts.FilterFunc = filterFunc
	return opts
}
