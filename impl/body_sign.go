package impl

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/dianpeng/mono-service/alog"
	"github.com/dianpeng/mono-service/config"
	"github.com/dianpeng/mono-service/hpl"
	"github.com/dianpeng/mono-service/service"
	"github.com/dianpeng/mono-service/util"
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
	config           *config.Service
	tempDir          string
	signPrefix       string
	verifyPrefix     string
	verifyHeaderName string
	opHeaderName     string
	methodHeaderName string
	policy           *hpl.Policy
}

type bodySignSession struct {
	hpl     *hpl.Hpl
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

func (b *bodySignService) Tag() string {
	return b.config.Tag
}

func (b *bodySignService) IDL() string {
	return ""
}

func (b *bodySignService) Policy() string {
	return b.config.Policy
}

func (b *bodySignService) PolicyDump() string {
	return b.policy.Dump()
}

func (b *bodySignService) Router() string {
	return b.config.Router
}

func (b *bodySignService) MethodList() []string {
	return b.config.Method
}

func (b *bodySignService) NewSession() (service.Session, error) {
	session := &bodySignSession{
		service: b,
	}
	session.hpl = hpl.NewHplWithPolicy(session.hplLoadVar, nil, nil, b.policy)
	return session, nil
}

func (b *bodySignSession) hplLoadVar(x *hpl.Evaluator, name string) (hpl.Val, error) {
	switch name {
	case "signMethod":
		return hpl.NewValStr(b.method), nil
	case "signOp":
		return hpl.NewValStr(b.op), nil
	case "sign":
		return hpl.NewValStr(b.r.result), nil
	case "signExpect":
		return hpl.NewValStr(b.r.sign), nil
	case "reqBody":
		return hpl.NewHplHttpBodyValFromStream(b.r.body), nil
	default:
		return hpl.NewValNull(), fmt.Errorf("invalid variable: %s", name)
	}
}

func (b *bodySignSession) Service() service.Service {
	return b.service
}

func (b *bodySignSession) reset() {
	b.r = nil
	b.op = ""
	b.method = ""
}

// getting out the method and op arguments
func (b *bodySignSession) getParam(req *http.Request, p hrouter.Params) (string, string, error) {
	op := p.ByName("op")
	if op == "" {
		xx := req.Header.Get(b.service.opHeaderName)
		if xx == "" {
			return "", "", fmt.Errorf("body_sign op parameter is not specified")
		}
	}

	method := p.ByName("method")
	if method == "" {
		xx := req.Header.Get(b.service.methodHeaderName)
		if xx == "" {
			return "", "", fmt.Errorf("body_sign method parameter is not specified")
		}
	}

	return op, method, nil
}

func (b *bodySignSession) Start(r service.SessionResource) error {
	b.res = r
	return b.hpl.OnGlobal(b)
}

func (b *bodySignSession) Done() {
	b.res = nil
}

func (b *bodySignSession) SessionResource() service.SessionResource {
	return b.res
}

func (b *bodySignSession) Accept(w http.ResponseWriter, req *http.Request, p hrouter.Params) {
	b.reset()
	defer func() {
		if b.r.body != nil {
			name := b.r.body.Name()
			b.r.body.Close()
			os.Remove(name)
		}
	}()

	op := p.ByName("op")
	method := p.ByName("method")

	op, method, err := b.getParam(req, p)
	if err != nil {
		util.ErrorRequest(w, fmt.Errorf("invalid operation %s", op))
		return
	}

	switch op {
	case "sign":
		err := b.sign(req.Body, method)
		if err != nil {
			util.ErrorRequest(w, err)
			return
		}
		break

	case "verify":
		err := b.verify(req.Body, req.Header, method)
		if err != nil {
			util.ErrorRequest(w, err)
			return
		}
		break

	default:
		util.ErrorRequest(w, fmt.Errorf("invalid operation %s", op))
		return
	}

	var selector string

	if op == "sign" {
		selector = "sign"
	} else {
		if b.r.sign == b.r.result {
			selector = "pass"
		} else {
			selector = "reject"
		}
	}

	err = b.hpl.OnHttpResponse(
		selector,
		hpl.HttpContext{
			Request:        req,
			ResponseWriter: w,
			QueryParams:    p,
		},
		b,
	)

	if err != nil {
		util.ErrorRequest(w, fmt.Errorf("HPL execution error: %s", err.Error()))
		return
	}
}

func (b *bodySignSession) Log(log *alog.SessionLog) error {
	return b.hpl.OnLog("log", log, b)
}

func (b *bodySignSession) Allow(_ *http.Request, _ hrouter.Params) error {
	return nil
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
func (b *bodySignSession) verify(data io.Reader, hdr http.Header, method string) error {
	r, err := b.hashBody(data, method)
	if err != nil {
		return err
	}
	b.r = r

	expect := hdr.Get(b.service.verifyHeaderName)
	if expect == "" {
		return fmt.Errorf("verification result header is not set")
	}

	b.r.sign = expect
	b.op = "verify"
	b.method = method
	return nil
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

5. reqBody, a duplicated stream of body of request. Notes after the sign
            the original request body is not usable anymore, ie it is empty
            since it has been consumed up

With this service, user can echo the request body back regardlessly however
large the request body is. For example it can sign a 4gb request body and
having no problem at all

`
}

func (b *bodySignServiceFactory) Create(config *config.Service) (service.Service, error) {
	p, err := hpl.CompilePolicy(config.Policy)
	if err != nil {
		return nil, err
	}
	svc := &bodySignService{
		config:           config,
		tempDir:          config.GetConfigStringDefault("temp_dir", defTempDir),
		signPrefix:       config.GetConfigStringDefault("sign_prefix", defSignPrefix),
		verifyPrefix:     config.GetConfigStringDefault("verify_prefix", defVerifyPrefix),
		verifyHeaderName: config.GetConfigStringDefault("verify_header_name", defVerifyHeaderName),
		opHeaderName:     config.GetConfigStringDefault("op_header_name", defOpHeaderName),
		methodHeaderName: config.GetConfigStringDefault("method_header_name", defMethodHeaderName),
		policy:           p,
	}
	return svc, nil
}

func init() {
	service.RegisterServiceFactory(&bodySignServiceFactory{})
}
