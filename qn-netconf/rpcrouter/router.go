package rpcrouter

import (
	"bytes"
	"fmt"
	"log"

	"qn-netconf/handlers" // Assuming handlers will be in this module path
)

// RPCHandler defines the function signature for NETCONF RPC handlers.
type RPCHandler func(miyagiSocketPath, frameEnd string, request []byte, msgID string) []byte

var rpcHandlers map[string]RPCHandler

func init() {
	// Initialize RPC handlers map
	rpcHandlers = map[string]RPCHandler{
		"get-vlans": func(miyagiSocketPath, frameEnd string, request []byte, msgID string) []byte {
			return handlers.BuildGetVlansResponse(miyagiSocketPath, msgID, frameEnd)
		},
		"edit-config": func(miyagiSocketPath, frameEnd string, request []byte, msgID string) []byte {
			if bytes.Contains(request, []byte(fmt.Sprintf("<vlans xmlns=\"%s\">", handlers.VlanNamespace))) {
				return handlers.HandleEditConfig(miyagiSocketPath, request, msgID, frameEnd)
			}
			log.Printf("NETCONF_RPC_ROUTER: Received <edit-config> for unknown model or malformed VLAN config: %s", string(request))
			return BuildOKResponse(frameEnd, msgID)
		},
		"get-interfaces": func(miyagiSocketPath, frameEnd string, request []byte, msgID string) []byte {
			return handlers.BuildGetInterfacesResponse(miyagiSocketPath, msgID, frameEnd)
		},
	}
}

func DispatchRequest(miyagiSocketPath, frameEnd string, request []byte) []byte {
	msgID := ExtractMessageID(request)

	if bytes.Contains(request, []byte("<get-vlans")) {
		if handler, ok := rpcHandlers["get-vlans"]; ok {
			return handler(miyagiSocketPath, frameEnd, request, msgID)
		}
	} else if bytes.Contains(request, []byte("<edit-config")) {
		if handler, ok := rpcHandlers["edit-config"]; ok {
			return handler(miyagiSocketPath, frameEnd, request, msgID)
		}
	} else if bytes.Contains(request, []byte("<get-interfaces")) {
		if handler, ok := rpcHandlers["get-interfaces"]; ok {
			return handler(miyagiSocketPath, frameEnd, request, msgID)
		}
	}

	log.Printf("NETCONF_RPC_ROUTER: Received unhandled RPC or malformed request: %s", string(request))
	return BuildOKResponse(frameEnd, msgID)
}

func ExtractMessageID(request []byte) string {
	if i := bytes.Index(request, []byte(`message-id="`)); i > -1 {
		request = request[i+12:]
		if j := bytes.IndexByte(request, '"'); j > -1 {
			return string(request[:j])
		}
	}
	return "1" // Default message-id
}

func BuildOKResponse(frameEnd string, msgID string) []byte {
	return []byte(fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?><rpc-reply message-id="%s" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><ok/></rpc-reply>%s`,
		msgID, frameEnd,
	))
}

func BuildErrorResponse(frameEnd string, msgID string, errType, errMsg string) []byte {
	return []byte(fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?><rpc-reply message-id="%s" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><rpc-error><error-type>rpc</error-type><error-tag>%s</error-tag><error-severity>error</error-severity><error-message>%s</error-message></rpc-error></rpc-reply>%s`,
		msgID, errType, errMsg, frameEnd,
	))
}
