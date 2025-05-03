package scanner

import (
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
)

// GetDNSRecords gets all DNS records for a domain
func (s *Scanner) GetDNSRecords(domain string) (map[string][]string, error) {
	records := make(map[string][]string)
	recordTypes := []struct {
		Name string
		Type uint16
	}{
		{"A", dns.TypeA},
		{"AAAA", dns.TypeAAAA},
		{"CNAME", dns.TypeCNAME},
		{"MX", dns.TypeMX},
		{"TXT", dns.TypeTXT},
		{"NS", dns.TypeNS},
		{"SOA", dns.TypeSOA},
	}
	
	// Configure the DNS client
	c := new(dns.Client)
	c.Timeout = s.Timeout
	
	// Try different DNS servers in case one fails
	dnsServers := []string{"8.8.8.8:53", "1.1.1.1:53", "9.9.9.9:53"}
	
	for _, rt := range recordTypes {
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn(domain), rt.Type)
		m.RecursionDesired = true
		
		var r *dns.Msg
		var err error
		
		// Try each DNS server until we get a response or exhaust all servers
		for _, server := range dnsServers {
			r, _, err = c.Exchange(m, server)
			if err == nil && r != nil && r.Rcode == dns.RcodeSuccess {
				break
			}
		}
		
		if err != nil || r == nil || r.Rcode != dns.RcodeSuccess {
			continue
		}
		
		var values []string
		for _, answer := range r.Answer {
			// Parse different record types
			switch rt.Type {
			case dns.TypeA:
				if a, ok := answer.(*dns.A); ok {
					values = append(values, a.A.String())
				}
			case dns.TypeAAAA:
				if aaaa, ok := answer.(*dns.AAAA); ok {
					values = append(values, aaaa.AAAA.String())
				}
			case dns.TypeCNAME:
				if cname, ok := answer.(*dns.CNAME); ok {
					values = append(values, cname.Target)
				}
			case dns.TypeMX:
				if mx, ok := answer.(*dns.MX); ok {
					values = append(values, fmt.Sprintf("%d %s", mx.Preference, mx.Mx))
				}
			case dns.TypeTXT:
				if txt, ok := answer.(*dns.TXT); ok {
					values = append(values, strings.Join(txt.Txt, " "))
				}
			case dns.TypeNS:
				if ns, ok := answer.(*dns.NS); ok {
					values = append(values, ns.Ns)
				}
			case dns.TypeSOA:
				if soa, ok := answer.(*dns.SOA); ok {
					values = append(values, fmt.Sprintf("%s %s %d %d %d %d %d", 
						soa.Ns, soa.Mbox, soa.Serial, soa.Refresh, soa.Retry, soa.Expire, soa.Minttl))
				}
			default:
				values = append(values, answer.String())
			}
		}
		
		if len(values) > 0 {
			records[rt.Name] = values
		}
	}
	
	return records, nil
}

// GetCreationInfo attempts to get creation date information via WHOIS
func (s *Scanner) GetCreationInfo(domain string) string {
	// Default return if we can't get creation date
	creationDate := "Not available"
	
	// Only attempt WHOIS on the main domain or direct subdomains
	// to avoid excessive WHOIS queries which may get rate limited
	parts := strings.Split(domain, ".")
	if len(parts) > 3 {
		return creationDate
	}
	
	try := 3 // Number of attempts to make
	for i := 0; i < try; i++ {
		// Try to get WHOIS data
		rawWhois, err := whois.Whois(domain)
		if err != nil {
			time.Sleep(time.Second) // Wait between retries
			continue
		}
		
		// Try to parse the WHOIS data
		result, err := whoisparser.Parse(rawWhois)
		if err != nil {
			time.Sleep(time.Second) // Wait between retries
			continue
		}
		
		// Extract creation date
		if result.Domain != nil && result.Domain.CreatedDate != "" {
			creationTime, err := time.Parse(time.RFC3339, result.Domain.CreatedDate)
			if err == nil {
				return creationTime.Format("2006-01-02")
			}
			return result.Domain.CreatedDate
		}
		
		// If we got this far but couldn't get a date, don't retry
		break
	}
	
	return creationDate
}