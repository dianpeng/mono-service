package application

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/hrouter"
	"github.com/dianpeng/mono-service/http/framework"
	"github.com/dianpeng/mono-service/pl"

	// crypto
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"golang.org/x/crypto/md4"
	"hash"
)

// default configuration
const (
	defTempDir          = "/tmp"
	defSignPrefix       = "BODY-SIGN-SIGN-"
	defVerifyPrefix     = "BODY-SIGN-VERIFY-"
	defVerifyHeaderName = "x-body-sign-verify-expect"
	defOpHeaderName     = "x-body-sign-op"
	defMethodHeaderName = "x-body-sign-digest"
)

const (
	algoMd4 = iota
	algoMd5
	algoSha1
	algoSha256
	algoSha224
	algoSha384
	algoSha512
)

type signResult struct {
	contentLength int64
	sign          string
	result        string
	body          *os.File
}

type bodySignConfig struct {
	tempDir          string
	signPrefix       string
	verifyPrefix     string
	verifyHeaderName string
	opHeaderName     string
	methodHeaderName string
}

// prepare's returned result
type bodySignInput struct {
	op     string
	method string
	expect string
	body   io.Reader
}

type bodySignApplication struct {
	r      *signResult
	op     string
	method string
	args   []pl.Val
	config bodySignConfig
}

type bodySignFactory struct{}

func (b *bodySignApplication) reset() {
	if b.r.body != nil {
		name := b.r.body.Name()
		b.r.body.Close()
		os.Remove(name)
	}
	b.r = nil
	b.op = ""
	b.method = ""
}

func (b *bodySignApplication) Prepare(req *http.Request, p hrouter.Params) (interface{}, error) {
	op := p.ByName("op")
	if op == "" {
		xx := req.Header.Get(b.config.opHeaderName)
		if xx == "" {
			return nil, fmt.Errorf("body_sign op parameter is not specified")
		}
	}

	method := p.ByName("method")
	if method == "" {
		xx := req.Header.Get(b.config.methodHeaderName)
		if xx == "" {
			return nil, fmt.Errorf("body_sign method parameter is not specified")
		}
	}

	expect := ""

	if op == "verify" {
		expect = req.Header.Get(b.config.verifyHeaderName)
		if expect == "" {
			return nil, fmt.Errorf("verification result header is not set")
		}
	}

	return &bodySignInput{
		op:     op,
		method: method,
		expect: expect,
		body:   req.Body,
	}, nil
}

func (b *bodySignApplication) Done(_ interface{}) {
	b.reset()
}

func (b *bodySignApplication) prepareConfig(
	context framework.ServiceContext,
) error {
	cfg := framework.NewPLConfig(
		context,
		b.args,
	)

	cfg.TryGetStr(0, &b.config.tempDir, defTempDir)
	cfg.TryGetStr(1, &b.config.signPrefix, defSignPrefix)
	cfg.TryGetStr(2, &b.config.verifyPrefix, defVerifyPrefix)
	cfg.TryGetStr(3, &b.config.verifyHeaderName, defVerifyHeaderName)
	cfg.TryGetStr(4, &b.config.opHeaderName, defOpHeaderName)
	cfg.TryGetStr(5, &b.config.methodHeaderName, defMethodHeaderName)
	return nil
}

