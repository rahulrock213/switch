package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net" // Standard library
	"os"
	"os/signal"
	"strings"
	"sync/atomic"

	"qn-netconf/handlers"
	"qn-netconf/utils" // Import the new utils package

	"golang.org/x/crypto/ssh"
)

var (
	sessionCounter uint32 = 1000
	appConfig      *Config
)

// RPCHandler defines the function signature for NETCONF RPC handlers.
type RPCHandler func(miyagiSocketPath, frameEnd string, request []byte, msgID string) []byte

var rpcHandlers map[string]RPCHandler

func init() {
	// Initialize RPC handlers map
	rpcHandlers = map[string]RPCHandler{
		// "get-vlans" was a custom RPC. Standard <get> with filter is now handled directly in generateResponse.
		// If you still need a custom <get-vlans> operation for other purposes, you can uncomment and keep this.
		// "get-vlans": func(miyagiSocketPath, frameEnd string, request []byte, msgID string) []byte {
		// 	return handlers.BuildGetVlansResponse(miyagiSocketPath, msgID, frameEnd)
		// }, // Keep this comma if you uncomment the above
		"edit-config": func(miyagiSocketPath, frameEnd string, request []byte, msgID string) []byte {
			// More specific dispatch for edit-config can be done here if needed
			// Check for the short VLAN namespace first
			if bytes.Contains(request, []byte("<vlans xmlns=\"yang:set_vlan\">")) {
				log.Printf("NETCONF_SERVER: Dispatching to HandleEditConfig for VLANs with custom 'yang:set_vlan' namespace. Message ID: %s", msgID)
				return handlers.HandleEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			} else if bytes.Contains(request, []byte(fmt.Sprintf("<vlans xmlns=\"%s\">", handlers.VlanNamespace))) {
				// Fallback to original namespace for compatibility / other tools
				log.Printf("NETCONF_SERVER: Dispatching to HandleEditConfig for VLANs with original namespace. Message ID: %s", msgID)
				return handlers.HandleEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			}
			// SSH checks
			sshCheckString := "<ssh xmlns=\"yang:set_ssh\">"
			containsSshShort := bytes.Contains(request, []byte(sshCheckString))
			// Safely log a snippet of the request
			requestSnippetForLog := string(request)
			if len(requestSnippetForLog) > 512 {
				requestSnippetForLog = requestSnippetForLog[:512] + "..."
			}
			// Log the request being checked and the result of bytes.Contains
			log.Printf("NETCONF_SERVER: DEBUG: In edit-config handler. Checking for substring '%s'. Found: %t. Full request snippet: %s", sshCheckString, containsSshShort, requestSnippetForLog)
			if containsSshShort {
				log.Printf("NETCONF_SERVER: Dispatching to HandleSSHEditConfig with custom 'yang:set_ssh' namespace. Message ID: %s", msgID)
				return handlers.HandleSSHEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			} else if bytes.Contains(request, []byte(fmt.Sprintf("<ssh xmlns=\"%s\">", handlers.SshConfigNamespace))) {
				log.Printf("NETCONF_SERVER: Dispatching to HandleSSHEditConfig with original namespace. Message ID: %s", msgID)
				return handlers.HandleSSHEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			} else if bytes.Contains(request, []byte("<telnet xmlns=\"yang:set_telnet\">")) {
				log.Printf("NETCONF_SERVER: Dispatching to HandleTelnetEditConfig with custom 'yang:set_telnet' namespace. Message ID: %s", msgID)
				return handlers.HandleTelnetEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			} else if bytes.Contains(request, []byte(fmt.Sprintf("<telnet xmlns=\"%s\">", handlers.TelnetConfigNamespace))) {
				return handlers.HandleTelnetEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			} else if bytes.Contains(request, []byte(fmt.Sprintf("<routing xmlns=\"%s\">", handlers.RoutingNamespace))) {
				return handlers.HandleRouteEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			} else if bytes.Contains(request, []byte("<routing xmlns=\"yang:set_route\">")) {
				log.Printf("NETCONF_SERVER: Dispatching to HandleRouteEditConfig with custom 'yang:set_route' namespace. Message ID: %s", msgID)
				return handlers.HandleRouteEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			} else if bytes.Contains(request, []byte("<ip-interfaces xmlns=\"yang:set_ip_interface\">")) {
				log.Printf("NETCONF_SERVER: Dispatching to HandleIpInterfaceEditConfig with custom 'yang:set_ip_interface' namespace. Message ID: %s", msgID)
				return handlers.HandleIpInterfaceEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			} else if bytes.Contains(request, []byte(fmt.Sprintf("<ip-interfaces xmlns=\"%s\">", handlers.IpInterfaceNamespace))) {
				return handlers.HandleIpInterfaceEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			} else if bytes.Contains(request, []byte(fmt.Sprintf("<port-configurations xmlns=\"%s\">", handlers.PortConfigNamespace))) {
				log.Printf("NETCONF_SERVER: Dispatching to HandlePortConfigurationEditConfig with original namespace. Message ID: %s", msgID)
				return handlers.HandlePortConfigurationEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			} else if bytes.Contains(request, []byte("<port-configurations xmlns=\"yang:set_port_config\">")) {
				log.Printf("NETCONF_SERVER: Dispatching to HandlePortConfigurationEditConfig with custom 'yang:set_port_config' namespace. Message ID: %s", msgID)
				return handlers.HandlePortConfigurationEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			} else if bytes.Contains(request, []byte("<port-channels xmlns=\"yang:set_port_channel\">")) {
				log.Printf("NETCONF_SERVER: Dispatching to HandleLagEditConfig with custom 'yang:set_port_channel' namespace. Message ID: %s", msgID)
				return handlers.HandlePortConfigurationEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			} else if bytes.Contains(request, []byte("<stp-global-config xmlns=\"yang:set_stp\">")) {
				log.Printf("NETCONF_SERVER: Dispatching to HandleStpEditConfig with custom 'yang:set_stp' namespace. Message ID: %s", msgID)
				return handlers.HandleStpEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			} else if bytes.Contains(request, []byte(fmt.Sprintf("<stp-global-config xmlns=\"%s\">", handlers.StpGlobalConfigNamespace))) {
				return handlers.HandleStpEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			}

			// Add other edit-config handlers based on content/namespace
			log.Printf("NETCONF_SERVER: Received <edit-config> for unknown model or malformed VLAN config: %s", string(request))
			return buildErrorResponse(frameEnd, msgID, "operation-failed", "Unsupported configuration target in edit-config")
		},
		// Example for a new handler (see Step 5)
		// "get-interfaces": handleGetInterfaces,
	}
}

