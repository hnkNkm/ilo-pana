package session

import (
	"net/http"
	"net/url"
	"sync"
)

// CookieTracker helps track cookies set during requests
// This is needed because cookiejar doesn't expose its cookies directly
type CookieTracker struct {
	mu      sync.RWMutex
	cookies map[string][]*SerializedCookie // domain -> cookies
}

// NewCookieTracker creates a new cookie tracker
func NewCookieTracker() *CookieTracker {
	return &CookieTracker{
		cookies: make(map[string][]*SerializedCookie),
	}
}

// TrackResponse extracts and tracks cookies from Set-Cookie headers
func (ct *CookieTracker) TrackResponse(resp *http.Response) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		return
	}
	
	// Get domain from request URL
	domain := resp.Request.URL.Hostname()
	if domain == "" {
		return
	}
	
	// Convert http.Cookie to SerializedCookie
	for _, cookie := range cookies {
		sc := &SerializedCookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  cookie.Expires,
			Secure:   cookie.Secure,
			HttpOnly: cookie.HttpOnly,
		}
		
		// Default domain to request domain if not set
		if sc.Domain == "" {
			sc.Domain = domain
		}
		
		// Default path if not set
		if sc.Path == "" {
			sc.Path = "/"
		}
		
		// Convert SameSite
		switch cookie.SameSite {
		case http.SameSiteStrictMode:
			sc.SameSite = "Strict"
		case http.SameSiteLaxMode:
			sc.SameSite = "Lax"
		case http.SameSiteNoneMode:
			sc.SameSite = "None"
		default:
			sc.SameSite = ""
		}
		
		// Update or add cookie
		ct.updateCookie(sc.Domain, sc)
	}
}

// updateCookie updates or adds a cookie for a domain
func (ct *CookieTracker) updateCookie(domain string, cookie *SerializedCookie) {
	domainCookies := ct.cookies[domain]
	
	// Look for existing cookie with same name
	for i, existing := range domainCookies {
		if existing.Name == cookie.Name && existing.Path == cookie.Path {
			// Update existing cookie
			domainCookies[i] = cookie
			return
		}
	}
	
	// Add new cookie
	ct.cookies[domain] = append(domainCookies, cookie)
}

// GetAllCookies returns all tracked cookies
func (ct *CookieTracker) GetAllCookies() []*SerializedCookie {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	
	var all []*SerializedCookie
	for _, domainCookies := range ct.cookies {
		all = append(all, domainCookies...)
	}
	return all
}

// GetCookiesForURL returns cookies that would be sent to the given URL
func (ct *CookieTracker) GetCookiesForURL(u *url.URL) []*SerializedCookie {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	
	domain := u.Hostname()
	path := u.Path
	if path == "" {
		path = "/"
	}
	
	var result []*SerializedCookie
	
	// Check cookies for this domain and parent domains
	for cookieDomain, cookies := range ct.cookies {
		if domainMatches(domain, cookieDomain) {
			for _, cookie := range cookies {
				if pathMatches(path, cookie.Path) {
					result = append(result, cookie)
				}
			}
		}
	}
	
	return result
}

// domainMatches checks if a cookie domain matches the request domain
func domainMatches(requestDomain, cookieDomain string) bool {
	// Exact match
	if requestDomain == cookieDomain {
		return true
	}
	
	// Subdomain match (cookie domain starts with .)
	if len(cookieDomain) > 0 && cookieDomain[0] == '.' {
		// .example.com matches foo.example.com
		return requestDomain == cookieDomain[1:] || 
			   hasSuffix(requestDomain, cookieDomain)
	}
	
	return false
}

// pathMatches checks if a request path matches a cookie path
func pathMatches(requestPath, cookiePath string) bool {
	// Exact match
	if requestPath == cookiePath {
		return true
	}
	
	// Prefix match
	if len(requestPath) > len(cookiePath) {
		if requestPath[:len(cookiePath)] == cookiePath {
			// Check if it's a proper prefix (ends with / or next char is /)
			if cookiePath[len(cookiePath)-1] == '/' {
				return true
			}
			if len(requestPath) > len(cookiePath) && requestPath[len(cookiePath)] == '/' {
				return true
			}
		}
	}
	
	return false
}

// hasSuffix is a simple suffix check
func hasSuffix(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}