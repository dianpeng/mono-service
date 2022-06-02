package hpl

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	// router
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/service"

	// library usage
	"encoding/base64"
	"github.com/google/uuid"
	"strings"
	"time"
)

// ----------------------------------------------------------------------------
// builtin functions/libraries for our internal usage

// {{# Codec Library Functions
func fnBase64Encode(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 1 {
		return pl.NewValNull(), fmt.Errorf("function: b64_encode expect 1 argument")
	}
	if argument[0].Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: b64_encode's frist argument must be string")
	}

	out := base64.StdEncoding.EncodeToString([]byte(argument[0].String))
	return pl.NewValStr(out), nil
}

func fnBase64Decode(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 1 {
		return pl.NewValNull(), fmt.Errorf("function: b64_decode expect 1 argument")
	}
	if argument[0].Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: b64_decode's frist argument must be string")
	}

	out, err := base64.StdEncoding.DecodeString(argument[0].String)
	if err != nil {
		return pl.NewValNull(), err
	} else {
		return pl.NewValStr(string(out)), nil
	}
}

func fnURLEncode(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 1 {
		return pl.NewValNull(), fmt.Errorf("function: url_encode expect 1 argument")
	}
	if argument[0].Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: url_encode's frist argument must be string")
	}

	out := url.QueryEscape(argument[0].String)
	return pl.NewValStr(out), nil
}

func fnURLDecode(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 1 {
		return pl.NewValNull(), fmt.Errorf("function: url_decode expect 1 argument")
	}
	if argument[0].Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: url_decode's frist argument must be string")
	}

	out, err := url.QueryUnescape(argument[0].String)
	if err != nil {
		return pl.NewValNull(), err
	} else {
		return pl.NewValStr(out), nil
	}
}

func fnPathEscape(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 1 {
		return pl.NewValNull(), fmt.Errorf("function: path_encode expect 1 argument")
	}
	if argument[0].Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: path_encode's frist argument must be string")
	}

	out := url.PathEscape(argument[0].String)
	return pl.NewValStr(out), nil
}

func fnPathUnescape(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 1 {
		return pl.NewValNull(), fmt.Errorf("function: path_decode expect 1 argument")
	}
	if argument[0].Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: path_decode's frist argument must be string")
	}

	out, err := url.PathUnescape(argument[0].String)
	if err != nil {
		return pl.NewValNull(), err
	} else {
		return pl.NewValStr(out), nil
	}
}

// #}}

// {{# General Functions

func fnToString(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 1 {
		return pl.NewValNull(), fmt.Errorf("function: to_string expect 1 argument")
	}
	x, err := argument[0].ToString()
	if err != nil {
		return pl.NewValNull(), err
	} else {
		return pl.NewValStr(x), nil
	}
}

func fnHttpDate(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 0 {
		return pl.NewValNull(), fmt.Errorf("function: http_date expect 1 argument")
	}

	now := time.Now()
	return pl.NewValStr(now.Format(time.RFC3339)), nil
}

func fnHttpDateNano(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 0 {
		return pl.NewValNull(), fmt.Errorf("function: http_datenano expect 1 argument")
	}

	now := time.Now()
	return pl.NewValStr(now.Format(time.RFC3339Nano)), nil
}

func fnUUID(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 0 {
		return pl.NewValNull(), fmt.Errorf("function: uuid expect 1 argument")
	}
	return pl.NewValStr(uuid.NewString()), nil
}

func fnEcho(argument []pl.Val) (pl.Val, error) {
	// printing the input out, regardless what it is and the echo should print it
	// out always
	var b []interface{}

	for _, x := range argument {
		b = append(b, x.ToNative())
	}

	x, _ := json.MarshalIndent(b, "", "  ")
	log.Println(string(x))
	return pl.NewValNull(), nil
}

// concate all input to a single stream, which can be represented as http body
func fnConcateHttpBody(argument []pl.Val) (pl.Val, error) {
	var input []io.Reader
	for idx, a := range argument {
		if a.Id() == "http.body" {
			body, _ := a.Usr.Context.(*HplHttpBody)
			input = append(input, body.ToStream())
		} else {
			str, err := a.ToString()
			if err != nil {
				return pl.NewValNull(),
					fmt.Errorf("concate_http_body: the %d argument cannot be converted to string: %s",
						idx+1, err.Error())
			}
			input = append(input, strings.NewReader(str))
		}
	}

	return NewHplHttpBodyValFromStream(io.MultiReader(input...)), nil
}

// #}}

// {{# String Manipulation Functions

func fnToUpper(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 1 {
		return pl.NewValNull(), fmt.Errorf("function: to_upper expect 1 argument")
	}
	if argument[0].Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: to_upper's frist argument must be string")
	}
	return pl.NewValStr(strings.ToUpper(argument[0].String)), nil
}

func fnToLower(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 1 {
		return pl.NewValNull(), fmt.Errorf("function: to_lower expect 1 argument")
	}
	if argument[0].Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: to_lower's frist argument must be string")
	}
	return pl.NewValStr(strings.ToLower(argument[0].String)), nil
}

