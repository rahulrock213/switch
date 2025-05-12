package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

// User struct for authentication.
// This is similar to models.User from your qn-rest project.
type User struct {
	Username string
	Password string // For FetchSNMPUsers, this will be the SHA1 hash from SNMP
	// Add other fields if necessary
}

const (
	// TODO: Make snmpTarget configurable, perhaps via main config.json
	snmpTarget    = "203.0.113.121" // Change to your SNMP target IP
	snmpCommunity = "qnpublic"      // TODO: Make configurable
	usernameOID   = ".1.3.6.1.4.1.89.79.17.1.1"
	passwordOID   = ".1.3.6.1.4.1.89.79.17.1.2"
)

// HashPasswordSHA1 hashes a password using SHA-1 and returns its hex string
func HashPasswordSHA1(password string) string {
	hash := sha1.New()
	hash.Write([]byte(password))
	return hex.EncodeToString(hash.Sum(nil))
}

// FetchSNMPUsers returns a list of users with username and SHA-1 hashed password from SNMP
func FetchSNMPUsers() ([]User, error) {
	params := &gosnmp.GoSNMP{
		Target:    snmpTarget,
		Port:      161,
		Community: snmpCommunity,
		Version:   gosnmp.Version2c,
		Timeout:   2 * time.Second,
		Retries:   2,
	}

	log.Printf("NETCONF_AUTH_SNMP: Connecting to SNMP target %s", snmpTarget)
	err := params.Connect()
	if err != nil {
		log.Printf("NETCONF_AUTH_SNMP: Failed to connect to SNMP: %v", err)
		return nil, err
	}
	defer params.Conn.Close()
	log.Printf("NETCONF_AUTH_SNMP: Connected to SNMP")

	userMap := make(map[string]string)
	passMap := make(map[string]string)

	// Walk for usernames
	err = params.Walk(usernameOID, func(pdu gosnmp.SnmpPDU) error {
		// log.Printf("NETCONF_AUTH_SNMP: Raw Username PDU: %+v", pdu) // Verbose
		if pdu.Type == gosnmp.OctetString {
			if val, ok := pdu.Value.([]byte); ok {
				index := strings.TrimPrefix(pdu.Name, usernameOID+".")
				username := string(val)
				userMap[index] = username
				// log.Printf("NETCONF_AUTH_SNMP: Found username: %s (index: %s)", username, index) // Verbose
			} else {
				log.Printf("NETCONF_AUTH_SNMP: Unexpected PDU value type for username: %v", pdu.Type)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("NETCONF_AUTH_SNMP: Error walking usernameOID: %v", err)
		return nil, err
	}
	if len(userMap) == 0 {
		log.Printf("NETCONF_AUTH_SNMP: No usernames found from SNMP walk")
	}

	// Walk for passwords
	err = params.Walk(passwordOID, func(pdu gosnmp.SnmpPDU) error {
		// log.Printf("NETCONF_AUTH_SNMP: Raw Password PDU: %+v", pdu) // Verbose
		if pdu.Type == gosnmp.OctetString {
			if val, ok := pdu.Value.([]byte); ok {
				index := strings.TrimPrefix(pdu.Name, passwordOID+".")
				// The password from SNMP is already a SHA1 hash, sometimes prefixed with '#'
				passwordHash := strings.TrimPrefix(string(val), "#")
				passMap[index] = passwordHash
				// log.Printf("NETCONF_AUTH_SNMP: Found password hash: %s (index: %s)", passwordHash, index) // Verbose
			} else {
				log.Printf("NETCONF_AUTH_SNMP: Unexpected PDU value type for password: %v", pdu.Type)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("NETCONF_AUTH_SNMP: Error walking passwordOID: %v", err)
		return nil, err
	}
	if len(passMap) == 0 {
		log.Printf("NETCONF_AUTH_SNMP: No passwords found from SNMP walk")
	}

	var users []User
	for index, username := range userMap {
		if passHash, ok := passMap[index]; ok {
			users = append(users, User{
				Username: strings.TrimSpace(username),
				Password: strings.TrimSpace(passHash), // This is the SHA1 hash from SNMP
			})
			// log.Printf("NETCONF_AUTH_SNMP: SNMP user loaded: %s -> %s (hash)", username, passHash) // Verbose
		}
	}

	log.Printf("NETCONF_AUTH_SNMP: Fetched %d Switch users via SNMP", len(users))
	return users, nil
}

// ValidateCredentials checks the provided username and plaintext password against users fetched from SNMP.
func ValidateCredentials(username, password string) (bool, error) {
	log.Printf("NETCONF_AUTH_SNMP: Authenticating user: %s", username)

	snmpUsers, err := FetchSNMPUsers()
	if err != nil {
		log.Printf("NETCONF_AUTH_SNMP: Failed to fetch SNMP users for validation: %v", err)
		return false, fmt.Errorf("internal error fetching user credentials")
	}

	hashedInputPassword := HashPasswordSHA1(strings.TrimSpace(password))

	for _, snmpUser := range snmpUsers {
		if strings.TrimSpace(snmpUser.Username) == strings.TrimSpace(username) &&
			strings.TrimSpace(snmpUser.Password) == hashedInputPassword { // snmpUser.Password is already the hash
			log.Printf("NETCONF_AUTH_SNMP: User '%s' authenticated successfully via SNMP.", username)
			return true, nil
		}
	}

	log.Printf("NETCONF_AUTH_SNMP: Invalid login attempt for username: %s", username)
	return false, errors.New("invalid username or password")
}
