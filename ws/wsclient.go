package ws

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
	"time"
)

const (
	DISCONNECTED State = iota
	CONNECTED
)

const (
	QUIT Command = 16 + iota
	PING
	USETEXT
	USEBINARY
)

type WsClient struct {
	URL     string
	Headers http.Header
	Input   <-chan []byte
	Output  chan<- []byte
	Status  <-chan Status
	Command chan<- Command
}

type Status struct {
	State State
	Error error
}

func New(url string, headers http.Header) *WsClient {
	inpCh := make(chan []byte, 8)
	outCh := make(chan []byte, 8)
	stsCh := make(chan Status, 2)
	cmdCh := make(chan Command, 2)
	rErrorCh := make(chan error, 1)
	wErrorCh := make(chan error, 1)
	ioEventCh := make(chan bool, 2)
	conCancelCh := make(chan bool, 1)
	controlCh := make(chan Command, 1)
	conReturnCh := make(chan *websocket.Conn, 1)
	var wg sync.WaitGroup
	go func() {
		var reading bool
		var writing bool
		var conn *websocket.Conn
		msgType := websocket.BinaryMessage
		go keepAlive(&wg, ioEventCh, controlCh)
		go connect(&wg, url, headers, stsCh, conReturnCh, ioEventCh, conCancelCh)
		defer safeClose(&wg, conn, conReturnCh, inpCh, outCh, stsCh, cmdCh, controlCh, ioEventCh, conCancelCh, rErrorCh, wErrorCh)
	LOOP:
		for {
			select {
			case conn = <-conReturnCh:
				if conn == nil {
					break LOOP
				}
				reading = true
				writing = true
				go read(&wg, conn, inpCh, ioEventCh, rErrorCh)
				go write(&wg, conn, msgType, outCh, ioEventCh, controlCh, wErrorCh)
			case err := <-rErrorCh:
				reading = false
				if writing {
					controlCh <- QUIT
					stsCh <- Status{State: DISCONNECTED, Error: err}
					continue
				}
				if conn != nil {
					conn.Close()
					conn = nil
				}
				go connect(&wg, url, headers, stsCh, conReturnCh, ioEventCh, conCancelCh)
			case err := <-wErrorCh:
				writing = false
				if reading {
					if conn != nil {
						conn.Close()
						conn = nil
					}
					stsCh <- Status{State: DISCONNECTED, Error: err}
					continue
				}
				go connect(&wg, url, headers, stsCh, conReturnCh, ioEventCh, conCancelCh)
			case cmd, ok := <-cmdCh:
				switch {
				case !ok || cmd == QUIT:
					if reading || writing || conn != nil {
						stsCh <- Status{State: DISCONNECTED}
					}
					break LOOP
				case cmd == PING:
					if conn != nil && writing {
						controlCh <- cmd
					}
				case cmd == USETEXT:
					msgType = websocket.TextMessage
					if writing {
						controlCh <- cmd
					}
				case cmd == USEBINARY:
					msgType = websocket.BinaryMessage
					if writing {
						controlCh <- cmd
					}
				}
			}
		}
	}()
	return &WsClient{URL: url, Headers: headers, Input: inpCh, Output: outCh, Status: stsCh, Command: cmdCh}
}

func connect(wg *sync.WaitGroup, url string, headers http.Header,
	stsCh chan Status, conReturnCh chan *websocket.Conn, ioEventCh, conCancelCh chan bool) {
	wg.Add(1)
	defer wg.Done()
	for {
		dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
		conn, _, err := dialer.Dial(url, headers)
		if err == nil {
			conn.SetPongHandler(func(string) error { ioEventCh <- true; return nil })
			conReturnCh <- conn
			stsCh <- Status{State: CONNECTED}
			return
		}
		stsCh <- Status{State: DISCONNECTED, Error: err}
		select {
		case <-time.After(30 * time.Second):
		case <-conCancelCh:
			stsCh <- Status{State: DISCONNECTED, Error: errors.New("cancelled")}
			return
		}
	}
}

