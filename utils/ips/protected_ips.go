package ips

import (
	"net"
	"regexp"
	"strings"

	"github.com/lavanet/lava/utils"
)

// isValidDomainName checks if the given string is a valid domain name.
func isValidDomainName(domain string) bool {
	// Basic regular expression for domain name validation
	// This is a simplified regex and may need adjustments based on your specific requirements
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)

	return domainRegex.MatchString(domain)
}

// getResolvedIPs returns the IP addresses that the given domain is pointing to.
func getResolvedIPs(domain string) ([]string, error) {
	ips, err := net.LookupHost(domain)
	if err != nil {
		return nil, err
	}
	return ips, nil
}

// isValidAddress checks if the given address is a valid and not a reserved or local address.
func IsValidAddress(address string) bool {
	// Split the address into host and port
	host, _, err := net.SplitHostPort(address)
	if err == nil {
		// Check if the host is an IP address
		ip := net.ParseIP(host)
		if ip != nil {
			// If the host is an IP address, check if it is not a loopback or reserved address
			return !ip.IsLoopback() && !ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast()
		}
	}

	if !isValidDomainName(host) {
		utils.LavaFormatError("Failed parsing domain name, and ip address", nil)
		return false
	}

	resolvedIps, err := getResolvedIPs(address)
	if err != nil {
		utils.LavaFormatError("Failed resolving IP address for domain", err, utils.LogAttr("address", address))
	}
	for _, resolvedIp := range resolvedIps {
		ip := net.ParseIP(resolvedIp)
		// validate all ips this domain is pointing to
		if ip != nil {
			// If the host is an IP address, check if it is not a loopback or reserved address
			valid := !ip.IsLoopback() && !ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast()
			if !valid {
				return false
			}
		}
	}
	return true
}

func ValidateProviderAddressConsensus(address string) (valid bool) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		// domain name and not ip. we cant validate so we return true.
		return true
	}
	if host == "0.0.0.0" {
		return false
	}
	if strings.HasPrefix(host, "127.0.0.") {
		return false
	}
	if host == "localhost" {
		return false
	}
	return true
}
