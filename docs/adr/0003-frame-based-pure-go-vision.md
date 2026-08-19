# 识别基于捕获帧的纯 Go 比色，弃用 AutoGo 原生图色 API

> **已被 [ADR-0004](0004-autogo-native-color-apis.md) 取代**（识别热路径）。本文只记录曾经为了桌面回归而自研帧扫描的原因；OCR 注入接缝的例外条款仍有效。

引擎每 tick 用 `CaptureScreen` 截取一帧 `*image.NRGBA`，所有画面识别（多点比色、找色、OCR 区域）在帧上由纯 Go 算法完成，不使用 AutoGo 的 `DetectsMultiColors`/`CmpColor`/`FindColor` 实时屏幕 API。

原因：游戏脚本的价值几乎全在识别数据上，迁移后几百个色点必须在设备上重新验证；帧分析让全部识别逻辑与特征数据可在桌面 `go test` 回归（喂保存截图断言命中/不命中），验证循环从"改代码→部署→设备跑"变为秒级本地循环。色点参数格式（`"x|y|color-偏移"` + 相似度）与图色工具产出、Lua 特征库、AutoGo API 三方完全一致，零转换复用。代价是自持约几百行比色算法（偏色按通道 ±0x10 容差、逐色点相似度比较），auto-cookie 的 `internal/vision` 已有同格式参考实现。cr-auto 面板的截图隐身与识别诊断页正是为此类帧分析设计的基础设施。

## 例外：OCR 识别引擎走注入接缝（M2a 起）

Lua 工程的 OCR（`lib/ocr.lua`）通过 TomatoOCR 引擎对区域截图做文字识别，本质是引擎调用而非纯算法；按结构直译（ADR-0002），`internal/lib/ocr` 保持同形门面，但引擎定义为可注入接口（`ocr.Engine`：按区域识别并返回原始结果串）。桌面测试注入假引擎即可回归全部 OCR 解析逻辑；设备端引擎（TomatoOCR 等价实现）在 M2 设备验收阶段由 main 注入。此例外不影响本 ADR 的主体——所有比色/找色仍在帧上由纯 Go 完成；OCR 仅是文字识别通道，与帧分析并存。
