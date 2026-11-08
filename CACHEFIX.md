# 补丁：压缩请求补 cache_control

问题：Claude Code 2.1.319 的压缩请求，消息里不带任何 cache_control（只有 system 部分带）。
CPA 不会给原生 Claude Code 请求补标记，部分上游中转站处理这种冷的大请求时会返回 429/503 "Service Unavailable"。

修复：对原生 Claude Code 请求，如果消息里一个 cache_control 都没有，就在第一条用户消息的最后一个块和最后一条可用消息上各补一个 {type: ephemeral}。消息里已经带标记的请求不受影响。

升级到新版本时：
1. git rebase <新版本标签>
2. CGO_ENABLED=1 go build -buildvcs=false -ldflags="-s -w -X 'main.Version=<版本>-cachefix'" -o CLIProxyAPI ./cmd/server/
