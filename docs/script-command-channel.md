# 命令面协议（中控客户端合同）

sun-controller 经 `adb forward` 连到本进程的 WebSocket。帧是 JSON 文本。本文件是词表的唯一出处；中控仓引用，不另写一份。

## 监听

进程一进入 `main` 的命令循环之前就开始听 `127.0.0.1:27183`（避开 AutoGo 外壳 `8989`）。只接受本机连接。无鉴权。路径 `/`。

## 帧

连上后脚本先发：

```json
{"type":"hello","v":1,"phase":"<phase>"}
```

`v` 必须为 `1`。`phase` 为宿主当前运行相位，取值与面板一致：`idle`、`running`、`paused`、`waiting`。中控把 `waiting` 显示成空闲。

之后脚本在相位变化时推：

```json
{"type":"status","phase":"<phase>"}
```

中控发命令（`id` 为本次调用的非空字符串）：

```json
{"id":"<id>","type":"engine.start"}
{"id":"<id>","type":"engine.pause"}
{"id":"<id>","type":"engine.resume"}
{"id":"<id>","type":"engine.stop"}
```

第一刀只接受以上四种 `type`。脚本把它们交给现有 `Host.Handle`（与控制面板、悬浮胶囊同一条路）。`engine.start` 不带面板草稿：吃设备上已存配置 / 默认值（与胶囊在 idle 时「开始」相同）。

回复：

```json
{"id":"<id>","ok":true}
{"id":"<id>","ok":false,"error":"<message>"}
```

`engine.stop` 仍退出进程。允许先回 `ok` 再断开，也允许不回、只关连接。中控以连接断开为停止完成的充分条件。

未知 `type` 回 `ok: false`。忽略 `config.save`、诊断等第一刀没有的命令。

## 实现约束

- 用 AutoGo 快速调试已白名单的 `gorilla/websocket` 听端口。
- 不要经 AutoGo `:8989/task` 冒充命令面。
- 控制面板第一刀保留；命令面是第三条 Command 来源，不是替换面板。
