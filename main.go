package main

import (
    "fmt"
	"log"
	"net"
    "net/http"

	"github.com/miekg/dns"
)

type Log struct{
    storage []string
    size int
    index int
}

func NewLog(size int) *Log {
    return &Log{
        size: size,
        storage: make([]string, size),
    }
}

func (l *Log) Store(v string) {
    l.storage[l.index % l.size] = v
    l.index++
}

func (l *Log) Do(do func(string)) {
    s := l.index
    for i := 0; i < l.size; i++ {
        do(l.storage[(s + i) % l.size])
    }
}

type server struct{
    l *Log
}

func (s server) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	// grab remote server addr for logs
	addr, _, err := net.SplitHostPort(w.RemoteAddr().String())
	if err != nil {
		log.Print(err)
		return
	}

	for _, question := range r.Question {
        v := fmt.Sprintf("%16s   %s", addr, question.String())
        s.l.Store(v)
        log.Print(v)
	}
}

func (s server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    w.Header().Add("Content-Type", "text/plain")
    s.l.Do(func(x string) {
        if x != "" {
            fmt.Fprintf(w, "%s\n", x)
        }
    })
}

func main() {
    s := server{NewLog(128)}
	go dns.ListenAndServe(":53", "udp", s)

    log.Fatal(http.ListenAndServe(":7000", s))
}

