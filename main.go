package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/boutros/marc"
)

const xmlHeader = `<?xml version="1.0" encoding="UTF-8"?>
<collection
  xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:schemaLocation="http://www.loc.gov/MARC21/slim http://www.loc.gov/standards/marcxml/schema/MARC21slim.xsd"
  xmlns="http://www.loc.gov/MARC21/slim">`

const xmlFooter = `</collection>`

func clean(s string) string {
	if len(s) < 2 {
		return s
	}
	switch s[len(s)-1] {
	case '.', ',', ':', ';', '/':
		return s[:len(s)-1]
	}
	return strings.TrimSuffix(s, "/")
}

func strip(resp *http.Response) error {
	dec := marc.NewDecoder(resp.Body, marc.MARCXML)
	defer resp.Body.Close()
	recs, err := dec.DecodeAll()
	if err != nil {
		return err
	}

	var res []*marc.Record
	for _, r := range recs {
		newRec := marc.NewRecord()
		for _, df := range r.DataFields {
			f := marc.NewDField(df.Tag)
			f.Ind1 = df.Ind1
			f.Ind2 = df.Ind2
			for _, sf := range df.SubFields {
				f = f.AddSubField(sf.Code, clean(sf.Value))
			}
			newRec.AddDField(f)
		}
		res = append(res, newRec)
	}
	var b bytes.Buffer
	b.WriteString(xmlHeader)
	enc := marc.NewEncoder(&b, marc.MARCXML)
	for _, r := range res {
		enc.Encode(r)
	}
	enc.Flush()
	b.WriteString(xmlFooter)
	resp.Body = ioutil.NopCloser(&b)
	resp.ContentLength = int64(len(b.Bytes()))
	resp.Header.Set("Content-Length", strconv.Itoa(len(b.Bytes())))
	return nil
}

func main() {
	var (
		fromAddr = flag.String("from", ":3001", "proxy from address")
		toAddr   = flag.String("to", "http://localhost:3000", "proxy to  address")
	)
	flag.Parse()
	toURL, err := url.Parse(*toAddr)
	if err != nil {
		log.Fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(toURL)
	proxy.ModifyResponse = strip
	if err := http.ListenAndServe(*fromAddr, proxy); err != nil {
		log.Fatal(err)
	}
}
