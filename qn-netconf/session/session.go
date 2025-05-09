package session

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"golang.org/x/crypto/ssh"

	"qn-netconf/config" // For appConfig access
	"qn-netconf/handlers"
	"qn-netconf/rpcrouter"
	"qn-netconf/utils"
)

func HandleConnection(netConn net.Conn, sshCfg *ssh.ServerConfig, appCfg *config.Config) {
	defer netConn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), appCfg.ConnectionTimeout)
	defer cancel()

	sshConn, chans, reqs, err := ssh.NewServerConn(netConn, sshCfg)
	if err != nil {
		log.Printf("NETCONF_SESSION: SSH handshake failed for %s: %v", netConn.RemoteAddr(), err)
		return
	}
	defer sshConn.Close()

	log.Printf("NETCONF_SESSION: New SSH connection: %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			log.Printf("NETCONF_SESSION: Rejected channel type %s from %s", newChannel.ChannelType(), sshConn.RemoteAddr())
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("NETCONF_SESSION: Could not accept channel from %s: %v", sshConn.RemoteAddr(), err)
			continue
		}
		log.Printf("NETCONF_SESSION: Accepted session channel from %s", sshConn.RemoteAddr())
		go handleNETCONFSession(ctx, channel, requests, appCfg)
	}
}

func handleNETCONFSession(ctx context.Context, channel ssh.Channel, reqs <-chan *ssh.Request, appCfg *config.Config) {
	defer channel.Close()
	sessionID := utils.GenerateSessionID()

	subsysChan := make(chan bool, 1)
	go func() {
		for req := range reqs {
			switch req.Type {
			case "subsystem":
				if strings.TrimSpace(string(req.Payload[4:])) == "netconf" {
					req.Reply(true, nil)
					subsysChan <- true
					return
				}
			}
			req.Reply(false, nil)
		}
		subsysChan <- false
	}()

	select {
	case success := <-subsysChan:
		if !success {
			log.Printf("NETCONF_SESSION: Client (session %s) didn't request netconf subsystem or request failed.", sessionID)
			return
		}
	case <-ctx.Done():
		log.Printf("NETCONF_SESSION: Subsystem request timed out for session %s: %v", sessionID, ctx.Err())
		return
	}
	log.Printf("NETCONF_SESSION: NETCONF subsystem established for session %s", sessionID)

	if err := handleNETCONFCommunication(channel, sessionID, appCfg); err != nil {
		log.Printf("NETCONF_SESSION: Communication error for session %s: %v", sessionID, err)
	}
}

func handleNETCONFCommunication(channel ssh.Channel, sessionID string, appCfg *config.Config) error {
	serverHello := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><capabilities><capability>urn:ietf:params:netconf:base:1.0</capability><capability>%s</capability><capability>%s</capability></capabilities><session-id>%s</session-id></hello>%s`,
		handlers.VlanNamespace, handlers.InterfaceNamespace, sessionID, appCfg.FrameEnd)

	if _, err := channel.Write([]byte(serverHello)); err != nil {
		return fmt.Errorf("failed to send server hello: %w", err)
	}

	clientHello, err := ReadFrame(channel, appCfg.FrameEnd)
	if err != nil {
		return fmt.Errorf("error reading client hello: %w", err)
	}
	log.Printf("NETCONF_SESSION: Session %s: Client hello received:\n%s", sessionID, clientHello)

	for {
		request, err := ReadFrame(channel, appCfg.FrameEnd)
		if err != nil {
			if err == io.EOF {
				log.Printf("NETCONF_SESSION: Session %s: Client closed connection gracefully.", sessionID)
				return nil
			}
			return fmt.Errorf("error reading RPC request: %w", err)
		}

		response := rpcrouter.DispatchRequest(appCfg.MiyagiSocketPath, appCfg.FrameEnd, request)
		if _, err := channel.Write(response); err != nil {
			return fmt.Errorf("failed to send RPC response: %w", err)
		}
	}
}

func ReadFrame(channel ssh.Channel, frameEnd string) ([]byte, error) {
	var buffer bytes.Buffer
	frameEndBytes := []byte(frameEnd)
	readBuf := make([]byte, 4096)

	for {
		n, err := channel.Read(readBuf)
		if err != nil {
			return nil, err
		}

		buffer.Write(readBuf[:n])

		if terminatorIndex := bytes.Index(buffer.Bytes(), frameEndBytes); terminatorIndex != -1 {
			message := buffer.Bytes()[:terminatorIndex]
			remainingData := buffer.Bytes()[terminatorIndex+len(frameEndBytes):]
			buffer.Reset()
			if len(remainingData) > 0 {
				buffer.Write(remainingData)
			}
			return bytes.TrimSpace(message), nil
		}
	}
}
