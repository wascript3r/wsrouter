package websocket

import (
	"net"
	"sync"
	"time"

	"github.com/wascript3r/gopool"

	"github.com/gobwas/ws"
	"github.com/mailru/easygo/netpoll"
)

type Server struct {
	Pool   *gopool.Pool
	poller netpoll.Poller
}

type Config struct {
	Listener net.Listener

	Cooldown  time.Duration
	Timeout   time.Duration
	IOTimeout time.Duration

	Ev EventHandler
}

func NewServer(pool *gopool.Pool) (*Server, error) {
	poller, err := netpoll.New(nil)
	if err != nil {
		return nil, err
	}
	return &Server{pool, poller}, nil
}

func (s *Server) Listen(cfg *Config) error {
	listenDesc, err := netpoll.HandleListener(cfg.Listener, netpoll.EventRead|netpoll.EventOneShot)
	if err != nil {
		return err
	}

	accept := make(chan error, 1)

	s.poller.Start(listenDesc, func(e netpoll.Event) {
		// Pool schedule error
		err := s.Pool.ScheduleTimeout(cfg.Timeout, func() {
			// Connection accept error
			conn, err := cfg.Listener.Accept()
			if err != nil {
				accept <- err
				return
			}

			accept <- nil

			// Handle error
			err = s.HandleConn(conn, cfg)
			if err != nil {
				// Called when "Handle error" occurs.
				cfg.Ev.OnError(err, conn)
			}
		})

		// Connection accept error
		if err == nil {
			err = <-accept
		}

		if err != nil {
			if err == gopool.ErrScheduleTimeout {
				goto cooldown
			} else if err == gopool.ErrPoolTerminated {
				s.poller.Stop(listenDesc)
				return
			}

			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				goto cooldown
			}

			// Called when "Pool schedule error" or "Connection accept error" occurs.
			cfg.Ev.OnError(err, nil)
			goto resume

		cooldown:
			t := time.NewTimer(cfg.Cooldown)
			select {
			case <-s.Pool.End():
				t.Stop()
				s.poller.Stop(listenDesc)
				return

			case <-t.C:
			}
		}

	resume:
		s.poller.Resume(listenDesc)
	})

	return nil
}

func (s *Server) HandleConn(conn net.Conn, cfg *Config) error {
	safeConn := NewSafeConn(conn, cfg.IOTimeout)
	socket := NewSocket(safeConn)

	_, err := ws.Upgrade(safeConn)
	if err != nil {
		safeConn.Close()
		return err
	}

	// Using conn instead of safeConn because of "could not get file descriptor" error.
	desc, err := netpoll.HandleRead(conn)
	if err != nil {
		safeConn.Close()
		return err
	}

	cfg.Ev.OnOpen(socket)

	s.poller.Start(desc, func(e netpoll.Event) {
		if e&(netpoll.EventReadHup|netpoll.EventHup) != 0 {
			// safeConn.Close()
			s.poller.Stop(desc)

			cfg.Ev.OnDisconnect(socket)
			return
		}

		err := s.Pool.Schedule(func() {
			socket.io.Lock()

			op, r, err := socket.getReaderUnsafe()

			if err != nil {
				safeConn.Close()
				s.poller.Stop(desc)
				socket.io.Unlock()

				cfg.Ev.OnError(err, socket)
				cfg.Ev.OnDisconnect(socket)
				return
			}

			once := sync.Once{}
			resume := func() {
				once.Do(socket.io.Unlock)
			}

			cfg.Ev.OnRead(
				socket,
				NewRequest(op, r),
				resume,
			)

			resume()
		})

		if err == gopool.ErrPoolTerminated {
			socket.ForceDisconnect("unexpected close")
			s.poller.Stop(desc)

			cfg.Ev.OnDisconnect(socket)
		}
	})

	return nil
}
