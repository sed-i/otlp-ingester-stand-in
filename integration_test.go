package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Log ingestion and query", func() {
	var (
		server  *httptest.Server
		baseURL string
	)

	BeforeEach(func() {
		server = httptest.NewServer(newRouter())
		baseURL = server.URL

		logStore.records = nil
		metricStore.records = nil
		spanStore.records = nil
	})

	AfterEach(func() {
		server.Close()
	})

	It("returns the matching log record when filtering by exact body", func() {
		logsJSON, err := os.ReadFile("testdata/logs.json")
		Expect(err).NotTo(HaveOccurred())

		resp, err := http.Post(baseURL+"/v1/logs", "application/json", bytes.NewReader(logsJSON))
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		resp.Body.Close()

		queryResp, err := http.Get(baseURL + "/api/v1/logs?body=Example+log+record")
		Expect(err).NotTo(HaveOccurred())
		defer queryResp.Body.Close()

		Expect(queryResp.StatusCode).To(Equal(http.StatusOK))

		body, err := io.ReadAll(queryResp.Body)
		Expect(err).NotTo(HaveOccurred())

		var records []FlatLogRecord
		err = json.Unmarshal(body, &records)
		Expect(err).NotTo(HaveOccurred())

		Expect(records).To(HaveLen(1))
		Expect(records[0].Body).To(Equal("Example log record"))
		Expect(records[0].Severity).To(Equal("Information"))
		Expect(records[0].Service).To(Equal("my.service"))
	})
})
