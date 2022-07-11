package ext

import (
	"fmt"
	"github.com/dianpeng/mono-service/http/framework"
	"github.com/dianpeng/mono-service/pl"
	"github.com/dianpeng/mono-service/util"
	"net/http"
)

func foreachHeaderKV(
	context string,
	args []pl.Val,
	ctx framework.ServiceContext,
	callback func(string, string),
) error {
	size := len(args)
	cfg := framework.NewPLConfig(
		ctx,
		args,
	)

	for i := 0; i < size; i++ {
		v := pl.NewValNull()
		if err := cfg.Get(
			i,
			&v,
		); err != nil {
			return err
		}

		if !v.IsPair() {
			return fmt.Errorf("%s: the argument must be pair to represent header kv", context)
		}

		key, err := v.Pair().First.ToString()
		if err != nil {
			return fmt.Errorf("%s: key is not string, %s", context, err.Error())
		}
		val, err := v.Pair().Second.ToString()
		if err != nil {
			return fmt.Errorf("%s: value is not string, %s", context, err.Error())
		}

		callback(key, val)
	}

	return nil
}

func foreachHeaderKey(
	context string,
	args []pl.Val,
	ctx framework.ServiceContext,
	callback func(string),
) error {
	size := len(args)
	cfg := framework.NewPLConfig(
		ctx,
		args,
	)

	for i := 0; i < size; i++ {
		v := pl.NewValNull()
		if err := cfg.Get(
			i,
			&v,
		); err != nil {
			return err
		}

		key, err := v.ToString()
		if err != nil {
			return fmt.Errorf("%s: key is not string, %s", context, err.Error())
		}

		callback(key)
	}

	return nil
}

func HeaderAdd(
	context string,
	args []pl.Val,
	header http.Header,
	ctx framework.ServiceContext,
) error {
	return foreachHeaderKV(
		context,
		args,
		ctx,
		func(key string, val string) {
			header.Add(key, val)
		},
	)
}

func HeaderSet(
	context string,
	args []pl.Val,
	header http.Header,
	ctx framework.ServiceContext,
) error {
	return foreachHeaderKV(
		context,
		args,
		ctx,
		func(key string, val string) {
			header.Set(key, val)
		},
	)
}

func HeaderDel(
	context string,
	args []pl.Val,
	header http.Header,
	ctx framework.ServiceContext,
) error {
	return foreachHeaderKey(
		context,
		args,
		ctx,
		func(key string) {
			m := util.ToMatcher(
				key,
			)
			for k, _ := range header {
				if m(k, key) {
					header.Del(k)
				}
			}
		},
	)
}
