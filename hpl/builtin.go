package hpl

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	// router
	"github.com/dianpeng/mono-service/pl"

	// library usage
	"strings"
)

var (
	fnProtoNewUrl          = pl.MustNewModFuncProto("http", "new_url", "{%0}{%s}")
	fnProtoConcateHttpBody = pl.MustNewModFuncProto("http", "concate_body", "%a*")
	fnProtoDo              = pl.MustNewModFuncProto("http", "do",
		"{%s%s}{%s%s%l}{%s%s%a%s}{%s%s%a%U['http.body']}")
)

// create an empty url for manipulation usage
func fnNewUrl(argument []pl.Val) (pl.Val, error) {
	alog, err := fnProtoNewUrl.Check(argument)
	if err != nil {
		return pl.NewValNull(), err
	}

	switch alog {
	case 1:
		rawUrl, err := url.Parse(argument[0].String())
		if err != nil {
			return pl.NewValNull(), fmt.Errorf("http::new_url cannot create url: %s", err.Error())
		}
		return NewUrlVal(rawUrl), nil

	default:
		return NewUrlVal(&url.URL{}), nil
	}
}

func fnConcateHttpBody(argument []pl.Val) (pl.Val, error) {
	_, err := fnProtoConcateHttpBody.Check(argument)
	if err != nil {
		return pl.NewValNull(), err
	}

	var input []io.Reader

	for idx, a := range argument {
		if ValIsHttpBody(a) {
			body, _ := a.Usr().Context.(*Body)
			input = append(input, body.Stream().Stream)
		} else {
			str, err := a.ToString()
			if err != nil {
				return pl.NewValNull(),
					fmt.Errorf("http::concate_body: the %d argument cannot be converted to string: %s",
						idx+1, err.Error())
			}
			input = append(input, strings.NewReader(str))
		}
	}

	return NewBodyValFromStream(neweofByteReadCloserFromStream(io.MultiReader(input...))), nil
}

// general HTTP request interfaces, allowing user to specify following method
// http(
//   [0]: url(string)
//   [1]: method(string)
//   [2]: [header](list[string]),
//   [3]: [body](string)
// )
func fnDoHttp(session SessionWrapper, argument []pl.Val) (pl.Val, error) {
	alog, err := fnProtoDo.Check(argument)
	if err != nil {
		return pl.NewValNull(), err
	}

	url := argument[0].String()
	method := argument[1].String()

	var body io.Reader
	body = http.NoBody

	if alog == 4 {
		switch argument[3].Type {
		case pl.ValStr:
			body = strings.NewReader(argument[3].String())
			break

		default:
			must(ValIsHttpBody(argument[3]), "must be http.body")
			b, _ := argument[3].Usr().Context.(*Body)
			body = b.Stream().Stream
			break
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return pl.NewValNull(), fmt.Errorf("http::do cannot create request: %s", err.Error())
	}

	// check header field
	if alog >= 3 && argument[2].Type == pl.ValList {
		for _, hdr := range argument[2].List().Data {
			if hdr.Type == pl.ValPair &&
				hdr.Pair.First.Type == pl.ValStr &&
				hdr.Pair.Second.Type == pl.ValStr {
				req.Header.Add(hdr.Pair.First.String(), hdr.Pair.Second.String())
			}
		}
	}

	client, err := session.GetHttpClient(url)
	if err != nil {
		return pl.NewValNull(), fmt.Errorf("http::do cannot create client: %s", err.Error())
	}

	resp, err := client.Do(req)
	if err != nil {
		return pl.NewValNull(), fmt.Errorf("http::do cannot issue request: %s", err.Error())
	}

	// serialize the response back to the normal object
	return NewResponseVal(resp), nil
}
