package impl

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/dianpeng/mono-service/config"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/service"
	hrouter "github.com/julienschmidt/httprouter"

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

type bodySignService struct {
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

type bodySignSession struct {
	service *bodySignService
	r       *signResult
	op      string
	method  string
	res     service.SessionResource
}

type bodySignServiceFactory struct{}

func (b *bodySignService) Name() string {
	return "body_sign"
}

func (b *bodySignService) IDL() string {
	return ""
}

func (b *bodySignService) NewSession() (service.Session, error) {
	session := &bodySignSession{
		service: b,
	}
	return session, nil
}

func (s *bodySignSession) OnLoadVar(_ int, _ *pl.Evaluator, name string) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("module(body_sign): unknown variable %s", name)
}

func (s *bodySignSession) OnStoreVar(_ int, _ *pl.Evaluator, name string, _ pl.Val) error {
	return fmt.Errorf("module(body_sign): unknown variable %s for storing", name)
}

func (s *bodySignSession) OnCall(_ int, _ *pl.Evaluator, name string, _ []pl.Val) (pl.Val, error) {
	return pl.NewValNull(), fmt.Errorf("module(body_sign): unknown function %s", name)
}

func (s *bodySignSession) OnAction(_ int, _ *pl.Evaluator, name string, _ pl.Val) error {
	return fmt.Errorf("module(body_sign): unknown action %s", name)
}

func (b *bodySignSession) Service() service.Service {
	return b.service
}

func (b *bodySignSession) reset() {
	if b.r.body != nil {
		name := b.r.body.Name()
		b.r.body.Close()
		os.Remove(name)
	}
	b.r = nil
	b.op = ""
	b.method = ""
}

func (b *bodySignSession) Prepare(req *http.Request, p hrouter.Params) (interface{}, error) {
	op := p.ByName("op")
	if op == "" {
		xx := req.Header.Get(b.service.opHeaderName)
		if xx == "" {
			return nil, fmt.Errorf("body_sign op parameter is not specified")
		}
	}

	method := p.ByName("method")
	if method == "" {
		xx := req.Header.Get(b.service.methodHeaderName)
		if xx == "" {
			return nil, fmt.Errorf("body_sign method parameter is not specified")
		}
	}

	expect := ""

	if op == "verify" {
		expect = req.Header.Get(b.service.verifyHeaderName)
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

func (b *bodySignSession) Start(r service.SessionResource) error {
	b.res = r
	return nil
}

func (b *bodySignSession) Done(_ interface{}) {
	b.reset()
	b.res = nil
}

func (b *bodySignSession) Accept(ctx interface{}) (service.SessionResult, error) {
	param, ok := ctx.(*bodySignInput)
	if !ok {
		return service.SessionResult{},
			fmt.Errorf("module(body_sign): input context parameter invalid")
	}

	switch param.op {
	case "sign":
		if err := b.sign(param.body, param.method); err != nil {
			return service.SessionResult{}, err
		}
		break

	case "verify":
		if err := b.verify(param.body, param.expect, param.method); err != nil {
			return service.SessionResult{}, err
		}
		break

	default:
		return service.SessionResult{}, fmt.Errorf("invalid operation %s", param.op)
	}

	var selector string

	if param.op == "sign" {
		selector = "sign"
	} else {
		if b.r.sign == b.r.result {
			selector = "pass"
		} else {
			selector = "reject"
		}
	}

	output := []pl.DynamicVariable{
		pl.DynamicVariable{
			Key:   "signMethod",
			Value: pl.NewValStr(param.method),
		},
		pl.DynamicVariable{
			Key:   "signOp",
			Value: pl.NewValStr(param.op),
		},
		pl.DynamicVariable{
			Key:   "sign",
			Value: pl.NewValStr(b.r.result),
		},
		pl.DynamicVariable{
			Key:   "signExpect",
			Value: pl.NewValStr(b.r.sign),
		},
		pl.DynamicVariable{
			Key:   "signBody",
			Value: hpl.NewHplHttpBodyValFromStream(b.r.body),
		},
	}

	return service.SessionResult{
		Event: selector,
		Vars:  output,
	}, nil
}

func (b *bodySignSession) hashBody(data io.Reader, method string) (*signResult, error) {
	file, err := os.CreateTemp(b.service.tempDir, b.service.signPrefix)
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

func (b *bodySignSession) newHasher(method string) (hash.Hash, error) {
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
func (b *bodySignSession) sign(data io.Reader, method string) error {
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
func (b *bodySignSession) verify(data io.Reader, expect string, method string) error {
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

func (b *bodySignSession) SessionResource() service.SessionResource {
	return b.res
}

func (b *bodySignServiceFactory) Name() string {
	return "body_sign"
}

func (b *bodySignServiceFactory) IDL() string {
	return ""
}

func (b *bodySignServiceFactory) Comment() string {
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


Additionally, it exposes following policy variable for user to use

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

func (b *bodySignServiceFactory) Create(config *config.Service) (service.Service, error) {
	svc := &bodySignService{
		tempDir:          config.GetConfigStringDefault("temp_dir", defTempDir),
		signPrefix:       config.GetConfigStringDefault("sign_prefix", defSignPrefix),
		verifyPrefix:     config.GetConfigStringDefault("verify_prefix", defVerifyPrefix),
		verifyHeaderName: config.GetConfigStringDefault("verify_header_name", defVerifyHeaderName),
		opHeaderName:     config.GetConfigStringDefault("op_header_name", defOpHeaderName),
		methodHeaderName: config.GetConfigStringDefault("method_header_name", defMethodHeaderName),
	}
	return svc, nil
}

func init() {
	service.RegisterServiceFactory(&bodySignServiceFactory{})
}