func (b *bodySignApplication) Accept(
	ctx interface{},
	context framework.ServiceContext,
) (framework.ApplicationResult, error) {
	if err := b.prepareConfig(context); err != nil {
		return framework.ApplicationResult{}, err
	}

	param, ok := ctx.(*bodySignInput)
	if !ok {
		return framework.ApplicationResult{},
			fmt.Errorf("module(body_sign): input context parameter invalid")
	}

	switch param.op {
	case "sign":
		if err := b.sign(param.body, param.method); err != nil {
			return framework.ApplicationResult{}, err
		}
		break

	case "verify":
		if err := b.verify(param.body, param.expect, param.method); err != nil {
			return framework.ApplicationResult{}, err
		}
		break

	default:
		return framework.ApplicationResult{}, fmt.Errorf("invalid operation %s", param.op)
	}

	var selector string

	if param.op == "sign" {
		selector = "body_sign.sign"
	} else {
		if b.r.sign == b.r.result {
			selector = "body_sign.pass"
		} else {
			selector = "body_sign.reject"
		}
	}

	output := framework.NewApplicationResult(selector)

	output.AddContext(
		"signMethod",
		pl.NewValStr(param.method),
	)

	output.AddContext(
		"signOp",
		pl.NewValStr(param.op),
	)
	output.AddContext(
		"sign",
		pl.NewValStr(b.r.result),
	)
	output.AddContext(
		"signExpect",
		pl.NewValStr(b.r.sign),
	)
	output.AddContext(
		"signBody",
		hpl.NewBodyValFromStream(b.r.body),
	)

	return output, nil
}

func (b *bodySignApplication) hashBody(data io.Reader, method string) (*signResult, error) {
	file, err := os.CreateTemp(b.config.tempDir, b.config.signPrefix)
	if err != nil {
		return nil, err
	}
	teedReader := io.TeeReader(data, file)
	hasher, err := b.newHasher(method)
	if err != nil {
		return nil, err
	}

	sz, err := io.Copy(hasher, teedReader)
	if err != nil {
		return nil, err
	}

	sign := fmt.Sprintf("%x", hasher.Sum(nil))

	file.Seek(0, io.SeekStart)

	return &signResult{
		contentLength: int64(sz),
		sign:          "",
		result:        sign,
		body:          file,
	}, nil
}

func (b *bodySignApplication) newHasher(method string) (hash.Hash, error) {
	switch method {
	case "md4":
		return md4.New(), nil
	case "md5", "":
		return md5.New(), nil
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New224(), nil
	case "sha512":
		return sha512.New(), nil
	case "sha384":
		return sha512.New384(), nil
	default:
		return nil, fmt.Errorf("invalid hash method %s", method)
	}
}

// sign the incomming http request, ie generate sign result from its body and
// also create a stream by using the temporary file
func (b *bodySignApplication) sign(data io.Reader, method string) error {
	r, err := b.hashBody(data, method)
	if err != nil {
		return err
	}
	b.r = r
	b.op = "sign"
	b.method = method
	return nil
}

// verify the incomming http request, ie generate sign result from its body and
// also create a stream by using the temporary file
func (b *bodySignApplication) verify(data io.Reader, expect string, method string) error {
	r, err := b.hashBody(data, method)
	if err != nil {
		return err
	}
	b.r = r

	b.r.sign = expect
	b.op = "verify"
	b.method = method
	return nil
}

func (b *bodySignFactory) Name() string {
	return "body_sign"
}

func (b *bodySignFactory) Comment() string {
	return `
A service does http body (via request's body) digest sign and verification job.
This service allows efficient memory management of very large http body by using
temporary file system storage. Additionally, it allows different digest algorithm
as following :

1. MD4
2. MD5
3. SHA1
4. SHA256
5. SHA512
6. SHA384

Additionally, it exposes following module variable for user to use

1. signMethod, a string indicate the signing digest method
2. signOp, a string of 2 values, "sign" or "verify"
3. sign, a string of signing result in hexical string
4. signExpect, if it is a verify operation, then the expect field will be
               stored here

5. signBody, a duplicated stream of body of request. Notes after the sign
             the original request body is not usable anymore, ie it is empty
             since it has been consumed up

With this service, user can echo the request body back regardlessly however
large the request body is. For example it can sign a 4gb request body and
having no problem at all

`
}

func (b *bodySignFactory) Create(args []pl.Val) (framework.Application, error) {
	return &bodySignApplication{
		args: args,
	}, nil
}

func init() {
	framework.AddApplicationFactory("body_sign", &bodySignFactory{})
}
