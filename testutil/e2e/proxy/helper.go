package e2e

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func formatURL(u string) (scheme string, user string, password string, finalURL string) {
	scheme = "http"
	if strings.Contains(u, "http://") {
		u = strings.Split(u, "//")[1]
	} else if strings.Contains(u, "https://") {
		u = strings.Split(u, "//")[1]
		scheme = "https"
	}
	if strings.Contains(u, "@") {
		userPass, finalURL := strings.Split(u, "@")[0], strings.Split(u, "@")[1]
		if strings.Contains(userPass, ":") {
			return scheme, strings.Split(userPass, ":")[0], strings.Split(userPass, ":")[1], finalURL
		}
		return scheme, "", "", u
	}
	return scheme, "", "", u
}
func createProxyRequest(req *http.Request, hostURL string, body string) (proxyRequest *http.Request, err error) {
	reqUrl := req.URL
	scheme, user, password, hostURL := formatURL(hostURL)
	println("!!!!!!!", user, password, hostURL)
	reqUrl.Host = hostURL
	// url.User(
	if user != "" {
		reqUrl.User = url.UserPassword(user, password)
	}
	reqUrlStr, err := url.QueryUnescape(reqUrl.String())
	if err != nil {
		return nil, fmt.Errorf(" ::: XXX ::: Could not QueryUnescape new request ::: " + reqUrl.Host + " ::: " + err.Error())
	}
	proxyReq, err := http.NewRequest(req.Method, scheme+":"+reqUrlStr, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf(" ::: XXX ::: Could not reproduce new request ::: " + reqUrl.Host + " ::: " + err.Error())
	}
	proxyReq.Header.Set("Host", req.Host)
	proxyReq.Header.Set("X-Forwarded-For", req.RemoteAddr)
	proxyReq.Header.Set("Accept", "application/json")
	proxyReq.Header.Set("Accept-Encoding", "identity")
	proxyReq.Header.Set("Content-Length", strconv.FormatInt(req.ContentLength, 10))
	// proxyReq.URL.Scheme = "https"
	proxyReq.URL.Scheme = scheme
	for header, values := range req.Header {
		for _, value := range values {
			// println(" ::: Adding Header ::: ", header, ":::", value)
			proxyReq.Header.Add(header, value)
		}
	}
	return proxyReq, nil
}
func sendRequest(request *http.Request) (*http.Response, error) {

	client := &http.Client{}
	proxyRes, err := client.Do(request)

	// proxyReq.Header.Set("Content-Length", )
	if err != nil {
		println(" ::: XXX ::: Reply From Host Error ::: "+request.Host+" ::: ", err.Error())
		return nil, err
	}
	return proxyRes, nil
}

func getDataFromIORead(feed *io.ReadCloser, reset bool) (rawBody []byte) {
	rawBody, err := ioutil.ReadAll(*feed)
	if reset {
		*feed = ioutil.NopCloser(bytes.NewBuffer(rawBody))
	}

	if err != nil {
		fmt.Printf("^^^^^^^^^^^^^^^^^^^^^^^^^^^%e\n", err)
	}
	fmt.Printf("rawBody %s", rawBody)

	return rawBody
}

func returnResponse(rw http.ResponseWriter, status int, body []byte) {
	rw.WriteHeader(status)
	rw.Write(body)
}