func main() {
	var err error
	// in main()
	appConfig, err = LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	config := &ssh.ServerConfig{
		PasswordCallback: passwordCallback,
		ServerVersion:    appConfig.ServerBanner,
		MaxAuthTries:     3,
		AuthLogCallback:  authLogCallback,
	}

	if err := loadHostKey(config, appConfig.HostKeyPath); err != nil {
		log.Fatalf("NETCONF_SERVER: Failed to load host key: %v", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", appConfig.SSHPort))
	if err != nil {
		log.Fatalf("NETCONF_SERVER: Failed to listen on port %d: %v", appConfig.SSHPort, err)
	}
	defer listener.Close()

	log.Printf("NETCONF_SERVER: Listening on port %d", appConfig.SSHPort)

	// Goroutine for handling connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// Check if the error is due to the listener being closed
				if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
					log.Println("NETCONF_SERVER: Listener closed, shutting down accept loop.")
					return
				}
				log.Printf("NETCONF_SERVER: Accept error: %v", err)
				continue
			}
			go handleConnection(conn, config)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill) // syscall.SIGINT, syscall.SIGTERM
	<-quit
	log.Println("NETCONF_SERVER: Shutting down...")

	// Close the listener to stop accepting new connections
	if err := listener.Close(); err != nil {
		log.Printf("NETCONF_SERVER: Error closing listener: %v", err)
	}

	// Add any other cleanup logic here (e.g., wait for active connections with a timeout)
	log.Println("NETCONF_SERVER: Shutdown complete.")
}

func loadHostKey(config *ssh.ServerConfig, path string) error {
	privateKeyBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read key file %s: %w", path, err)
	}

	privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return fmt.Errorf("parse private key from %s: %w", path, err)
	}

	config.AddHostKey(privateKey)
	return nil
}

