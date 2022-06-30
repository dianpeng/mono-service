package vhost

import (
	"fmt"
	"github.com/dianpeng/mono-service/pl"
)

func propSetString(
	v pl.Val,
	ptr *string,
	name string,
) error {
	str, err := v.ToString()
	if err != nil {
		return fmt.Errorf("%s: set field error, %s", name, err.Error())
	}
	*ptr = str
	return nil
}

func propSetInt(
	v pl.Val,
	ptr *int,
	name string,
) error {
	if !v.IsInt() {
		return fmt.Errorf("%s: set field error, value is not int")
	}

	*ptr = int(v.Int())
	return nil
}

func propSetInt64(
	v pl.Val,
	ptr *int64,
	name string,
) error {
	if !v.IsInt() {
		return fmt.Errorf("%s: set field error, value is not int")
	}

	*ptr = v.Int()
	return nil
}
