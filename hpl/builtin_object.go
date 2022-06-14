package hpl

import (
	"fmt"
	"io"
	"net/url"

	// router
	"github.com/dianpeng/mono-service/pl"

	// library usage
	"strings"
)

// create an empty url for manipulation usage
func fnNewUrl(info *pl.IntrinsicInfo,
	_ *pl.Evaluator,
	_ string,
	argument []pl.Val,
) (pl.Val, error) {
	alog, err := info.Check(argument)
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

func fnNewRequest(info *pl.IntrinsicInfo,
	_ *pl.Evaluator,
	_ string,
	argument []pl.Val,
) (pl.Val, error) {
	alog, err := info.Check(argument)
	if err != nil {
		return pl.NewValNull(), err
	}

	switch alog {
	case 2:
		return NewRequestValFromStream(argument[0].String(), argument[1].String(), &eofReadCloser{})

	default:
		return NewRequestValFromVal(argument[0].String(), argument[1].String(), argument[2])
	}
}

func fnNewHeader(info *pl.IntrinsicInfo,
	_ *pl.Evaluator,
	_ string,
	argument []pl.Val,
) (pl.Val, error) {
	_, err := info.Check(argument)
	if err != nil {
		return pl.NewValNull(), err
	}

	hdr, err := NewHeaderValFromVal(argument[0])
	if err != nil {
		return pl.NewValNull(), fmt.Errorf("http::new_header cannot create header: %s", err.Error())
	}
	return hdr, nil
}

func fnConcateHttpBody(info *pl.IntrinsicInfo,
	_ *pl.Evaluator,
	_ string,
	argument []pl.Val,
) (pl.Val, error) {

	_, err := info.Check(argument)
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

func init() {
	pl.AddModFunction(
		"http",
		"new_url",
		"",
		"{%0}{%s}",
		fnNewUrl,
	)

	pl.AddModFunction(
		"http",
		"new_request",
		"",
		"{%s%s}{%s%s%s}{%s%s%U['http.body']}{%s%s%U['.readablestream']}",
		fnNewRequest,
	)

	pl.AddModFunction(
		"http",
		"new_header",
		"",
		"{%p}{%l}{%m}{%U['http.header']}",
		fnNewHeader,
	)

	pl.AddModFunction(
		"http",
		"concate_body",
		"",
		"%a*",
		fnConcateHttpBody,
	)
}
