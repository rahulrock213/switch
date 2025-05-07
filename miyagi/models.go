package miyagi

import "encoding/json"

// MiyagiRequest defines the structure for requests to the Miyagi agent.
type MiyagiRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
	ID     int                    `json:"id"`
}

// MiyagiResponse defines a generic structure for responses from Miyagi.
type MiyagiResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"` // Use RawMessage to delay parsing
	Error  *MiyagiError    `json:"error,omitempty"`
}

// MiyagiError defines the structure of an error object from Miyagi.
type MiyagiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// VlanInfoFromMiyagi represents the structure of a single VLAN from Miyagi's Get.VLAN.Table.
// This is an assumption; you might need to adjust it based on the actual Miyagi response.
// For example, if Miyagi returns [ ["VLAN0001", "1"], ["VLAN0010", "10"] ]
// then the parsing logic will be different.
// For now, let's assume a more structured response for easier parsing.
type VlanInfoFromMiyagi struct {
	VlanID int    `json:"vlan_id"`
	Name   string `json:"name"`
	// Add other fields if Miyagi provides them and they are needed for NETCONF.
}
