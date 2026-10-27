// Package nginx reads nginx sites on a server, turns them on and off,
// tests the configuration and reloads it, and watches TLS certificates.
package nginx

import "strings"

// Server is what matters of one server block.
type Server struct {
	Names     []string `json:"names"`
	Listen    []string `json:"listen"`
	Root      string   `json:"root,omitempty"`
	SSL       bool     `json:"ssl"`
	Certs     []string `json:"certs"`
	ProxyPass []string `json:"proxyPass"`
	// Line is where the block starts, to open it in the Editor.
	Line int `json:"line"`
}

// Parsed is what a config file declares.
type Parsed struct {
	Servers   []Server `json:"servers"`
	Upstreams []string `json:"upstreams"`
}

type token struct {
	text string
	line int
	// punct is ";", "{" or "}"; empty for a word.
	punct string
}

// tokenize splits nginx syntax into words and punctuation, dropping
// comments and quotes.
func tokenize(src string) []token {
	var out []token
	line := 1
	i := 0
	for i < len(src) {
		c := src[i]
		switch c {
		case '\n':
			line++
			i++
		case ' ', '\t', '\r':
			i++
		case '#':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case ';', '{', '}':
			out = append(out, token{punct: string(c), line: line})
			i++
		case '"', '\'':
			start, startLine := i+1, line
			i++
			for i < len(src) && src[i] != c {
				switch src[i] {
				case '\\':
					i++
				case '\n':
					line++
				}
				i++
			}
			out = append(out, token{text: src[start:min(i, len(src))], line: startLine})
			i++
		default:
			start := i
			for i < len(src) && !strings.ContainsRune(" \t\r\n;{}#", rune(src[i])) {
				if src[i] == '\\' {
					i++
				}
				i++
			}
			out = append(out, token{text: src[start:min(i, len(src))], line: line})
		}
	}
	return out
}

// Parse reads the server and upstream blocks of a config file. It does
// not follow includes: sites are read file by file.
func Parse(src string) Parsed {
	p := Parsed{Servers: []Server{}, Upstreams: []string{}}
	var stack []string // block names, outermost first
	var words []token
	server := -1 // index in p.Servers of the block being read
	serverDepth := 0
	for _, tok := range tokenize(src) {
		switch tok.punct {
		case "":
			words = append(words, tok)
		case "{":
			name := ""
			if len(words) > 0 {
				name = words[0].text
			}
			if name == "upstream" && len(words) > 1 {
				p.Upstreams = append(p.Upstreams, words[1].text)
			}
			stack = append(stack, name)
			if name == "server" && server < 0 && !inside(stack[:len(stack)-1], "upstream") {
				p.Servers = append(p.Servers, Server{Names: []string{}, Listen: []string{}, Certs: []string{}, ProxyPass: []string{}, Line: words[0].line})
				server, serverDepth = len(p.Servers)-1, len(stack)
			}
			words = nil
		case "}":
			if len(stack) > 0 {
				if len(stack) == serverDepth {
					server, serverDepth = -1, 0
				}
				stack = stack[:len(stack)-1]
			}
			words = nil
		case ";":
			if server >= 0 && len(words) > 0 {
				directive(&p.Servers[server], words, len(stack) == serverDepth)
			}
			words = nil
		}
	}
	return p
}

func inside(stack []string, name string) bool {
	for _, s := range stack {
		if s == name {
			return true
		}
	}
	return false
}

// directive records one directive of a server block; top is false inside
// a location, where only proxy_pass and root count.
func directive(s *Server, words []token, top bool) {
	args := make([]string, 0, len(words)-1)
	for _, w := range words[1:] {
		args = append(args, w.text)
	}
	switch words[0].text {
	case "server_name":
		if top {
			s.Names = append(s.Names, args...)
		}
	case "listen":
		if top && len(args) > 0 {
			s.Listen = append(s.Listen, strings.Join(args, " "))
			for _, a := range args[1:] {
				if a == "ssl" {
					s.SSL = true
				}
			}
		}
	case "ssl_certificate":
		if top && len(args) > 0 {
			s.Certs = append(s.Certs, args[0])
			s.SSL = true
		}
	case "root":
		if top && len(args) > 0 {
			s.Root = args[0]
		}
	case "proxy_pass":
		if len(args) > 0 {
			s.ProxyPass = append(s.ProxyPass, args[0])
		}
	}
}