func passwordCallback(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	// Use the new SNMP-based authentication
	authenticated, err := utils.ValidateCredentials(conn.User(), string(password))
	if err != nil {
		// Log the internal error but return a generic auth failed to the client
		log.Printf("NETCONF_SERVER: Authentication internal error for user %s: %v", conn.User(), err)
		return nil, fmt.Errorf("authentication failed")
	}
	if authenticated {
		return nil, nil
	}
	return nil, fmt.Errorf("authentication failed for user %s", conn.User())
}

func authLogCallback(conn ssh.ConnMetadata, method string, err error) {
	status := "FAILED"
	if err == nil {
		status = "SUCCESS"
	}
	log.Printf("NETCONF_SERVER: Auth attempt: user=%s method=%s status=%s remote=%s", conn.User(), method, status, conn.RemoteAddr())
}

func handleConnection(netConn net.Conn, config *ssh.ServerConfig) {
	defer netConn.Close()
	// Use configured connection timeout
	ctx, cancel := context.WithTimeout(context.Background(), appConfig.ConnectionTimeout)
	defer cancel()

	sshConn, chans, reqs, err := ssh.NewServerConn(netConn, config)
	if err != nil {
		log.Printf("NETCONF_SERVER: SSH handshake failed for %s: %v", netConn.RemoteAddr(), err)
		return
	}
	defer sshConn.Close()

	log.Printf("NETCONF_SERVER: New SSH connection: %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			log.Printf("NETCONF_SERVER: Rejected channel type %s from %s", newChannel.ChannelType(), sshConn.RemoteAddr())
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("NETCONF_SERVER: Could not accept channel from %s: %v", sshConn.RemoteAddr(), err)
			continue
		}
		log.Printf("NETCONF_SERVER: Accepted session channel from %s", sshConn.RemoteAddr())
		go handleNETCONFSession(ctx, channel, requests)
	}
}

func handleNETCONFSession(ctx context.Context, channel ssh.Channel, reqs <-chan *ssh.Request) {
	defer channel.Close()
	sessionID := generateSessionID()

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
			log.Printf("NETCONF_SERVER: Client (session %s) didn't request netconf subsystem or request failed.", sessionID)
			return
		}
	case <-ctx.Done():
		log.Printf("NETCONF_SERVER: Subsystem request timed out for session %s: %v", sessionID, ctx.Err())
		return
	}
	log.Printf("NETCONF_SERVER: NETCONF subsystem established for session %s", sessionID)

	if err := handleNETCONFCommunication(channel, sessionID); err != nil {
		log.Printf("NETCONF_SERVER: Communication error for session %s: %v", sessionID, err)
	}
}

