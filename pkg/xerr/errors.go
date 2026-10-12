package xerr

import "github.com/zeromicro/x/errors"

//错误构造入口

func New(code int, msg string) error {
	return errors.New(code, msg)
}

func NewDBErr() error {
	return New(DbError, ErrMsg(DbError))
}

func NewServerCommonErr() error {
	return New(ServerCommonError, ErrMsg(ServerCommonError))
}
