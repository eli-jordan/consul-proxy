package main

import (
	"log"
	"github.com/miekg/dns"
	"fmt"
	"strconv"
)

type DnsRecord struct {
	ip   string
	port uint16
}

type MockDnsServer struct {
	records map[string]*DnsRecord
	port    int

	server *dns.Server
}

func (s *MockDnsServer) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		s.writeAnswer(m)
	}

	w.WriteMsg(m)
}

func (s *MockDnsServer) writeAnswer(m *dns.Msg) {
	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeSRV:
			log.Printf("Query for %s\n", q.Name)

			record := s.records[q.Name]

			log.Printf("Found record %v", record)
			srv, err := dns.NewRR(fmt.Sprintf("%s 6 IN SRV 1 1 %v %v", q.Name, record.port, record.ip))

			if err != nil {
				panic(err)
			}

			m.Answer = append(m.Answer, srv)
			log.Printf("Sending Answer: %v", m.Answer)
		}
	}
}

func (s *MockDnsServer) start() {
	if s.port == 0 {
		s.port = getFreePort()
	}

	// attach request handler func
	dns.HandleFunc("service.", s.handleRequest)

	s.server = &dns.Server{Addr: ":" + strconv.Itoa(s.port), Net: "udp" }
	log.Printf("Starting at %d\n", s.port)
	go func() {
		err := s.server.ListenAndServe()
		if err != nil {
			log.Fatalf("Failed to start server: %s\n ", err.Error())
		}
	}()
}

func (s *MockDnsServer) stop() {
	if s.server != nil {
		s.server.Shutdown()
	}
}