func handleNETCONFCommunication(channel ssh.Channel, sessionID string) error {
	serverHello := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.0</capability>
    <capability>%s</capability> <!-- VLAN Capability -->
    <capability>%s</capability> <!-- Interface Capability -->
    <capability>%s</capability> <!-- SSH Server Config Capability -->
    <capability>%s</capability> <!-- Telnet Server Config Capability -->    
    <capability>%s</capability> <!-- IP Interface Capability -->
    <capability>%s</capability> <!-- Port Configuration Capability -->
    <capability>%s</capability> <!-- STP Global Configuration Capability -->
    <capability>%s</capability> <!-- Port Status Capability -->
    <capability>%s</capability> <!-- Port Description Capability -->
    <capability>%s</capability> <!-- Port Speed Capability -->    
  </capabilities>
  <session-id>%s</session-id>
</hello>
%s`,
		handlers.VlanNamespace,
		handlers.InterfaceNamespace,
		handlers.SshConfigNamespace,
		handlers.TelnetConfigNamespace,
		handlers.IpInterfaceNamespace,
		handlers.PortConfigNamespace,
		handlers.StpGlobalConfigNamespace,
		handlers.PortStatusCapability,      // Add Port Status Capability
		handlers.PortDescriptionCapability, // Add Port Description Capability
		handlers.PortSpeedCapability,       // Add Port Speed Capability
		sessionID,
		appConfig.FrameEnd)
	// Added handlers.StpGlobalConfigNamespace to advertise STP Global Configuration capability

	if _, err := channel.Write([]byte(serverHello)); err != nil {
		return fmt.Errorf("failed to send server hello: %w", err) // Corrected format specifier count
	}

	// Read client's hello message.
	clientHello, err := readFrame(channel)
	if err != nil {
		return fmt.Errorf("error reading client hello: %w", err)
	}
	// If the first frame from client looks like an RPC, try to process it immediately.
	// A proper client hello would also be an XML document, but we are being lenient here.
	if bytes.HasPrefix(clientHello, []byte("<rpc")) {
		log.Printf("NETCONF_SERVER: Session %s: First client frame looks like RPC, attempting to process:\n%s", sessionID, clientHello)
		response := generateResponse(clientHello)
		if _, err := channel.Write(response); err != nil {
			return fmt.Errorf("failed to send response to initial client RPC: %w", err)
		}
	}
	log.Printf("NETCONF_SERVER: Session %s: Client hello received:\n%s", sessionID, clientHello)

	for {
		request, err := readFrame(channel)
		if err != nil {
			if err == io.EOF {
				log.Printf("NETCONF_SERVER: Session %s: Client closed connection gracefully.", sessionID)
				return nil
			}
			return fmt.Errorf("error reading RPC request: %w", err)
		}

		response := generateResponse(request)
		if _, err := channel.Write(response); err != nil {
			return fmt.Errorf("failed to send RPC response: %w", err)
		}
	}
}

func generateSessionID() string {
	return fmt.Sprintf("%d", atomic.AddUint32(&sessionCounter, 1))
}

// generateResponse dispatches RPC requests to appropriate handlers.
func generateResponse(request []byte) []byte {
	msgID := extractMessageID(request)

	// Attempt to find the actual start of the <rpc> tag within the received frame
	// This will help ignore any leading garbage text like "asdasd"
	rpcStartIndex := bytes.Index(request, []byte("<rpc"))
	var rpcQuery []byte
	if rpcStartIndex != -1 {
		// If <rpc is found, consider the message from that point
		rpcQuery = request[rpcStartIndex:]
	} else {
		// No <rpc> tag found at all in the frame, definitely malformed
		log.Printf("NETCONF_SERVER: Malformed request, no <rpc> tag found in frame: %s", string(request))
		return buildErrorResponse(appConfig.FrameEnd, msgID, "malformed-message", "Request frame does not contain an <rpc> tag.")
	}

	// Check for standard <get> operation
	if bytes.HasPrefix(rpcQuery, []byte("<rpc")) && bytes.Contains(rpcQuery, []byte("<get>")) && bytes.Contains(rpcQuery, []byte("</get>")) {
		// It's a <get> operation. Check for specific filters.

		// VLAN checks
		if bytes.Contains(request, []byte("<vlans")) &&
			(bytes.Contains(request, []byte("xmlns=\"yang:vlan\"")) || bytes.Contains(request, []byte("xmlns='yang:vlan'"))) { // User requested "yang:vlan" for get
			log.Printf("NETCONF_SERVER: Dispatching to BuildGetVlansResponse for <get> with custom 'yang:vlan' namespace. Message ID: %s", msgID)
			return handlers.BuildGetVlansResponse(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<vlans xmlns=\"%s\"", handlers.VlanNamespace))) {
			// Fallback for original namespace / other tools
			log.Printf("NETCONF_SERVER: Dispatching to BuildGetVlansResponse for <get> with original VLAN filter. Message ID: %s", msgID)
			return handlers.BuildGetVlansResponse(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
			// Interface checks
		} else if bytes.Contains(request, []byte("<interfaces")) &&
			(bytes.Contains(request, []byte("xmlns=\"yang:get_interface\"")) || bytes.Contains(request, []byte("xmlns='yang:get_interface'"))) {
			log.Printf("NETCONF_SERVER: Dispatching to BuildGetInterfacesResponse for <get> with custom 'yang:get_interface' namespace. Message ID: %s", msgID)
			return handlers.BuildGetInterfacesResponse(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<interfaces xmlns=\"%s\"", handlers.InterfaceNamespace))) {
			log.Printf("NETCONF_SERVER: Dispatching to BuildGetInterfacesResponse for <get> with Interface filter. Message ID: %s", msgID)
			return handlers.BuildGetInterfacesResponse(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
			// SSH checks
		} else if bytes.Contains(request, []byte("<ssh")) &&
			(bytes.Contains(request, []byte("xmlns=\"yang:get_ssh\"")) || bytes.Contains(request, []byte("xmlns='yang:get_ssh'"))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleSSHGetConfig for <get> with custom 'yang:get_ssh' namespace. Message ID: %s", msgID)
			return handlers.HandleSSHGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<ssh xmlns=\"%s\"", handlers.SshConfigNamespace))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleSSHGetConfig for <get> with SSH filter. Message ID: %s", msgID)
			return handlers.HandleSSHGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte("<telnet")) &&
			(bytes.Contains(request, []byte("xmlns=\"yang:get_telnet\"")) || bytes.Contains(request, []byte("xmlns='yang:get_telnet'"))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleTelnetGetConfig for <get> with custom 'yang:get_telnet' namespace. Message ID: %s", msgID)
			return handlers.HandleTelnetGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<telnet xmlns=\"%s\"", handlers.TelnetConfigNamespace))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleTelnetGetConfig for <get> with Telnet filter. Message ID: %s", msgID)
			return handlers.HandleTelnetGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
			// IP Interface checks for <get>
		} else if bytes.Contains(request, []byte("<ip-interfaces")) &&
			(bytes.Contains(request, []byte("xmlns=\"yang:get_ip_interface\"")) || bytes.Contains(request, []byte("xmlns='yang:get_ip_interface'"))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleIpInterfaceGetConfig for <get> with custom 'yang:get_ip_interface' namespace. Message ID: %s", msgID)
			return handlers.HandleIpInterfaceGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<ip-interfaces xmlns=\"%s\"", handlers.IpInterfaceNamespace))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleIpInterfaceGetConfig for <get> with original IP Interface filter. Message ID: %s", msgID)
			return handlers.HandleIpInterfaceGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
			// Port Channel checks for <get>
		} else if bytes.Contains(request, []byte("<port-channels")) &&
			(bytes.Contains(request, []byte("xmlns=\"yang:get_port_channel\"")) || bytes.Contains(request, []byte("xmlns='yang:get_port_channel'"))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleLagGetConfig for <get> with custom 'yang:get_port_channel' namespace. Message ID: %s", msgID)
			return handlers.HandleLagGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte("<port-configurations")) && // Physical port get (short namespace)
			(bytes.Contains(request, []byte("xmlns=\"yang:get_port_config\"")) || bytes.Contains(request, []byte("xmlns='yang:get_port_config'"))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandlePortConfigurationGetConfig for <get> with custom 'yang:get_port_config' namespace. Message ID: %s", msgID)
			return handlers.HandlePortConfigurationGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<port-configurations xmlns=\"%s\"", handlers.PortConfigNamespace))) { // Physical port get (original namespace)
			log.Printf("NETCONF_SERVER: Dispatching to HandlePortConfigurationGetConfig for <get> with Port Configuration filter. Message ID: %s", msgID)
			return handlers.HandlePortConfigurationGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
			// STP Global checks for <get>
		} else if bytes.Contains(request, []byte("<stp-global-config")) &&
			(bytes.Contains(request, []byte("xmlns=\"yang:get_stp\"")) || bytes.Contains(request, []byte("xmlns='yang:get_stp'"))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleStpGetConfig for <get> with custom 'yang:get_stp' namespace. Message ID: %s", msgID)
			return handlers.HandleStpGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<stp-global-config xmlns=\"%s\"", handlers.StpGlobalConfigNamespace))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleStpGetConfig for <get> with original STP Global filter. Message ID: %s", msgID)
			return handlers.HandleStpGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
			// Port Status checks for <get>
		} else if bytes.Contains(request, []byte("<port-status")) &&
			(bytes.Contains(request, []byte(fmt.Sprintf("xmlns=\"%s\"", handlers.PortStatusNamespace))) || bytes.Contains(request, []byte(fmt.Sprintf("xmlns='%s'", handlers.PortStatusNamespace)))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleGetPortStatus for <get> with Port Status filter. Message ID: %s", msgID)
			return handlers.HandleGetPortStatus(appConfig.MiyagiSocketPath, request, msgID, appConfig.FrameEnd)
			// Port Description checks for <get>
		} else if bytes.Contains(request, []byte("<port-description")) &&
			(bytes.Contains(request, []byte(fmt.Sprintf("xmlns=\"%s\"", handlers.PortDescriptionNamespace))) || bytes.Contains(request, []byte(fmt.Sprintf("xmlns='%s'", handlers.PortDescriptionNamespace)))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleGetPortDescription for <get> with Port Description filter. Message ID: %s", msgID)
			return handlers.HandleGetPortDescription(appConfig.MiyagiSocketPath, request, msgID, appConfig.FrameEnd)
			// Port Speed checks for <get>
		} else if bytes.Contains(request, []byte("<port-speed")) &&
			(bytes.Contains(request, []byte(fmt.Sprintf("xmlns=\"%s\"", handlers.PortSpeedNamespace))) || bytes.Contains(request, []byte(fmt.Sprintf("xmlns='%s'", handlers.PortSpeedNamespace)))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleGetPortSpeed for <get> with Port Speed filter. Message ID: %s", msgID)
			return handlers.HandleGetPortSpeed(appConfig.MiyagiSocketPath, request, msgID, appConfig.FrameEnd)

		}
		// If it's a <get> but not for VLANs as per the filter above, it's unhandled by this specific logic.
		log.Printf("NETCONF_SERVER: Received <get> operation with an unhandled filter. Message ID: %s. Request: %s", msgID, string(request))
		return buildErrorResponse(appConfig.FrameEnd, msgID, "operation-not-supported", "The <get> operation with the specified filter is not supported.")

	} else if bytes.HasPrefix(rpcQuery, []byte("<rpc")) && bytes.Contains(rpcQuery, []byte("<get-config>")) && bytes.Contains(rpcQuery, []byte("</get-config>")) {
		// It's a <get-config> operation. Check for specific filters.

		// VLAN checks
		if bytes.Contains(request, []byte("<vlans")) &&
			(bytes.Contains(request, []byte("xmlns=\"yang:vlan\"")) || bytes.Contains(request, []byte("xmlns='yang:vlan'"))) { // User requested "yang:vlan" for get
			log.Printf("NETCONF_SERVER: Dispatching to BuildGetVlansResponse for <get-config> with custom 'yang:vlan' namespace. Message ID: %s", msgID)
			return handlers.BuildGetVlansResponse(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<vlans xmlns=\"%s\"", handlers.VlanNamespace))) {
			// Fallback for original namespace / other tools
			log.Printf("NETCONF_SERVER: Dispatching to BuildGetVlansResponse for <get-config> with original VLAN filter. Message ID: %s", msgID)
			return handlers.BuildGetVlansResponse(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
			// Interface checks
		} else if bytes.Contains(request, []byte("<interfaces")) &&
			(bytes.Contains(request, []byte("xmlns=\"yang:get_interface\"")) || bytes.Contains(request, []byte("xmlns='yang:get_interface'"))) {
			log.Printf("NETCONF_SERVER: Dispatching to BuildGetInterfacesResponse for <get-config> with custom 'yang:get_interface' namespace. Message ID: %s", msgID)
			return handlers.BuildGetInterfacesResponse(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<interfaces xmlns=\"%s\"", handlers.InterfaceNamespace))) {
			log.Printf("NETCONF_SERVER: Dispatching to BuildGetInterfacesResponse for <get-config> with Interface filter. Message ID: %s", msgID)
			return handlers.BuildGetInterfacesResponse(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
			// SSH checks
		} else if bytes.Contains(request, []byte("<ssh")) &&
			(bytes.Contains(request, []byte("xmlns=\"yang:get_ssh\"")) || bytes.Contains(request, []byte("xmlns='yang:get_ssh'"))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleSSHGetConfig for <get-config> with custom 'yang:get_ssh' namespace. Message ID: %s", msgID)
			return handlers.HandleSSHGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<ssh xmlns=\"%s\"", handlers.SshConfigNamespace))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleSSHGetConfig for <get-config> with SSH filter. Message ID: %s", msgID)
			return handlers.HandleSSHGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
			// Telnet checks
		} else if bytes.Contains(request, []byte("<telnet")) &&
			(bytes.Contains(request, []byte("xmlns=\"yang:get_telnet\"")) || bytes.Contains(request, []byte("xmlns='yang:get_telnet'"))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleTelnetGetConfig for <get-config> with custom 'yang:get_telnet' namespace. Message ID: %s", msgID)
			return handlers.HandleTelnetGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<telnet xmlns=\"%s\"", handlers.TelnetConfigNamespace))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleTelnetGetConfig for <get-config> with Telnet filter. Message ID: %s", msgID)
			return handlers.HandleTelnetGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte("<port-channels")) && // Port Channel get-config (short namespace)
			(bytes.Contains(request, []byte("xmlns=\"yang:get_port_channel\"")) || bytes.Contains(request, []byte("xmlns='yang:get_port_channel'"))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleLagGetConfig for <get-config> with custom 'yang:get_port_channel' namespace. Message ID: %s", msgID)
			return handlers.HandleLagGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte("<port-configurations")) && // Physical port get-config (short namespace)
			(bytes.Contains(request, []byte("xmlns=\"yang:get_port_config\"")) || bytes.Contains(request, []byte("xmlns='yang:get_port_config'"))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandlePortConfigurationGetConfig for <get-config> with custom 'yang:get_port_config' namespace. Message ID: %s", msgID)
			return handlers.HandlePortConfigurationGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<ip-interfaces xmlns=\"%s\"", handlers.IpInterfaceNamespace))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleIpInterfaceGetConfig for <get-config> with IP Interface filter. Message ID: %s", msgID)
			return handlers.HandleIpInterfaceGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<port-configurations xmlns=\"%s\"", handlers.PortConfigNamespace))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandlePortConfigurationGetConfig for <get-config> with Port Configuration filter. Message ID: %s", msgID)
			return handlers.HandlePortConfigurationGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<stp-global-config xmlns=\"%s\"", handlers.StpGlobalConfigNamespace))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleStpGetConfig for <get-config> with STP Global filter. Message ID: %s", msgID)
			return handlers.HandleStpGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
			// IP Interface checks for <get-config> (short namespace)
		} else if bytes.Contains(request, []byte("<ip-interfaces")) &&
			(bytes.Contains(request, []byte("xmlns=\"yang:get_ip_interface\"")) || bytes.Contains(request, []byte("xmlns='yang:get_ip_interface'"))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleIpInterfaceGetConfig for <get-config> with custom 'yang:get_ip_interface' namespace. Message ID: %s", msgID)
			return handlers.HandleIpInterfaceGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd) // This was a typo, should be HandleIpInterfaceGetConfig
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<port-configurations xmlns=\"%s\"", handlers.PortConfigNamespace))) { // Physical port get-config (original namespace)
			log.Printf("NETCONF_SERVER: Dispatching to HandlePortConfigurationGetConfig for <get-config> with Port Configuration filter. Message ID: %s", msgID)
			return handlers.HandlePortConfigurationGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd) // Corrected this line
			// STP Global checks for <get-config>
		} else if bytes.Contains(request, []byte("<stp-global-config")) &&
			(bytes.Contains(request, []byte("xmlns=\"yang:get_stp\"")) || bytes.Contains(request, []byte("xmlns='yang:get_stp'"))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleStpGetConfig for <get-config> with custom 'yang:get_stp' namespace. Message ID: %s", msgID)
			return handlers.HandleStpGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
		} else if bytes.Contains(request, []byte(fmt.Sprintf("<stp-global-config xmlns=\"%s\"", handlers.StpGlobalConfigNamespace))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleStpGetConfig for <get-config> with original STP Global filter. Message ID: %s", msgID)
			return handlers.HandleStpGetConfig(appConfig.MiyagiSocketPath, msgID, appConfig.FrameEnd)
			// Port Status checks for <get-config>
		} else if bytes.Contains(request, []byte("<port-status")) &&
			(bytes.Contains(request, []byte(fmt.Sprintf("xmlns=\"%s\"", handlers.PortStatusNamespace))) || bytes.Contains(request, []byte(fmt.Sprintf("xmlns='%s'", handlers.PortStatusNamespace)))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleGetPortStatus for <get-config> with Port Status filter. Message ID: %s", msgID)
			return handlers.HandleGetPortStatus(appConfig.MiyagiSocketPath, request, msgID, appConfig.FrameEnd)
			// Port Description checks for <get-config>
		} else if bytes.Contains(request, []byte("<port-description")) &&
			(bytes.Contains(request, []byte(fmt.Sprintf("xmlns=\"%s\"", handlers.PortDescriptionNamespace))) || bytes.Contains(request, []byte(fmt.Sprintf("xmlns='%s'", handlers.PortDescriptionNamespace)))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleGetPortDescription for <get-config> with Port Description filter. Message ID: %s", msgID)
			return handlers.HandleGetPortDescription(appConfig.MiyagiSocketPath, request, msgID, appConfig.FrameEnd)
			// Port Speed checks for <get-config>
		} else if bytes.Contains(request, []byte("<port-speed")) &&
			(bytes.Contains(request, []byte(fmt.Sprintf("xmlns=\"%s\"", handlers.PortSpeedNamespace))) || bytes.Contains(request, []byte(fmt.Sprintf("xmlns='%s'", handlers.PortSpeedNamespace)))) {
			log.Printf("NETCONF_SERVER: Dispatching to HandleGetPortSpeed for <get-config> with Port Speed filter. Message ID: %s", msgID)
			return handlers.HandleGetPortSpeed(appConfig.MiyagiSocketPath, request, msgID, appConfig.FrameEnd)
		}
		log.Printf("NETCONF_SERVER: Received <get-config> operation with an unhandled filter. Message ID: %s. Request: %s", msgID, string(request))
		return buildErrorResponse(appConfig.FrameEnd, msgID, "operation-not-supported", "The <get-config> operation with the specified filter is not supported.")

	} else if bytes.HasPrefix(rpcQuery, []byte("<rpc")) && bytes.Contains(rpcQuery, []byte("<edit-config")) {
		if handler, ok := rpcHandlers["edit-config"]; ok {
			// The original 'request' is passed to the handler as it might need the full message including XML declaration for some parsing.
			return handler(appConfig.MiyagiSocketPath, appConfig.FrameEnd, request, msgID)
		}
		// If <edit-config> is present but doesn't match the handler's internal checks (e.g. for <vlans>),
		// the handler itself (defined in init()) returns an error or OK.
	}

	log.Printf("NETCONF_SERVER: Received unhandled RPC or malformed request: %s", string(request))
	return buildErrorResponse(appConfig.FrameEnd, msgID, "operation-not-supported", "Operation not supported or request malformed.")
}
func extractMessageID(request []byte) string {
	if i := bytes.Index(request, []byte(`message-id="`)); i > -1 {
		request = request[i+12:]
		if j := bytes.IndexByte(request, '"'); j > -1 {
			return string(request[:j])
		}
	}
	return "1"
}

