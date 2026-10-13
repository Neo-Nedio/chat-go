package ctxdata

import "context"

// GetUId Getuid 从 context 取用户 ID
// 用途：在请求处理链里，中间件（拦截器）会先解析 JWT，把 uid 塞进 context
func GetUId(ctx context.Context) string {
	if u, ok := ctx.Value(Identify).(string); ok {
		return u
	}
	return ""
}
