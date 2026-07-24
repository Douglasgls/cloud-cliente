package forwarding

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"

	"cloud-client/pkg/logger"
)

type TCPProxy struct {
	id         string
	localPort  int
	targetHost string
	remotePort int
	dialer     Dialer
	log        *logger.Logger

	listener  net.Listener
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	mu        sync.Mutex
}

func NewTCPProxy(id string, localPort int, targetHost string, remotePort int, dialer Dialer, log *logger.Logger) *TCPProxy {
	return &TCPProxy{
		id:         id,
		localPort:  localPort,
		targetHost: strings.TrimSpace(targetHost),
		remotePort: remotePort,
		dialer:     dialer,
		log:        log,
	}
}

func (p *TCPProxy) ID() string {
	return p.id
}

func (p *TCPProxy) LocalPort() int {
	return p.localPort
}

func (p *TCPProxy) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.isRunning
}

func (p *TCPProxy) logInfo(msg string, args ...interface{}) {
	if p.log != nil {
		p.log.Info(msg, args...)
	}
}

func (p *TCPProxy) logError(msg string, args ...interface{}) {
	if p.log != nil {
		p.log.Error(msg, args...)
	}
}

func (p *TCPProxy) Start() error {
	p.mu.Lock()
	if p.isRunning {
		p.mu.Unlock()
		return nil
	}

	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(p.localPort))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		p.mu.Unlock()
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.listener = listener
	p.ctx = ctx
	p.cancel = cancel
	p.isRunning = true
	p.mu.Unlock()

	p.logInfo("[Proxy %s] escutando em 127.0.0.1:%d (destino: %s:%d)", p.id, p.localPort, p.targetHost, p.remotePort)
	go p.acceptLoop()
	return nil
}

func (p *TCPProxy) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return nil
	}

	p.isRunning = false
	if p.cancel != nil {
		p.cancel()
	}
	if p.listener != nil {
		err := p.listener.Close()
		p.listener = nil
		p.logInfo("[Proxy %s] parando proxy na porta %d", p.id, p.localPort)
		return err
	}
	return nil
}

func (p *TCPProxy) acceptLoop() {
	for {
		p.mu.Lock()
		listener := p.listener
		running := p.isRunning
		p.mu.Unlock()

		if !running || listener == nil {
			return
		}

		clientConn, err := listener.Accept()
		if err != nil {
			p.mu.Lock()
			running = p.isRunning
			p.mu.Unlock()
			if !running {
				return
			}
			if p.ctx != nil {
				select {
				case <-p.ctx.Done():
					return
				default:
					continue
				}
			}
			return
		}

		go p.handleConn(clientConn)
	}
}

type closeWriter interface {
	CloseWrite() error
}

func closeWrite(conn net.Conn) {
	if cw, ok := conn.(closeWriter); ok {
		_ = cw.CloseWrite()
	}
}

type copyResult struct {
	dir   string
	bytes int64
	err   error
}

func (p *TCPProxy) handleConn(clientConn net.Conn) {
	defer clientConn.Close()

	clientAddr := clientConn.RemoteAddr().String()
	p.logInfo("[Proxy %s] cliente conectado: %s -> 127.0.0.1:%d", p.id, clientAddr, p.localPort)

	targetAddr := net.JoinHostPort(p.targetHost, strconv.Itoa(p.remotePort))
	targetConn, err := p.dialer.DialContext(p.ctx, "tcp", targetAddr)
	if err != nil {
		p.logError("[Proxy %s] erro ao conectar ao destino %s: %v", p.id, targetAddr, err)
		return
	}
	defer targetConn.Close()

	ch := make(chan copyResult, 2)

	// Goroutine 1: Cliente -> Remoto (Bytes enviados)
	go func() {
		n, err := io.Copy(targetConn, clientConn)
		closeWrite(targetConn)
		ch <- copyResult{dir: "cliente -> remoto", bytes: n, err: err}
	}()

	// Goroutine 2: Remoto -> Cliente (Bytes recebidos)
	go func() {
		n, err := io.Copy(clientConn, targetConn)
		closeWrite(clientConn)
		ch <- copyResult{dir: "remoto -> cliente", bytes: n, err: err}
	}()

	// Espera primeira direção finalizar
	var first copyResult
	select {
	case first = <-ch:
	case <-p.ctx.Done():
		p.logInfo("[Proxy %s] encerramento forçado por cancelamento de contexto", p.id)
		return
	}

	firstSide := "cliente"
	if first.dir == "remoto -> cliente" {
		firstSide = "remoto"
	}
	reason := formatCloseReason(first.err)

	p.logInfo("[Proxy %s] lado que encerrou primeiro: %s", p.id, firstSide)
	p.logInfo("[Proxy %s] motivo do encerramento: %s", p.id, reason)

	// Se houve erro grave na primeira cópia, força fechar ambos para não travar a segunda cópia
	if first.err != nil && !isEOFOrClosed(first.err) {
		_ = clientConn.Close()
		_ = targetConn.Close()
	}

	// Espera segunda direção finalizar
	var second copyResult
	select {
	case second = <-ch:
	case <-p.ctx.Done():
		p.logInfo("[Proxy %s] encerramento forçado por cancelamento de contexto", p.id)
		return
	}

	if first.dir == "cliente -> remoto" {
		p.logInfo("[Proxy %s] bytes enviados (cliente -> remoto): %d", p.id, first.bytes)
		p.logInfo("[Proxy %s] bytes recebidos (remoto -> cliente): %d", p.id, second.bytes)
	} else {
		p.logInfo("[Proxy %s] bytes recebidos (remoto -> cliente): %d", p.id, first.bytes)
		p.logInfo("[Proxy %s] bytes enviados (cliente -> remoto): %d", p.id, second.bytes)
	}
}

func formatCloseReason(err error) string {
	if err == nil || errors.Is(err, io.EOF) {
		return "EOF (conexão encerrada normalmente)"
	}
	if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
		return "conexão fechada pelo proxy/aplicação"
	}
	return err.Error()
}

func isEOFOrClosed(err error) bool {
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	return strings.Contains(err.Error(), "use of closed network connection")
}
