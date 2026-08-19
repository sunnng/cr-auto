# 识别热路径使用 AutoGo 原生图色 API

运行时比色/找色通过 `internal/lib/color` 的 `Screen` 接缝调用 AutoGo `images` API：`DetectsMultiColors`、`CmpColor`、`FindMultiColors`、`FindMultiColorsAll`。设备端由 `main` 注入带引用计数截图隐身的适配器。`internal/vision` 只保留特征库格式与 AutoGo 色串转换，不再自研像素扫描。

取代 [ADR-0003](0003-frame-based-pure-go-vision.md) 对识别热路径的决策。截图仍用于诊断预览/存档，不作为比色输入。OCR 仍走独立注入引擎（ADR-0003 的例外条款继续有效）。

原因：自研 `Match` 把 `Feature.Sim` 当成色点命中比例，AutoGo/`sim` 是颜色相似度；找色区域在 Go `image.Rect` 上是开区间，Lua/AutoGo 是闭区间。对齐原生 API 才能与图色工具、Lua 特征库同一套语义。桌面测试注入 `ScriptedScreen`，不维护第二套扫描算法。
