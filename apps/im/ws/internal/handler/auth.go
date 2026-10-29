package handler

import (
	"chat-go/apps/im/ws/internal/svc"
	"chat-go/pkg/ctxdata"
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/token"
)

type JwtAuth struct {
	svc    *svc.ServiceContext
	parser *token.TokenParser
	logx.Logger
}

func NewJwtAuth(svc *svc.ServiceContext) *JwtAuth {
	return &JwtAuth{
		svc:    svc,
		parser: token.NewTokenParser(),
		Logger: logx.WithContext(context.Background()),
	}
}
func (j *JwtAuth) Auth(w http.ResponseWriter, r *http.Request) bool {
	//兼容浏览器 WebSocket 的 token 传递
	if tok := r.Header.Get("sec-websocket-protocol"); tok != "" {
		r.Header.Set("Authorization", tok)
	}
	//解析 token
	tok, err := j.parser.ParseToken(r, j.svc.Config.JwtAuth.AccessSecret, "")
	if err != nil {
		return false

	}
	if !tok.Valid {
		return false
	}
	//取 claims
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}

	//把用户 ID 塞进 context
	*r = *r.WithContext(context.WithValue(r.Context(), ctxdata.Identify, claims[ctxdata.Identify]))
	return true
}

func (j *JwtAuth) UserId(r *http.Request) string {
	return ctxdata.GetUId(r.Context())
}