// buildErrorResponse is a generic helper to construct an rpc-error reply.
// This can be used by main.go for dispatch errors or unhandled operations.
func buildErrorResponse(frameEnd string, msgID string, errTag string, errMsg string) []byte {
	// Basic XML escaping for the error message
	escapedErrMsg := strings.ReplaceAll(errMsg, "<", "&lt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, ">", "&gt;")
	escapedErrMsg = strings.ReplaceAll(escapedErrMsg, "&", "&amp;")

	return []byte(fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply message-id="%s" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <rpc-error>
    <error-type>application</error-type>
    <error-tag>%s</error-tag>
    <error-severity>error</error-severity>
    <error-message xml:lang="en">%s</error-message>
  </rpc-error>
</rpc-reply>
%s`, msgID, errTag, escapedErrMsg, frameEnd,
	))
}

func readFrame(channel ssh.Channel) ([]byte, error) {
	var buffer bytes.Buffer
	frameEndBytes := []byte(appConfig.FrameEnd) // Use configured frame end
	readBuf := make([]byte, 4096)

	for {
		n, err := channel.Read(readBuf)
		if err != nil {
			return nil, err // Propagate EOF or other errors
		}

		buffer.Write(readBuf[:n])

		// Check if the full frame terminator is in the buffer
		if terminatorIndex := bytes.Index(buffer.Bytes(), frameEndBytes); terminatorIndex != -1 {
			// Extract the message part, up to the terminator
			message := buffer.Bytes()[:terminatorIndex]

			// Prepare buffer for the next read, keeping any data that came after the current frame's terminator
			// This handles cases where multiple frames might be in the buffer or a partial next frame.
			remainingData := buffer.Bytes()[terminatorIndex+len(frameEndBytes):]
			buffer.Reset() // Clear the buffer
			if len(remainingData) > 0 {
				buffer.Write(remainingData) // Write back any remaining data
			}

			return bytes.TrimSpace(message), nil
		}
	}
}
