package hpl

import (
	"fmt"
	"io"
	"net/http"

	// router
	"github.com/dianpeng/mono-service/pl"

	// library usage
	"strings"
)

type trivialReadCloser struct {
	r io.Reader
}

func (e *trivialReadCloser) Read(b []byte) (int, error) {
	return e.r.Read(b)
}

func (e *trivialReadCloser) Close() error {
	return nil
}

func fnConcateHttpBody(argument []pl.Val) (pl.Val, error) {
	var input []io.Reader
	for idx, a := range argument {
		if a.Id() == "http.body" {
			body, _ := a.Usr().Context.(*Body)
			input = append(input, body.Stream().Stream)
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

	return NewBodyValFromStream(&trivialReadCloser{r: io.MultiReader(input...)}), nil
}

// general HTTP request interfaces, allowing user to specify following method
// http(
//   [0]: url(string)
//   [1]: method(string)
//   [2]: [header](list[string]),
//   [3]: [body](string)
// )
func fnDoHttp(session SessionWrapper, argument []pl.Val) (pl.Val, error) {
	if len(argument) < 2 {
		return pl.NewValNull(), fmt.Errorf("function: http expect at least 2 arguments")
	}
	url := argument[0]
	method := argument[1]
	if url.Type != pl.ValStr || method.Type != pl.ValStr {
		return pl.NewValNull(), fmt.Errorf("function: http's first 2 arguments must be string")
	}

	var body io.Reader
	body = http.NoBody

	if len(argument) == 4 {
		switch argument[3].Type {
		case pl.ValStr:
			body = strings.NewReader(argument[3].String())
			break

		default:
			if argument[3].Id() == HttpBodyTypeId {
				b, _ := argument[3].Usr().Context.(*Body)
				body = b.Stream().Stream
				break
			}
			break
		}
	}

	req, err := http.NewRequest(method.String(), url.String(), body)
	if err != nil {
		return pl.NewValNull(), err
	}

	// check header field
	if len(argument) >= 3 && argument[2].Type == pl.ValList {
		for _, hdr := range argument[2].List().Data {
			if hdr.Type == pl.ValPair &&
				hdr.Pair.First.Type == pl.ValStr &&
				hdr.Pair.Second.Type == pl.ValStr {
				req.Header.Add(hdr.Pair.First.String(), hdr.Pair.Second.String())
			}
		}
	}

	client, err := session.GetHttpClient(url.String())
	if err != nil {
		return pl.NewValNull(), fmt.Errorf("function: http cannot create client: %s", err.Error())
	}

	resp, err := client.Do(req)
	if err != nil {
		return pl.NewValNull(), err
	}

	// serialize the response back to the normal object
	return NewResponseVal(resp), nil
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

	hdr, ok := argument[0].Usr().Context.(*Header)
	must(ok, "must be http.header")

	if vv := hdr.header.Get(argument[1].String()); vv == "" {
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

	hdr, ok := argument[0].Usr().Context.(*Header)
	must(ok, "must be http.header")

	cnt := doHeaderDelete(hdr.header, argument[1].String())
	return pl.NewValInt(cnt), nil
}