func keepAlive(wg *sync.WaitGroup,
	ioEventCh chan bool, controlCh chan Command) {
	wg.Add(1)
	defer wg.Done()
	dur := 30 * time.Second
	timer := time.NewTimer(dur)
	timer.Stop()
LOOP:
	for {
		select {
		case _, ok := <-ioEventCh:
			if !ok {
				timer.Stop()
				break LOOP
			}
			timer.Reset(dur)
		case <-timer.C:
			timer.Reset(dur)
			select {
			case controlCh <- PING:
			default:
			}
		}
	}
}

func write(wg *sync.WaitGroup, conn *websocket.Conn, msgType int,
	outCh chan []byte, ioEventCh chan bool, controlCh chan Command, wErrorCh chan error) {
	wg.Add(1)
	defer wg.Done()
LOOP:
	for {
		select {
		case msg, ok := <-outCh:
			if !ok {
				wErrorCh <- errors.New("outCh closed")
				break LOOP
			}
			ioEventCh <- true
			if err := conn.SetWriteDeadline(time.Now().Add(3 * time.Second)); err != nil {
				wErrorCh <- err
				break LOOP
			}
			if err := conn.WriteMessage(msgType, msg); err != nil {
				wErrorCh <- err
				break LOOP
			}
			conn.SetWriteDeadline(time.Time{})
		case cmd, ok := <-controlCh:
			if !ok {
				wErrorCh <- errors.New("controlCh closed")
				break LOOP
			}
			switch cmd {
			case QUIT:
				wErrorCh <- errors.New("cancelled")
				break LOOP
			case PING:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(3*time.Second)); err != nil {
					wErrorCh <- errors.New("cancelled")
					break LOOP
				}
			case USETEXT:
				msgType = websocket.TextMessage
			case USEBINARY:
				msgType = websocket.BinaryMessage
			}
		}
	}
}

func read(wg *sync.WaitGroup, conn *websocket.Conn,
	inpCh chan []byte, ioEventCh chan bool, rErrorCh chan error) {
	wg.Add(1)
	defer wg.Done()
	for {
		if _, msg, err := conn.ReadMessage(); err == nil {
			ioEventCh <- true
			inpCh <- msg
		} else {
			rErrorCh <- err
			break
		}
	}
}

func safeClose(wg *sync.WaitGroup, conn *websocket.Conn,
	conReturnCh chan *websocket.Conn, inpCh, outCh chan []byte, stsCh chan Status, cmdCh, controlCh chan Command,
	ioEventCh, conCancelCh chan bool, rErrorCh, wErrorCh chan error) {
	if conn != nil {
		conn.Close()
	}
	close(ioEventCh)
	close(controlCh)
	close(conCancelCh)
	<-time.After(50 * time.Millisecond)
LOOP:
	for {
		select {
		case _, ok := <-outCh:
			if !ok {
				outCh = nil
			}
		case _, ok := <-cmdCh:
			if !ok {
				inpCh = nil
			}
		case conn, ok := <-conReturnCh:
			if conn != nil {
				conn.Close()
			}
			if !ok {
				conReturnCh = nil
			}
		case _, ok := <-rErrorCh:
			if !ok {
				rErrorCh = nil
			}
		case _, ok := <-wErrorCh:
			if !ok {
				wErrorCh = nil
			}
		default:
			break LOOP
		}
	}
	wg.Wait()
	close(inpCh)
	close(stsCh)
}

type State byte

type Command byte

func (s State) String() string {
	switch s {
	case DISCONNECTED:
		return "DISCONNECTED"
	case CONNECTED:
		return "CONNECTED"
	}
	return fmt.Sprintf("UNKNOWN STATUS %s", s)
}

func (c Command) String() string {
	switch c {
	case QUIT:
		return "QUIT"
	case PING:
		return "PING"
	case USETEXT:
		return "USE_TEXT"
	case USEBINARY:
		return "USE_BINARY"
	}
	return fmt.Sprintf("UNKNOWN COMMAND %s", c)
}
