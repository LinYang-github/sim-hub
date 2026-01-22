package data

import (
	"log/slog"
	"time"

	"github.com/liny/sim-hub/internal/conf"
	"github.com/nats-io/nats.go"
)

type NATSClient struct {
	Conn    *nats.Conn
	Encoded *nats.EncodedConn
	Config  conf.NATS
}

func NewNATS(c *conf.NATS) (*NATSClient, error) {
	if !c.Enabled {
		slog.Debug("NATS 消息队列已禁用 (本地队列模式)")
		return &NATSClient{Config: *c}, nil
	}

	opts := []nats.Option{
		nats.Name("SimHub API"),
		nats.Timeout(5 * time.Second),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(5),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			slog.Warn("NATS 断开连接", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.Info("NATS 重新连接成功", "url", nc.ConnectedUrl())
		}),
	}

	nc, err := nats.Connect(c.URL, opts...)
	if err != nil {
		return nil, err
	}

	// 使用 JSON 编码的连接，方便直接传输结构体
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		nc.Close()
		return nil, err
	}

	slog.Info("NATS 连接成功", "url", c.URL, "subject", c.Subject)

	return &NATSClient{
		Conn:    nc,
		Encoded: ec,
		Config:  *c,
	}, nil
}

func (n *NATSClient) Close() {
	if n.Encoded != nil {
		n.Encoded.Close()
	}
}
