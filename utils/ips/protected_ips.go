package ips

import (
	"net"
	"regexp"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/lavanet/lava/utils"
)

// isValidDomainName checks if the given string is a valid domain name.
func isValidDomainName(domain string) bool {
	// Basic regular expression for domain name validation
	// This is a simplified regex and may need adjustments based on your specific requirements
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
	return domainRegex.MatchString(domain)
}

func validateIp(host string) bool {
	ip := net.ParseIP(host)
	if ip != nil {
		// If the host is an IP address, check if it is not a loopback or reserved address
		return !ip.IsLoopback() && !ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast()
	}
	return false
}

func isValidIPv4(address string) bool {
	ipv4Regex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	return ipv4Regex.MatchString(address)
}

// isValidAddress checks if the given address is a valid and not a reserved or local address.
func IsValidNetworkAddress(address string) bool {
	// Split the address into host and port
	if !IsValidNetworkAddressConsensus(address) {
		return false
	}
	host, _, err := net.SplitHostPort(address)
	if err == nil && host != "" {
		if isValidIPv4(host) {
			return validateIp(host)
		}
		// probably a domain name
		if !isValidDomainName(host) {
			utils.LavaFormatError("Failed parsing domain name, and ip address", nil)
			return false
		}
	}
	return true
}

func IsValidNetworkAddressConsensus(address string) (valid bool) {
	invalidOrProtectedIps := []string{
		"0.0.0.0", "::", ":", "localhost",
	}
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		return false // grpc addresses does not support http prefix.
	}
	if slices.Contains(invalidOrProtectedIps, address) {
		return false
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	if strings.Contains(host, ":") {
		return false // too many columns in url.
	}
	if slices.Contains(invalidOrProtectedIps, host) {
		return false
	}
	if strings.HasPrefix(host, "127.0.0.") {
		return false
	}
	return true
}
