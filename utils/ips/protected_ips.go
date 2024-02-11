package ips

import (
	"net"
	"regexp"
	"strings"

	"github.com/lavanet/lava/utils"
)

var disallowedIps = []string{
	"127.0.0.",   // IPv4 loopback
	"10.0.0.",    // RFC1918
	"172.16.0.",  // RFC1918
	"192.168.0.", // RFC1918
	"169.254.0.", // RFC3927 link-local
	"::1",        // IPv6 loopback
	"fe80::",     // IPv6 link-local
	"fc00::",     // IPv6 unique local addr
	"0.0.0.",
	"::",
	":",
}

// isValidDomainName checks if the given string is a valid domain name.
func isValidDomainName(domain string) bool {
	// Basic regular expression for domain name validation
	// This is a simplified regex and may need adjustments based on your specific requirements
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
	return domainRegex.MatchString(domain)
}

func validateIp(ip net.IP) bool {
	if ip != nil {
		// If the host is an IP address, check if it is not a loopback or reserved address
		isLoopback := ip.IsLoopback()
		isGlobal := ip.IsGlobalUnicast()
		isPrivate := ip.IsPrivate()
		isLinkUnicast := ip.IsLinkLocalUnicast()
		return !isLoopback && !isPrivate && !isLinkUnicast && isGlobal
	}
	return false
}

func isValidIPv4(address string) bool {
	ipv4Regex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	return ipv4Regex.MatchString(address)
}

func isValidIPv6(address string) bool {
	ipv6Regex := regexp.MustCompile(`^([0-9a-fA-F]{0,4}:){1,7}[0-9a-fA-F]{0,4}$`)
	return ipv6Regex.MatchString(address)
}

// isValidAddress checks if the given address is a valid and not a reserved or local address.
func IsValidNetworkAddressProtocol(address string) bool {
	// Split the address into host and port
	if !IsValidNetworkAddressConsensus(address) {
		return false
	}
	// in case we want to add more functionality for protocol specific
	return true
}

// using disallowedIps to validate off network.
func validateIpFromDisallowedList(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false // Not a valid IP address so we cant parse it
	}

	// Check if the address is in the disallowed list
	for _, disallowed := range disallowedIps {
		if strings.HasPrefix(addr, disallowed) {
			return false // Disallowed IP address
		}
	}
	return validateIp(ip)
}

func IsValidNetworkAddressConsensus(address string) (valid bool) {
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		return false // grpc addresses does not support http prefix.
	}
	host, _, err := net.SplitHostPort(address)
	if host == "localhost" {
		return false // do not allow localhost to be used.
	}
	if err != nil {
		return false // failed getting host, invalid address
	}
	if isValidIPv4(host) || isValidIPv6(host) { // validate ipv4 address
		return validateIpFromDisallowedList(host) // validate ip address
	}
	// probably a domain name, validate it.
	if !isValidDomainName(host) {
		utils.LavaFormatError("Failed parsing domain name, and ip address", nil)
		return false
	}
	return true
}