func fnSplit(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 2 {
		return pl.NewValNull(), fmt.Errorf("function: split expect 2 arguments")
	}
	if argument[0].Type != pl.ValStr && argument[1].Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: split expects 2 string arguments")
	}
	x := strings.Split(argument[0].String, argument[1].String)
	ret := pl.NewValList()
	for _, xx := range x {
		ret.AddList(pl.NewValStr(xx))
	}
	return ret, nil
}

func fnTrim(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 2 {
		return pl.NewValNull(), fmt.Errorf("function: trim expect 2 arguments")
	}
	if argument[0].Type != pl.ValStr && argument[1].Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: trim expects 2 string arguments")
	}
	return pl.NewValStr(strings.Trim(argument[0].String, argument[1].String)), nil
}

func fnLTrim(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 2 {
		return pl.NewValNull(), fmt.Errorf("function: ltrim expect 2 arguments")
	}
	if argument[0].Type != pl.ValStr && argument[1].Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: ltrim expects 2 string arguments")
	}
	return pl.NewValStr(strings.TrimLeft(argument[0].String, argument[1].String)), nil
}

func fnRTrim(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 2 {
		return pl.NewValNull(), fmt.Errorf("function: rtrim expect 2 arguments")
	}
	if argument[0].Type != pl.ValStr && argument[1].Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: rtrim expects 2 string arguments")
	}
	return pl.NewValStr(strings.TrimRight(argument[0].String, argument[1].String)), nil
}

// #}}

// {{# HTTP/Network Functions

// general HTTP request interfaces, allowing user to specify following method
// http(
//   [0]: url(string)
//   [1]: method(string)
//   [2]: [header](list[string]),
//   [3]: [body](string)
// )
func fnDoHttp(res service.SessionResource, argument []pl.Val) (pl.Val, error) {
	if len(argument) < 2 {
		return pl.NewValNull(), fmt.Errorf("function: http expect at least 2 arguments")
	}
	url := argument[0]
	method := argument[1]
	if url.Type != pl.ValStr || method.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: http's first 2 arguments must be string")
	}

	var body io.Reader
	if len(argument) == 4 && argument[3].Type == pl.ValStr {
		body = strings.NewReader(argument[3].String)
	} else {
		body = http.NoBody
	}

	req, err := http.NewRequest(method.String, url.String, body)
	if err != nil {
		return pl.NewValNull(), err
	}

	// check header field
	if len(argument) >= 3 && argument[2].Type == pl.ValList {
		for _, hdr := range argument[2].List.Data {
			if hdr.Type == pl.ValPair && hdr.Pair.First.Type == pl.ValStr && hdr.Pair.Second.Type == pl.ValStr {
				req.Header.Add(hdr.Pair.First.String, hdr.Pair.Second.String)
			}
		}
	}

	client, err := res.GetHttpClient(url.String)
	if err != nil {
		return pl.NewValNull(), err
	}

	resp, err := client.Do(req)
	if err != nil {
		return pl.NewValNull(), err
	}

	// serialize the response back to the normal object
	return NewHplHttpResponseVal(resp), nil
}

// header related functions
func fnHeaderHas(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 2 {
		return pl.NewValNull(), fmt.Errorf("function: header_has requires 2 arguments")
	}

	if argument[0].Id() != "http.header" || argument[1].Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: header_has's " +
			"first argument must be header and second " +
			"argument must be string")
	}

	hdr, ok := argument[0].Usr.Context.(*HplHttpHeader)
	must(ok, "must be http.header")

	if vv := hdr.header.Get(argument[1].String); vv == "" {
		return pl.NewValBool(false), nil
	} else {
		return pl.NewValBool(true), nil
	}
}

func doHeaderDelete(hdr http.Header, key string) int {
	prefixWildcard := strings.HasPrefix(key, "*")
	suffixWildcard := strings.HasSuffix(key, "*")
	cnt := 0

	if prefixWildcard && suffixWildcard {
		midfix := key[1 : len(key)-1]
		for key, _ := range hdr {
			if strings.Contains(key, midfix) {
				hdr.Del(key)
				cnt++
			}
		}
	} else if prefixWildcard {
		target := key[1:]
		for key, _ := range hdr {
			if strings.HasSuffix(key, target) {
				hdr.Del(key)
				cnt++
			}
		}
	} else if suffixWildcard {
		target := key[:len(key)-1]
		for key, _ := range hdr {
			if strings.HasPrefix(key, target) {
				hdr.Del(key)
				cnt++
			}
		}
	} else {
		hdr.Del(key)
		cnt++
	}

	return cnt
}

func fnHeaderDelete(argument []pl.Val) (pl.Val, error) {
	if len(argument) != 2 {
		return pl.NewValNull(), fmt.Errorf("function: header_delete requires 2 arguments")
	}

	if argument[0].Id() != "http.header" || argument[1].Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: header_delete's " +
			"first argument must be header and second " +
			"argument must be string")
	}

	hdr, ok := argument[0].Usr.Context.(*HplHttpHeader)
	must(ok, "must be http.header")

	cnt := doHeaderDelete(hdr.header, argument[1].String)
	return pl.NewValInt(cnt), nil
}

// #}}
