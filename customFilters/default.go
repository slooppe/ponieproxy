package customFilters

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"

	"github.com/ctoyan/ponieproxy/internal/config"
	"github.com/ctoyan/ponieproxy/internal/filters"
	"github.com/ctoyan/ponieproxy/pkg/utils"
	"github.com/elazarl/goproxy"
)

/* Request filter
 * Write to UserData for every request.
 *
 * UserData is a part of the proxy context.
 * It is passed to every request and response.
 */
func PopulateUserdata(f *config.Flags) filters.RequestFilter {
	urlsList, err := utils.ReadLines(f.URLFile)
	if err != nil {
		log.Fatalf("error reading lines from file: %v", err)
	}

	return filters.RequestFilter{
		Conditions: []goproxy.ReqCondition{
			goproxy.UrlMatches(regexp.MustCompile(fmt.Sprintf("(%v)", strings.Join(urlsList, ")|(")))),
		},
		Handler: func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			reqBody, err := ioutil.ReadAll(req.Body)
			if err != nil {
				fmt.Printf("error reading reqBody: %v\n", err)
			}

			requestDump, err := httputil.DumpRequest(req, false)
			if err != nil {
				fmt.Printf("error on request dump: %v\n", err)
			}

			checksum := sha1.Sum([]byte(fmt.Sprintf("%s%s%s", req.URL.Host, req.URL.Path, reqBody)))
			ctx.UserData = filters.UserData{
				ReqBody:  string(reqBody),
				ReqDump:  string(requestDump),
				Checksum: hex.EncodeToString(checksum[:]),
			}

			req.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody))
			return req, nil
		},
	}
}

/* Request filter
 * Write it to a uniquely named *.req file, in the output folder
 *
 * The only filter condition, wraps every line from your urls file
 * between braces and concatenates them, making the following regex:
 * (LINE_ONE)|(LINE_TWO)|(LINE_THREE), where LINE_N is a single line from your file.
 */
func WriteReq(f *config.Flags) filters.RequestFilter {
	urlsList, err := utils.ReadLines(f.URLFile)
	if err != nil {
		log.Fatalf("error reading lines from file: %v", err)
	}

	return filters.RequestFilter{
		Conditions: []goproxy.ReqCondition{
			goproxy.UrlMatches(regexp.MustCompile(fmt.Sprintf("(%v)", strings.Join(urlsList, ")|(")))),
		},
		Handler: func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			ud := ctx.UserData.(filters.UserData)
			go utils.WriteUniqueFile(ud.Checksum, ud.ReqBody, f.OutputDir, ud.ReqDump, "req")

			return req, nil
		},
	}
}

/* Response filter
 * Write it to a uniquely named *.res file, in the output folder
 *
 * The only filter condition, wraps every line from your urls file
 * between braces and concatenates them, making the following regex:
 * (LINE_ONE)|(LINE_TWO)|(LINE_THREE), where LINE_N is a single line from your file.
 */
func WriteResp(f *config.Flags) filters.ResponseFilter {
	urlsList, err := utils.ReadLines(f.URLFile)
	if err != nil {
		log.Fatalf("error reading lines from file: %v", err)
	}

	return filters.ResponseFilter{
		Conditions: []goproxy.RespCondition{
			goproxy.UrlMatches(regexp.MustCompile(fmt.Sprintf("(%v)", strings.Join(urlsList, ")|(")))),
		},
		Handler: func(res *http.Response, ctx *goproxy.ProxyCtx) *http.Response {

			responseDump, err := httputil.DumpResponse(res, true)
			if err != nil {
				fmt.Printf("error on response dump: %v\n", err)
			}

			ud := ctx.UserData.(filters.UserData)
			go utils.WriteUniqueFile(ud.Checksum, ud.ReqBody, f.OutputDir, string(responseDump), "res")

			return res
		},
	}
}

/* Request filter
 * Detect IDOR params
 *
 * Following the HUNT Methodology (https://github.com/bugcrowd/HUNT):
 * We're looking for the following strings in request url or in request body:
 * "account", "doc", "edit", "email", "group", "id", "key", "no", "number", "order", "profile", "report", "user"
 */
func DetectIDOR(f *config.Flags) filters.RequestFilter {
	urlsList, err := utils.ReadLines(f.URLFile)
	if err != nil {
		log.Fatalf("error reading lines from file: %v", err)
	}

	return filters.RequestFilter{
		Conditions: []goproxy.ReqCondition{
			goproxy.UrlMatches(regexp.MustCompile(fmt.Sprintf("(%v)", strings.Join(urlsList, ")|(")))),
		},
		Handler: func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			ud := ctx.UserData.(filters.UserData)
			reqQueryMap := req.URL.Query()

			idorParams := []string{"account", "doc", "edit", "email", "group", "id", "key", "no", "number", "order", "profile", "report", "user"}
			for _, idorParam := range idorParams {
				for queryParam := range reqQueryMap {
					if strings.Contains(strings.ToLower(queryParam), strings.ToLower(idorParam)) {
						slackMsg := fmt.Sprintf("IDOR \nQUERY PARAM: `%v` \nFILE:  `%v`", queryParam, ud.Checksum)
						go utils.SendSlackNotification("https://hooks.slack.com/services/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", slackMsg)
					}
				}
			}

			//Check in body

			return req, nil
		},
	}
}
