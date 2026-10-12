package ctxdata

import (
	"github.com/golang-jwt/jwt/v4"
)

const IdentityKey = "peninsula"

// GetJwtToken 生成 JWT 令牌
func GetJwtToken(secretKey string, iat, seconds int64, uid string) (string, error) {
	claims := make(jwt.MapClaims)
	claims["exp"] = iat + seconds // 过期时间 = 签发时间 + 有效期
	claims["iat"] = iat           // 签发时间
	claims[IdentityKey] = uid     // 用户 ID

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secretKey))
}
