package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
)

func (e *localDNSProviderSolver) handleDNSRequest(w dns.ResponseWriter, req *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(req)
	switch req.Opcode {
	case dns.OpcodeQuery:
		for _, q := range msg.Question {
			if err := e.addDNSAnswer(q, msg, req); err != nil {
				msg.SetRcode(req, dns.RcodeServerFailure)
				break
			}
		}
	}
	w.WriteMsg(msg)
}

func (e *localDNSProviderSolver) PublicIPIsCNAME() bool {
	// if e.PublicIP is not an IP address, it must be a CNAME
	pubIP := e.PublicIP
	if pubIP == "" {
		pubIP = e.Nameserver
	}
	if pubIP == "" {
		return false
	}
	// if the IP is not an IP address, it must be a CNAME
	return net.ParseIP(pubIP) == nil
}

func (e *localDNSProviderSolver) addDNSAnswer(q dns.Question, msg *dns.Msg, req *dns.Msg) error {
	q.Name = strings.ToLower(q.Name)
	l := log.WithFields(log.Fields{
		"question": q,
		"qname":    q.Name,
		"type":     dns.TypeToString[q.Qtype],
		"msg":      msg,
		"req":      req,
	})
	l.Debug("handling DNS request")
	switch q.Qtype {
	// TXT records are the only important record for ACME dns-01 challenges
	case dns.TypeTXT:
		l.Debug("handling TXT request")
		record, err := e.store.Get(q.Name)
		if err != nil {
			msg.SetRcode(req, dns.RcodeNameError)
			return nil
		}
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN TXT %s", q.Name, record))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		return nil

	case dns.TypeCNAME:
		l.Debug("handling CNAME request")
		// always return bad domain name error for any CNAME query except
		// for the domain name we are authoritative for
		if q.Name != e.DomainName {
			msg.SetRcode(req, dns.RcodeNameError)
			return nil
		}
		// if the public IP is a CNAME, return that
		if e.PublicIPIsCNAME() {
			rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN CNAME %s", q.Name, e.PublicIP))
			if err != nil {
				return err
			}
			msg.Answer = append(msg.Answer, rr)
			return nil
		}
		return nil
		// Always return loopback for any A query
	case dns.TypeA:
		l.Debug("handling A request")
		v := "127.0.0.1"
		if q.Name == e.DomainName && !e.PublicIPIsCNAME() {
			v = e.PublicIP
		}
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN A %s", q.Name, v))
		if err != nil {
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		l.WithField("answer", rr).Debug("added answer")
		return nil

	// NS and SOA are for authoritative lookups, return the values configured
	case dns.TypeNS:
		l.Debug("handling NS request")
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN NS %s", q.Name, e.Nameserver))
		if err != nil {
			l.WithError(err).Error("failed to create NS record")
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		l.WithField("answer", rr).Debug("added answer")
		return nil
	case dns.TypeSOA:
		l.Debug("handling SOA request")
		rr, err := dns.NewRR(fmt.Sprintf("%s 5 IN SOA %s %s 0 0 0 0 0", e.Nameserver, e.Nameserver, e.RName))
		if err != nil {
			l.WithError(err).Error("failed to create SOA record")
			return err
		}
		msg.Answer = append(msg.Answer, rr)
		l.WithField("answer", rr).Debug("added answer")
		return nil
	default:
		return fmt.Errorf("unimplemented record type %v", q.Qtype)
	}
}
