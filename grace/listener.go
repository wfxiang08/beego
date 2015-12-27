package grace

import (
	"net"
	"os"
	"syscall"
	"time"
)

type graceListener struct {
	net.Listener
	stop    chan error
	stopped bool
	server  *graceServer
}

func newGraceListener(l net.Listener, srv *graceServer) (el *graceListener) {
	el = &graceListener{
		Listener: l,
		stop:     make(chan error),
		server:   srv,
	}
	go func() {
		// 这是什么逻辑?
		// 首先: 等待关闭
		_ = <-el.stop
		el.stopped = true
		// 关闭之后，在给el.stop消息
		el.stop <- el.Listener.Close()
	}()
	return
}

//
// 封装: Listener.Accept
//
func (gl *graceListener) Accept() (c net.Conn, err error) {
	tc, err := gl.Listener.(*net.TCPListener).AcceptTCP()
	if err != nil {
		return
	}

	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)

	// 创建一个Conn
	c = graceConn{
		Conn:   tc,
		server: gl.server,
	}

	gl.server.wg.Add(1)
	return
}

func (el *graceListener) Close() error {
	if el.stopped {
		return syscall.EINVAL
	}

	// 关注: newGraceListener
	el.stop <- nil
	// 在： newGraceListener#func中, 对应的Listener要准备关闭, 然后在等待Listener关闭完成，再返回
	return <-el.stop
}

func (el *graceListener) File() *os.File {
	// returns a dup(2) - FD_CLOEXEC flag *not* set
	tl := el.Listener.(*net.TCPListener)
	fl, _ := tl.File()
	return fl
}
