# cachefix 补丁说明

基于官方 CLIProxyAPI，针对 Claude Code 经部分上游中转站转发时出现的报错。每个补丁按「报错现象 → 原因 → 解决方法」记录。

## 补丁 1：压缩请求补 cache_control

**报错现象**
- 自动压缩或手动 `/compact` 失败，等待 15–50 秒后返回：
  `HTTP 429 {"error":{"message":"Service Unavailable","type":"error"},"type":"error"}`（有时是 503）
- Claude Code 会反复重试，会话一直卡在压缩上。
- 缓存还热时偶尔能成功；会话闲置几分钟后（缓存过期），再压缩几乎必然失败。

**原因**
- Claude Code 2.1.319 的压缩请求，消息部分不带任何 `cache_control`，只有 system 部分带。
- CPA 对原生 Claude Code 请求不补缓存标记（原样转发）。
- 部分上游中转站处理这种不带消息缓存标记、十几万 token 的冷请求时，会返回 429/503。

**解决方法**
- 原生 Claude Code 请求如果消息里一个 `cache_control` 都没有，就在第一条用户消息的最后一个块和最后一条可用消息上各补一个 `{type: ephemeral}`。两处都要补，只补一处仍会失败。
- 消息里已经带标记的请求不受影响。
- 验证：同一个失败的压缩请求补标记后返回 200；冷缓存下约 18 万、28 万 token 的压缩也都能成功。

## 补丁 2：删除空的 system 消息

**报错现象**
- 手动 `/compact` 之后，这个会话的每个请求都失败，哪怕只有几万 token，每次约 17 秒后返回：
  `HTTP 429 {"error":{"message":"Service Unavailable","type":"error"},"type":"error"}`
- CPA 重试 3 次都失败，会话无法继续。

**原因**
- 手动 `/compact` 之后，Claude Code 2.1.319 会发出一条 content 为空数组的 system 消息：
  `{"role":"system","content":[],"output_config":{"effort":"max"}}`
- 部分上游中转站遇到这条空消息，整个请求返回 429。之后这个会话的每个请求都带着它，所以一直失败。

**解决方法**
- 转发前删除内容为空的 system 消息（空数组、空字符串或没有 content）。
- 验证：原样发送返回 429，只删掉这条空消息返回 200。

## 排查方法

- 失败请求的完整内容在 CPA 日志目录的 `error-v1-messages-*.log`。这些文件会很快被删除，要马上复制出来。
- 取 `=== API REQUEST 1 ===` 后面的请求体直接发给上游，逐项删改后重发，对比哪个版本成功、哪个失败。

## 升级到新版本时

1. `git rebase <新版本标签>`
2. `CGO_ENABLED=1 go build -buildvcs=false -ldflags="-s -w -X 'main.Version=<版本>-cachefix'" -o CLIProxyAPI ./cmd/server/`
