package ctxdata

import (
	"github.com/golang-jwt/jwt/v4"
)

const Identify = "12344321"

// GetJwtToken 生成 JWT 令牌
func GetJwtToken(secretKey string, iat, seconds int64, uid string) (string, error) {
	claims := make(jwt.MapClaims)
	claims["exp"] = iat + seconds
	claims["iat"] = iat
	claims[Identify] = uid

	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims

	return token.SignedString([]byte(secretKey))
}
