# CHANGELOG-ZETA

## 用途

這份文件記錄 `new-api` 私有 fork 的 ZETA 特有 Git 歷史。技術帳本保留已釋出、僅備份與未釋出的確切 OID；可讀摘要則說明每個 release epoch 中首次進入正式釋出圖的行為與維護變更。上游可達的原始物件全部排除，等價重建但 OID 不同的備份提交仍各自保留。

文件分為兩層：可讀摘要依版本由新到舊整理變更，後面的技術帳本逐筆記錄 OID、雙時間戳與來源證據，使治理規則 17 剪除舊版本分支後，版本歸屬仍可由本快照重現。

## 快照邊界

本文件涵蓋 2026-09-02 產出的 157 個 ZETA commit OID：從當時本地與 origin 的 `release/**`、`dev/**`、`feat/**`、`fix/**` 聯集，先排除下文兩條 Task 4 失敗封存分支及其 26 個 OID，再扣除全部 686 個已驗證 tag、`upstream/HEAD`、`upstream/main` 與 `origin/main` 可達的上游物件。

**快照不含最終 `docs: refresh zeta changelog` 提交，也不含其後的任何提交。** 產出時 signed nullable-priority release merge `50f0f0d13f09b08601bc15ce2e40bf989e351e3e` 是 rc.30 釋出上界，development 尖端 `f9f1a465e7572569a7392d79024697fdb5e7a1e7` 已完整包含於該 merge。最終 changelog 提交落地後兩個本地 ref 都會前進，因此拿 live ref 重跑完整性驗證時，該最終提交會被算成 missing，這是快照邊界的預期差異。要重現這 157 筆，請以 `50f0f0d13f09b08601bc15ce2e40bf989e351e3e` 為 rc.30 的釋出上界，並納入下列主題備份 tips。

另有兩條 Task 4 失敗嘗試的封存分支（訊息格式錯誤與 trailer 重複各 13 個 OID，共 26 個）被明確排除，與帳本的交集為 0，不視為 ZETA 歷史。rc.26 與 rc.27 因為舊 release ref 已被剪除，版本歸屬改用明文記錄的 epoch 邊界判定，詳見技術帳本的參考區塊。

## 狀態與分類圖例

**狀態**分三種。`已釋出`指該 OID 位於本快照的 canonical ZETA release graph 祖先鏈；這是版本歸屬證據，不代表每個後續版本都曾部署到生產環境。`僅備份`指該 OID 只存在於 `feat/v<version>/**` 或 `fix/v<version>/**` 主題備份分支，是相同行為在對應上游 tag 上建立的可重用副本，行為可能已由另一個 OID 釋出。`未釋出`指停留在未合併主題分支上的工作。

**分類**用來標示提交性質：`Feature`、`Fix`、`Test`、`Docs`、`Governance`、`Integration`（釋出合併或上游 tag 合併）、`Carry snapshot`（跨版本搬移時以 range-diff 或 tree 比對證明等價的整批快照）、`Backup rebuild`（帶有 `cherry picked from` trailer 或 patch-id 相符的逐筆重建）。`Carry snapshot` 與 `Backup rebuild` 的差別來自等價性證據的種類，而不是物件本身有什麼不同。

---

## v1.0.0-rc.33

Astra requests with omitted effort no longer inherit an unsupported `none`
from shared `keep_origin` channel defaults. Explicit effort values remain
subject to model validation; bare Astra passed real Chat and Responses calls.

Integrates upstream rc.33 reasoning preservation and Astra expression pricing.
Restores native tools supplied through existing Chat parameter overrides and
adapts structured `reasoning_effort` override paths to Responses while preserving
`keep_origin`. The original channel override passed real upstream curl requests
for Luna `low` through both APIs and Luna `pro-max` through Responses.
See the [implementation and validation record](docs/releases/2026-09-05-rc33-openai-compat.md).

## v1.0.0-rc.32

Model-family discovery now supports base-only channel registration and pricing for GPT-5.6 Sol/Terra/Luna and GPT-6 Astra. Client model lists expand valid aliases automatically; explicit pricing overrides remain intact. See the [implementation and validation record](docs/releases/2026-09-05-model-family-discovery.md).

本版為 GPT-6 Astra 加入模型別名支援。呼叫格式為 `gpt-6-astra[-standard|-pro][-low|-medium|-high|-xhigh|-max]`，例如 `gpt-6-astra-pro-max` 代表 pro 模式搭配 max 推理強度。

| Item | Evidence |
| --- | --- |
| Upstream integration | `b549c767674cb4924c9173b48583c3a7e5a710e7` |
| Astra feature | `f68c8fad4c62929347cb8ac881fd20ebf1527076` |
| Signed release merge | `dae0c9b169c1148a3621410b65b6e91a9e24c973` |
| Verification and five themed backups | [rc.32 release report](docs/releases/2026-09-05-rc32-astra.md) |
| Deployment | Published to Dev and Production at `2a673c04b2f2b22b85b700c188c5f5e5fc02e2be`; version and startup verified on 2026-09-05 |

This entry is a new release-candidate appendix. The 157-OID historical ledger and its 2026-09-02 snapshot boundary below remain unchanged.

## v1.0.0-rc.30

本版把整合釋出從 rc.29 推進到上游 `v1.0.0-rc.30`，並在合併時保住兩處與上游重疊的自訂行為：`model/log.go` 的用量日誌自訂邏輯，以及 `model/main.go` 的 PostgreSQL migrator 包裝層。上游新增的 `tokens.key` 舊唯一性遷移在 GORM `AutoMigrate` 之前執行，ZETA 的 dialector 包裝則在 `AutoMigrate` 期間持續生效，兩者並存。

PostgreSQL prefill 唯一性採兩階段發布。Bridge 先保留舊版辨識得到的 global unique object，建立有效的 partial unique index，並把 migration lock wait 限定為 `5s`、statement timeout 限定為 `30s`，使 rc.26 snapshot 仍可回復；Contract 再只移除 allowlist 內的舊物件，完成 soft-delete name reuse。無效或未 ready 的舊索引會在資料異動前被拒絕。

- 修正 nullable channel priority 選路 — [50f0f0d13f09](../../commit/50f0f0d13f09b08601bc15ce2e40bf989e351e3e)
- 實作 `NULL` 與明示零的相同 priority 語意 — [f9f1a465e757](../../commit/f9f1a465e7572569a7392d79024697fdb5e7a1e7)
- 完成 prefill uniqueness Contract — [85da814a0d8f](../../commit/85da814a0d8f0d7a10e6d479017aabcf3c829e88)
- 實作 Contract 並保留 migration failure atomicity — [6f2abbdf9810](../../commit/6f2abbdf9810a5cc51b9c2cccdb4368bc7ca4870)
- 推進 rollback Bridge 至 rc.30 候選版 — [35869c53edb2](../../commit/35869c53edb2b3da775f071e2f4c1c86178e07d1)
- 拒絕無效的 prefill legacy indexes — [61c001f84079](../../commit/61c001f8407980c750c40c238421134fc78bf1e1)
- 新增 rc.30 rollback Bridge — [df818ccd0f11](../../commit/df818ccd0f11428ef5a7de9fd33179c56ff929af)
- 修正 rollback Bridge 介面設計 — [3c988a4f0d98](../../commit/3c988a4f0d98e63bc20bb4e500ed25bd5d1be497)
- 規劃 rc.30 rollback Bridge — [91f41f7b680e](../../commit/91f41f7b680e509d1c389f88a8fef03beb1928eb)
- 設計 rc.30 rollback Bridge — [a94284fe5c6b](../../commit/a94284fe5c6b10112c43bfb7832caa82945c8dff)
- 修正 rc.29 backup 數量 — [97c9413e88ce](../../commit/97c9413e88ce48033d2638529c61810902adc842)
- 新增 ZETA release changelog — [e18d488a6c11](../../commit/e18d488a6c1132d9267276f690fd0bf6659f7d91)

- 整合上游 rc.30 tag，簽章合併並保留雙親 — [75cca222f607](../../commit/75cca222f607b703cef48bcd3e478b101e28f062)
- 記錄 rc.30 升級計畫 — [57ebbda62da4](../../commit/57ebbda62da4afa2b5b41c113f3c26cc4c5c4f21)
- 修訂 rc.30 升級設計 — [43b18f09db5e](../../commit/43b18f09db5e42b7e6a01677b6498e41f567c165)
- 設計 rc.30 升級與 changelog — [e8e2d3469cfa](../../commit/e8e2d3469cfaa8df759957731a6ff3c2306393e2)

五個主題備份分支全部從乾淨的 rc.30 tag 重建，各自只含自己的主題：

- `feat/v1.0.0-rc.30/reasoning-model-support`，14 筆，尖端 [c19ae9c73f49](../../commit/c19ae9c73f49af76542e767b3c82ba1c790011fc)
- `feat/v1.0.0-rc.30/chatcompletions-responses-compat`，1 筆，尖端 [f2f151c1df31](../../commit/f2f151c1df31e0e3f28e1eee53e58ecc3de11795)
- `fix/v1.0.0-rc.30/usage-logs-realtime-refresh`，2 筆，尖端 [6b6a176e7443](../../commit/6b6a176e74433e45c30b3dff874815d5e0fc9d4d)
- `fix/v1.0.0-rc.30/channel-affinity-test-isolation`，1 筆，尖端 [78ea94034805](../../commit/78ea940348058f04037b90cb87ad8b90aa466c72)
- `fix/v1.0.0-rc.30/postgres-automigrate-compat`，3 筆，尖端 [8615f9a843ef](../../commit/8615f9a843efd8f5bed2e604f96e6b46507fbc55)

## v1.0.0-rc.29

本版的核心是 PostgreSQL 相容性。rc.28 把 GORM 升到 `v1.25.12` 之後，GORM 會自行推導出 `uni_<table>_<column>` 這種預設約束名稱並直接下 `DROP CONSTRAINT`，但實際的單欄唯一性可能來自另一個具名約束或索引，結果就是啟動時噴 SQLSTATE 42704。ZETA 的 migrator 包裝層在丟棄之前先檢查該約束是否真的存在，同時轉發 dialector 設定、savepoint、錯誤轉譯與 63 字元識別字上限。

Claude 的 effort 別名處理也在這版收斂：別名觸發時清掉 top-p 避免與 thinking 參數衝突，並讓別名定價快取與基礎模型同步。用量日誌在 token 輪換後會重新訂閱，解決換 token 之後即時刷新失效的問題。

- 推進 rc29 PostgreSQL 修正至釋出 — [dccf03595850](../../commit/dccf03595850addfd0901523e5ace279ecb9da83)
- 記錄 PostgreSQL 遷移備份程序 — [55c0dea9d857](../../commit/55c0dea9d8573d9b0f55fd2891d906938e00a850)
- 保護 PostgreSQL 唯一性遷移 — [8e026e7c6abb](../../commit/8e026e7c6abb6a8129bf6b39834ee5c05563aa6e)
- token 輪換後重新訂閱日誌 — [74373bad18bd](../../commit/74373bad18bd878d610660d95f368efefb7757ad)
- effort 別名清除 Claude top-p — [6916cbb25631](../../commit/6916cbb2563121bf67fd921219124a0ec91268d9)
- 強化 Claude helper 迴歸測試 — [2823b818b5a5](../../commit/2823b818b5a5c23afddda994918f0e74b080ffc3)
- 同步別名定價快取 — [505120faf7d6](../../commit/505120faf7d6df3e3456b2037a154d0b3a1535ce)
- 整合上游 rc29 — [e3ab5bc85d13](../../commit/e3ab5bc85d13e370a664ea37bcdc98f3bf09d043)

備份：`feat/v1.0.0-rc.29/reasoning-model-support` 13 筆、`feat/v1.0.0-rc.29/chatcompletions-responses-compat` 1 筆、`fix/v1.0.0-rc.29/usage-logs-realtime-refresh` 2 筆、`fix/v1.0.0-rc.29/channel-affinity-test-isolation` 1 筆、`fix/v1.0.0-rc.29/postgres-automigrate-compat` 1 筆，共 18 筆僅備份提交。

## v1.0.0-rc.27

這版把 GLM 的 reasoning effort 別名從 Zhipu V4 專屬擴展成通用機制：解析泛用 effort 後綴、把別名映射回基礎模型、在 Zhipu 請求中帶上 effort、讓 OpenAI 相容的 chat 路徑也能轉送 effort，並在 passthrough 路徑轉換 GLM 回應格式。多個 chat 入口之間的 effort 保存也一併修好。

- 推進 GLM effort 別名至 rc27 — [15ccc2555635](../../commit/15ccc25556359fa097d039426966d618c5e7d2b8)
- passthrough 轉換 GLM 回應 — [f2a4893e73f1](../../commit/f2a4893e73f1efc840ce50274d310687c21443e4)
- 跨 chat 路徑保存 GLM effort — [d6c83c725fb7](../../commit/d6c83c725fb7c54124152f7d91fb73c0371af049)
- 涵蓋 GLM 通道邊界測試 — [e11702d6decb](../../commit/e11702d6decb935619ab1211433eb0b6f9dcf195)
- 涵蓋 OpenRouter GLM effort 設定測試 — [bbdcc7cdc968](../../commit/bbdcc7cdc968ef74eeff1e4e87f3a6a0f623291d)
- 經由 OpenAI chat 轉送 GLM effort — [b6f58d87726b](../../commit/b6f58d87726b44e941016c28862861b1a739a718)
- Zhipu 請求加入 GLM effort — [c33f9553d53e](../../commit/c33f9553d53ec1ab2ac08354e3c53260fe501f82)
- GLM 別名映射至基礎模型 — [16c88e7f38c1](../../commit/16c88e7f38c152bb5cd53c466d67c6ccd920b1d6)
- 路由通用 GLM effort 別名 — [d63eade70d45](../../commit/d63eade70d45da5120f023a6398cb1c4069abd95)
- 解析通用 GLM effort 別名 — [523feccf6ea7](../../commit/523feccf6ea713f5f71879584d16312a674e72d7)
- 涵蓋 rc27 通道型別過濾測試 — [ceeb7a0c7aec](../../commit/ceeb7a0c7aecb69610177d2e2c5b4f7f24894a0d)
- 整合上游 rc27 — [12b7ac06e9af](../../commit/12b7ac06e9af895c9aea47a963297cf5831fcfac)
- 修正 rc27 tag 驗證流程 — [7add487c63e2](../../commit/7add487c63e295287b4a0419ae5d983eac6040c9)
- 規劃 rc27 GLM 實作 — [74e223c6cdff](../../commit/74e223c6cdff076ea55e611074f127e80e69ed63)
- 設計 rc27 GLM effort 支援 — [3084ef434183](../../commit/3084ef43418370ebc3ab9c266ad11105df2a207c)

備份：`feat/v1.0.0-rc.27/reasoning-model-support` 10 筆、`feat/v1.0.0-rc.27/chatcompletions-responses-compat` 1 筆、`fix/v1.0.0-rc.27/usage-logs-realtime-refresh` 1 筆、`fix/v1.0.0-rc.27/channel-affinity-test-isolation` 1 筆，共 13 筆僅備份提交。

## v1.0.0-rc.26

這版只做上游整合與治理文件，沒有新的執行期行為。rc.26 曾長期是實際的生產部署版本。

- 記錄 rc26 基線例外 — [c0b9df91c8b0](../../commit/c0b9df91c8b0e9e4cc84d71740d7ffaacdb6d2b7)
- 合併上游 v1.0.0-rc.26 — [b4e1486c4f1c](../../commit/b4e1486c4f1c91d3ef485b77f4027bf90d02d709)
- 規劃 rc26 升級 — [fd81ba893e64](../../commit/fd81ba893e649c3cb9cb946dec023e60a2fa7b5f)
- 設計 rc26 升級 — [e595d819211d](../../commit/e595d819211dba119c8f9e9543a287e0c1122798)

備份：四個主題各 1 筆 carry snapshot，共 4 筆僅備份提交。

## v1.0.0-rc.25

這版加入 GLM-5.3 的 effort 後綴，同時把前端用量日誌測試遷移到 Vitest，並在治理層面確立了兩件事：備份分支改為依主題分組，以及禁止向上游開 pull request。

- 推進 GLM-5.3 effort 後綴至釋出 — [fbaea03172fb](../../commit/fbaea03172fb4204041a28f77b39209096f4830d)
- 備份改為依主題分組 — [7352232c5d3f](../../commit/7352232c5d3fa185f26a153640090066b37b25ff)
- 新增 GLM-5.3 effort 後綴 — [b9236f0f61fc](../../commit/b9236f0f61fc92f3ee84dc1cd4581bb6053c67f5)
- 涵蓋 GLM-5.3 effort 後綴測試 — [ca5026c1606e](../../commit/ca5026c1606e8d1d2646981e17f093638a1d6908)
- 設計 GLM-5.3 effort 後綴 — [7ea510fa3416](../../commit/7ea510fa3416f01142e63de6034e4a2770d77ad5)
- 前端用量日誌測試遷移至 Vitest — [c31ec9e81309](../../commit/c31ec9e813096147e1da029934daa98a04291a13)
- 移除已退役的精簡定價優先序測試 — [0c44f42f3456](../../commit/0c44f42f34562a1c6fc438ce5b12d14ad330ec9d)
- 禁止向上游開 pull request — [6b418c2cf23c](../../commit/6b418c2cf23c8afed69fe25ec5887637419e7f4e)
- 合併上游 v1.0.0-rc.25 — [c936432d3659](../../commit/c936432d36591f63eb112201ce86c5d052dd6292)

## v1.0.0-rc.24

這版擴充了三個供應商的 reasoning 控制：Kimi K3 的 none 後綴、Grok 的 effort 後綴、GLM-5.2 的 effort 別名與後綴解析，另外加上 DeepSeek 的 low effort 後綴。GLM 別名的重試優先序與 Zhipu V4 範圍限制在這版一併修正，避免別名污染其他通道的選擇。

- 推進 Kimi K3 none 後綴至釋出 — [f4daf51fc6ff](../../commit/f4daf51fc6ffc658446b3dc2a168538811ab9a3c)
- 新增 Kimi K3 none 後綴 — [79e94e27b8ca](../../commit/79e94e27b8ca193977908b983f6f5152c7c016c1)
- 推進 GLM-5.2 effort 後綴至釋出 — [e1c0767de16c](../../commit/e1c0767de16c99c7fefad6306c9c701368b1d6a9)
- 保存 GLM 別名重試優先序 — [0c8d0cc109d0](../../commit/0c8d0cc109d02049c1451d42899e46a9676566eb)
- 限制 GLM 別名於 Zhipu V4 — [6c7054468f3d](../../commit/6c7054468f3dbf1b5d16b0dba88546aaadd3b4c6)
- 保護 GLM-5.2 relay metadata — [4dea093de3c7](../../commit/4dea093de3c739979dece3d97fdd7a3a278e4e70)
- 新增 GLM-5.2 effort 別名 — [05cbbf6cbb0c](../../commit/05cbbf6cbb0c94a3bc5b7d40819230e138fa0c1f)
- 解析 GLM-5.2 effort 後綴 — [ea771fc2db91](../../commit/ea771fc2db910833e33561cd51c4cca42ba34bd6)
- 設計 GLM-5.2 effort 後綴 — [8bb827cf5b32](../../commit/8bb827cf5b3221187c610e36560ead9fec6a87b8)
- 推進 Grok effort 後綴至釋出 — [bcf2ed95dc81](../../commit/bcf2ed95dc81f8424095dcf32c071cd3c82684b2)
- 新增 Grok effort 後綴 — [b53436d02375](../../commit/b53436d02375f1bce2477fb821404a1e18604781)
- 推進 rc.24 候選版 — [160937517778](../../commit/160937517778c2a1a86e89402025208995e29eda)
- 隔離 affinity 快取計數器 — [2e58c0dfd2d6](../../commit/2e58c0dfd2d6233f9342e7ba69792c9fc8505ea8)
- 新增 DeepSeek low effort 後綴 — [b54f87ecd006](../../commit/b54f87ecd006ee0aa945e905515f5ad719f0a97f)
- 合併上游 rc.24 — [9c42a8083486](../../commit/9c42a808348676e561ca4b53ed017a8918278ed6)

## v1.0.0-rc.21

這是 ZETA 自訂內容最多的一版。Kimi 系列拿到完整的 reasoning 模式控制：K3 預設不推理、Moonshot thinking 後綴、K3 max effort 與 effort 後綴別名，固定參數的正規化也一併處理。用量日誌改為串流更新，測試結束後會自動刷新。Claude Opus 5 的定價在這版加入。治理層面則確立了版本化釋出流程、本地測試程序、Zeabur 開發與釋出部署對應表，以及只保留當前版本分支集的剪枝政策。

- 釋出 Claude Opus 5 定價 — [2013b9a29194](../../commit/2013b9a2919468af1e824acf32405d400dc6bec2)
- 新增 Claude Opus 5 定價 — [035ee56654e3](../../commit/035ee56654e31c4b11eeb8c9dcea1efbf9f24a9f)
- 釋出 K3 effort 後綴別名 — [b69297ac8cef](../../commit/b69297ac8cef8af880ed9044d0127f64fe432f34)
- 新增 K3 effort 後綴別名 — [89e518e20646](../../commit/89e518e20646eed40c298d7f3ff84b65210bd7fe)
- 整合 Zeabur 部署對應文件 — [46833cedc697](../../commit/46833cedc697839ebba99ddc1ab1a38e793ca0c8)
- 記錄 Zeabur 開發與釋出部署對應 — [8fac33ce047b](../../commit/8fac33ce047b647a67486366ba8eb025df43bfa8)
- 推進日誌串流刷新至釋出 — [f857d4c477af](../../commit/f857d4c477af95ffb8eb6fd41a1bd985666624a0)
- 記錄本地測試程序 — [bafea1f0c477](../../commit/bafea1f0c47799e8591fe18b62154a0eea2910e0)
- 串流用量日誌更新 — [5be3f1e7d695](../../commit/5be3f1e7d695dee9bc7b240d94edcb5b83b1745e)
- 只保留當前版本分支 — [935bf63036ed](../../commit/935bf63036ed7bd69198757dbc65e4ce6f214c94)
- 整合用量日誌修正 — [99af294b7bd1](../../commit/99af294b7bd158ec69f7bb9cd61c8be0f563a353)
- 測試後刷新用量日誌 — [3285b198e9a9](../../commit/3285b198e9a914919d231d6cee1c5644f182c89e)
- 新增 Kimi reasoning 模式 — [54174ef5d4d1](../../commit/54174ef5d4d1b0e347538fa6581f82110114679e)
- K3 預設不啟用推理 — [e479446253f9](../../commit/e479446253f9a8b55dcbe1d335c66e8b1558720e)
- 正規化 Kimi 固定參數 — [c329e717efa5](../../commit/c329e717efa520223c986cce3204fb9ed581530d)
- 新增 Moonshot thinking 後綴 — [6d07f6f079cf](../../commit/6d07f6f079cf19a642672f4239ba4fe72f0235bb)
- 支援 Kimi K3 max reasoning effort — [682204ac367c](../../commit/682204ac367c228aa7daee778b5e484bc9a1cccb)
- 記錄 agent skills 與 issue 追蹤 — [3e55e6428489](../../commit/3e55e6428489f0528681dec8f919d015d1cf26ef)
- 定義版本化釋出流程 — [46dc0c793c5c](../../commit/46dc0c793c5cf7e4c872a04293adc1846995d719)
- 保留所有 rc 功能分支 — [2968c851a74d](../../commit/2968c851a74d5f6ee04150f3886842e6ecff4125)
- 合併 tag v1.0.0-rc.21 — [27d6c0220d38](../../commit/27d6c0220d381f3c95aa047feed93506105ebc20)

## v1.0.0-rc.20

這版加入萬用字元模型定價與 GPT-5.6 的 reasoning 後綴，並首次寫下釋出攜帶政策。

- 新增萬用字元模型定價 — [d1a0dfd547aa](../../commit/d1a0dfd547aa69f9dadff6360905c22847e06e2b)
- 新增 GPT-5.6 reasoning 後綴 — [7c6b82bcbc7e](../../commit/7c6b82bcbc7ea8b9f1b05d0498d9394301d35385)
- 新增釋出攜帶政策 — [60489207ec3e](../../commit/60489207ec3e775c2e407c422a4c0e7b783cd5ff)
- 合併 tag v1.0.0-rc.20 — [5705d3a40d78](../../commit/5705d3a40d78b9999556e10a02a7733d9496cc68)

## v1.0.0-rc.18

這版只有上游整合，沒有 ZETA 自訂行為變更。

- 合併 tag v1.0.0-rc.18 — [17a0d49c4c56](../../commit/17a0d49c4c56665007858ddf536fff65d8a05309)

## v1.0.0-rc.15

這是 ZETA 自訂歷史的起點，也是唯一沒有 Integration OID 的版本。以下維持 newest-first；四筆具有相同 committer timestamp 的提交再依實際拓撲由新到舊排列。classic zh-TW 自訂已於 2026-07-18 由擁有者明確宣告退役，後續版本不再攜帶。

- 新增 Claude Sonnet 5 支援 — [58e14b1294a8](../../commit/58e14b1294a8e48b5c588fd8d5fd771d815c0cc6)
- 補齊 classic zh-TW 翻譯 — [612da8085a89](../../commit/612da8085a89bb4faeedeaf2218dba1827eafcc8)
- 新增 Claude adaptive Fable 模型 — [16c884b0d66c](../../commit/16c884b0d66c05ee1f347bb4ba6927efeb8f9400)
- 對齊 responses reasoning 指標 — [7390b78a4f64](../../commit/7390b78a4f64ce7827a4ad1646284c89bae01eb5)
- 保存 responses 轉換細節 — [93a7cd118fa6](../../commit/93a7cd118fa65f22a9f17ed63810c57874a37d78)

---

## 未釋出

以下 3 筆停留在未合併的 `feat/rc25/glm-effort-openai-openrouter` 主題分支上，目標版本是 rc.25。這批工作嘗試讓 GLM effort 服務於 Zhipu V4 以外的通道；rc.27 後來以另一組 OID 實作了相近行為，但那不會把這 3 個 OID 轉為已釋出歷史。

- 讓 GLM effort 服務於 Zhipu V4 之外 — [5fd1641ae442](../../commit/5fd1641ae442a24ee7d5c70e7a70186ae6417af1)
- 涵蓋 Zhipu V4 之外的 GLM effort 測試 — [7ceecb3b7174](../../commit/7ceecb3b7174c458f3a0ade4a297a90cec91ad86)
- 設計 Zhipu V4 之外的 GLM effort — [552bc0dc411e](../../commit/552bc0dc411ef9880f5ddc97801fd2d2f48cbe11)

## 技術帳本

狀態由 Provenance 欄明示；Version 欄記錄 release epoch 或未釋出工作的目標版本。最新 Released rows 指向 signed release merge `50f0f0d13f09b08601bc15ce2e40bf989e351e3e`；較早 rows 保留 Contract merge `85da814a0d8f0d7a10e6d479017aabcf3c829e88` 與首次完整快照 `75cca222f607b703cef48bcd3e478b101e28f062` 作為歷史來源證據，三者都位於目前 rc.30 釋出祖先鏈。

| Version | Class | Commit | Author date | Committer date | Subject | Provenance |
| --- | --- | --- | --- | --- | --- | --- |
| v1.0.0-rc.30 | Backup rebuild | [c19ae9c73f49](../../commit/c19ae9c73f49af76542e767b3c82ba1c790011fc) | 2026-09-02T23:11:45+08:00 | 2026-09-02T23:24:22+08:00 | fix: preserve nullable channel priority | Backup-only; `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support`; exact patch-id of `f9f1a465e7572569a7392d79024697fdb5e7a1e7` |
| v1.0.0-rc.30 | Integration | [50f0f0d13f09](../../commit/50f0f0d13f09b08601bc15ce2e40bf989e351e3e) | 2026-09-02T23:23:08+08:00 | 2026-09-02T23:23:08+08:00 | fix: preserve nullable channel priority | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `50f0f0d13f09b08601bc15ce2e40bf989e351e3e` |
| v1.0.0-rc.30 | Fix | [f9f1a465e757](../../commit/f9f1a465e7572569a7392d79024697fdb5e7a1e7) | 2026-09-02T23:11:45+08:00 | 2026-09-02T23:11:45+08:00 | fix: preserve nullable channel priority | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `50f0f0d13f09b08601bc15ce2e40bf989e351e3e` |
| v1.0.0-rc.30 | Backup rebuild | [8615f9a843ef](../../commit/8615f9a843efd8f5bed2e604f96e6b46507fbc55) | 2026-09-02T17:56:09+08:00 | 2026-09-02T17:56:09+08:00 | fix: complete prefill uniqueness contract | Backup-only; `refs/heads/fix/v1.0.0-rc.30/postgres-automigrate-compat`; exact Contract delta |
| v1.0.0-rc.30 | Integration | [85da814a0d8f](../../commit/85da814a0d8f0d7a10e6d479017aabcf3c829e88) | 2026-09-02T17:47:03+08:00 | 2026-09-02T17:47:03+08:00 | fix: promote rc30 uniqueness contract | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `85da814a0d8f0d7a10e6d479017aabcf3c829e88` |
| v1.0.0-rc.30 | Fix | [6f2abbdf9810](../../commit/6f2abbdf9810a5cc51b9c2cccdb4368bc7ca4870) | 2026-09-02T17:18:05+08:00 | 2026-09-02T17:38:03+08:00 | fix: complete prefill uniqueness contract | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `85da814a0d8f0d7a10e6d479017aabcf3c829e88` |
| v1.0.0-rc.30 | Integration | [35869c53edb2](../../commit/35869c53edb2b3da775f071e2f4c1c86178e07d1) | 2026-09-01T21:55:39+08:00 | 2026-09-01T21:55:39+08:00 | fix: promote rc30 rollback bridge | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `85da814a0d8f0d7a10e6d479017aabcf3c829e88` |
| v1.0.0-rc.30 | Backup rebuild | [e19302477e75](../../commit/e19302477e75e411e9e08563b77e1ed9db177707) | 2026-09-01T21:34:59+08:00 | 2026-09-01T21:34:59+08:00 | fix: add rc30 rollback bridge | Backup-only; `refs/heads/fix/v1.0.0-rc.30/postgres-automigrate-compat`; exact Bridge delta |
| v1.0.0-rc.30 | Fix | [61c001f84079](../../commit/61c001f8407980c750c40c238421134fc78bf1e1) | 2026-09-01T21:17:07+08:00 | 2026-09-01T21:17:07+08:00 | fix: reject invalid prefill legacy indexes | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `85da814a0d8f0d7a10e6d479017aabcf3c829e88` |
| v1.0.0-rc.30 | Fix | [df818ccd0f11](../../commit/df818ccd0f11428ef5a7de9fd33179c56ff929af) | 2026-09-01T20:52:58+08:00 | 2026-09-01T20:52:58+08:00 | fix: add rc30 rollback bridge | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `85da814a0d8f0d7a10e6d479017aabcf3c829e88` |
| v1.0.0-rc.30 | Docs | [3c988a4f0d98](../../commit/3c988a4f0d98e63bc20bb4e500ed25bd5d1be497) | 2026-09-01T19:14:25+08:00 | 2026-09-01T19:14:25+08:00 | docs: fix rollback bridge interfaces | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `85da814a0d8f0d7a10e6d479017aabcf3c829e88` |
| v1.0.0-rc.30 | Docs | [91f41f7b680e](../../commit/91f41f7b680e509d1c389f88a8fef03beb1928eb) | 2026-09-01T18:40:57+08:00 | 2026-09-01T18:40:57+08:00 | docs: plan rc30 rollback bridge | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `85da814a0d8f0d7a10e6d479017aabcf3c829e88` |
| v1.0.0-rc.30 | Docs | [a94284fe5c6b](../../commit/a94284fe5c6b10112c43bfb7832caa82945c8dff) | 2026-09-01T18:31:14+08:00 | 2026-09-01T18:31:14+08:00 | docs: design rc30 rollback bridge | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `85da814a0d8f0d7a10e6d479017aabcf3c829e88` |
| v1.0.0-rc.30 | Docs | [97c9413e88ce](../../commit/97c9413e88ce48033d2638529c61810902adc842) | 2026-09-01T16:46:30+08:00 | 2026-09-01T16:46:30+08:00 | docs: correct rc29 backup count | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `85da814a0d8f0d7a10e6d479017aabcf3c829e88` |
| v1.0.0-rc.30 | Docs | [e18d488a6c11](../../commit/e18d488a6c1132d9267276f690fd0bf6659f7d91) | 2026-09-01T08:41:58+08:00 | 2026-09-01T12:13:33+08:00 | docs: add Zeta release changelog | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `85da814a0d8f0d7a10e6d479017aabcf3c829e88` |
| v1.0.0-rc.30 | Integration | [75cca222f607](../../commit/75cca222f607b703cef48bcd3e478b101e28f062) | 2026-09-01T04:30:47+08:00 | 2026-09-01T04:30:47+08:00 | build: integrate upstream rc30 | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.30 | Docs | [57ebbda62da4](../../commit/57ebbda62da4afa2b5b41c113f3c26cc4c5c4f21) | 2026-09-01T04:22:48+08:00 | 2026-09-01T04:22:48+08:00 | docs: plan rc30 upgrade | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.30 | Docs | [43b18f09db5e](../../commit/43b18f09db5e42b7e6a01677b6498e41f567c165) | 2026-09-01T04:17:41+08:00 | 2026-09-01T04:17:41+08:00 | docs: revise rc30 upgrade design | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.30 | Docs | [e8e2d3469cfa](../../commit/e8e2d3469cfaa8df759957731a6ff3c2306393e2) | 2026-09-01T03:42:49+08:00 | 2026-09-01T03:42:49+08:00 | docs: design rc30 upgrade and changelog | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.30 | Backup rebuild | [b37f6be3a309](../../commit/b37f6be3a309c89b60067e64bb80c47340800096) | 2026-08-31T13:05:39+08:00 | 2026-09-01T07:39:27+08:00 | fix: guard PostgreSQL unique migrations | Backup-only; `refs/heads/fix/v1.0.0-rc.30/postgres-automigrate-compat`; cherry-picked from `0a8422e16490cc7854dd20c9d0ca5767498dea9e` |
| v1.0.0-rc.30 | Backup rebuild | [78ea94034805](../../commit/78ea940348058f04037b90cb87ad8b90aa466c72) | 2026-08-19T18:50:22+08:00 | 2026-09-01T07:14:37+08:00 | fix: isolate rc27 affinity tests | Backup-only; `refs/heads/fix/v1.0.0-rc.30/channel-affinity-test-isolation`; cherry-picked from `bc4e43801de037257a6b00f6b44c23b8482b5cce` |
| v1.0.0-rc.30 | Backup rebuild | [6b6a176e7443](../../commit/6b6a176e74433e45c30b3dff874815d5e0fc9d4d) | 2026-08-31T11:24:05+08:00 | 2026-09-01T06:51:22+08:00 | fix: resubscribe logs after token rotation | Backup-only; `refs/heads/fix/v1.0.0-rc.30/usage-logs-realtime-refresh`; cherry-picked from `78e4960269ed35c6ea1d90b10205fc347fd76976` |
| v1.0.0-rc.30 | Backup rebuild | [59ba761701a0](../../commit/59ba761701a0a07b4046c813e6d19f8702a38e98) | 2026-08-19T18:50:04+08:00 | 2026-09-01T06:49:35+08:00 | fix: refresh rc27 usage logs in realtime | Backup-only; `refs/heads/fix/v1.0.0-rc.30/usage-logs-realtime-refresh`; cherry-picked from `601b53b61e155075cc68bda5b3f4ee0c5add04ae` |
| v1.0.0-rc.30 | Backup rebuild | [f2f151c1df31](../../commit/f2f151c1df31e0e3f28e1eee53e58ecc3de11795) | 2026-08-19T18:46:46+08:00 | 2026-09-01T06:33:47+08:00 | feat: add rc27 responses compatibility | Backup-only; `refs/heads/feat/v1.0.0-rc.30/chatcompletions-responses-compat`; cherry-picked from `965b46fd5bb3188cfb779b86b03f88fab8debb31` |
| v1.0.0-rc.30 | Backup rebuild | [d283b48067f9](../../commit/d283b48067f9410a0e4bc09b8555167c009fc1cd) | 2026-08-31T10:19:09+08:00 | 2026-09-01T06:01:13+08:00 | test: harden Claude helper regression | Backup-only; `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support`; cherry-picked from `91450384951ac181a8d25ce2774743afb6e3f62c` |
| v1.0.0-rc.30 | Backup rebuild | [e809f06cbe47](../../commit/e809f06cbe471089888e421487908dc209e7f1bc) | 2026-08-31T10:04:28+08:00 | 2026-09-01T06:01:11+08:00 | fix: clear Claude top-p for effort aliases | Backup-only; `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support`; cherry-picked from `c0d641470e3419efba35ca767181ea49449e9f50` |
| v1.0.0-rc.30 | Backup rebuild | [a1279e9980d3](../../commit/a1279e9980d34cab544b44e528bf0bbbcbf863ce) | 2026-08-31T08:43:19+08:00 | 2026-09-01T06:01:10+08:00 | fix: synchronize alias pricing cache | Backup-only; `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support`; cherry-picked from `99184f095d652e2eafe7d23393b49ce6bf198f1c` |
| v1.0.0-rc.30 | Backup rebuild | [c6c8abff3fc5](../../commit/c6c8abff3fc5faa1d24617a49c5e3dcb2f860480) | 2026-08-30T19:14:16+08:00 | 2026-09-01T06:01:09+08:00 | fix: convert GLM responses in passthrough | Backup-only; `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support`; cherry-picked from `f2a4893e73f1efc840ce50274d310687c21443e4` |
| v1.0.0-rc.30 | Backup rebuild | [02cd75ee860f](../../commit/02cd75ee860f3ddd83cd9e0643db0fc4f607d027) | 2026-08-30T19:09:04+08:00 | 2026-09-01T06:01:08+08:00 | fix: preserve GLM effort across chat paths | Backup-only; `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support`; cherry-picked from `d6c83c725fb7c54124152f7d91fb73c0371af049` |
| v1.0.0-rc.30 | Backup rebuild | [2e5eb4593d1f](../../commit/2e5eb4593d1fbc672c014856f24328b2789a419a) | 2026-08-30T18:46:38+08:00 | 2026-09-01T06:01:07+08:00 | test: cover GLM channel boundaries | Backup-only; `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support`; cherry-picked from `e11702d6decb935619ab1211433eb0b6f9dcf195` |
| v1.0.0-rc.30 | Backup rebuild | [8a1275feebc4](../../commit/8a1275feebc47e4880e03cf9d1b897260008ae89) | 2026-08-30T18:44:01+08:00 | 2026-09-01T06:01:06+08:00 | test: cover OpenRouter GLM effort config | Backup-only; `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support`; cherry-picked from `bbdcc7cdc968ef74eeff1e4e87f3a6a0f623291d` |
| v1.0.0-rc.30 | Backup rebuild | [e5a2429609d3](../../commit/e5a2429609d33422d34c0b8b74cf731ff7120d64) | 2026-08-30T18:42:26+08:00 | 2026-09-01T06:01:05+08:00 | feat: relay GLM effort through OpenAI chat | Backup-only; `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support`; cherry-picked from `b6f58d87726b44e941016c28862861b1a739a718` |
| v1.0.0-rc.30 | Backup rebuild | [56633722ad0d](../../commit/56633722ad0dd71a83fbac004ecc635b07faa57a) | 2026-08-30T18:39:15+08:00 | 2026-09-01T06:01:04+08:00 | feat: add GLM effort to Zhipu requests | Backup-only; `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support`; cherry-picked from `c33f9553d53ec1ab2ac08354e3c53260fe501f82` |
| v1.0.0-rc.30 | Backup rebuild | [227348acbad6](../../commit/227348acbad64fa3abbd4673c354272bed6791a8) | 2026-08-30T18:35:04+08:00 | 2026-09-01T06:01:03+08:00 | feat: map GLM aliases through base models | Backup-only; `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support`; cherry-picked from `16c88e7f38c152bb5cd53c466d67c6ccd920b1d6` |
| v1.0.0-rc.30 | Backup rebuild | [72c0d721a618](../../commit/72c0d721a6186fa16ac474b04c33e3b106078c5c) | 2026-08-30T18:32:45+08:00 | 2026-09-01T06:01:02+08:00 | feat: route generic GLM effort aliases | Backup-only; `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support`; cherry-picked from `d63eade70d45da5120f023a6398cb1c4069abd95` |
| v1.0.0-rc.30 | Backup rebuild | [c27f8fb79406](../../commit/c27f8fb79406dd6b6ab23a326fcbe6d672a70e84) | 2026-08-30T18:25:03+08:00 | 2026-09-01T06:01:01+08:00 | feat: parse generic GLM effort aliases | Backup-only; `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support`; cherry-picked from `523feccf6ea713f5f71879584d16312a674e72d7` |
| v1.0.0-rc.30 | Backup rebuild | [9fb194b2fb28](../../commit/9fb194b2fb28732d9e537bdd0ec8482172255496) | 2026-08-30T19:02:50+08:00 | 2026-09-01T05:59:00+08:00 | feat: carry rc27 reasoning model support | Backup-only; `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support`; cherry-picked from `c242c586a6e9b3a5ea1eba435f382c85e78b0cf6` |
| v1.0.0-rc.29 | Integration | [dccf03595850](../../commit/dccf03595850addfd0901523e5ace279ecb9da83) | 2026-08-31T13:15:11+08:00 | 2026-08-31T13:15:11+08:00 | build: promote rc29 PostgreSQL fix | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.29 | Governance | [55c0dea9d857](../../commit/55c0dea9d8573d9b0f55fd2891d906938e00a850) | 2026-08-31T13:13:07+08:00 | 2026-08-31T13:13:07+08:00 | docs: add PostgreSQL migration backup | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.29 | Fix | [8e026e7c6abb](../../commit/8e026e7c6abb6a8129bf6b39834ee5c05563aa6e) | 2026-08-31T13:05:39+08:00 | 2026-08-31T13:05:39+08:00 | fix: guard PostgreSQL unique migrations | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.29 | Fix | [74373bad18bd](../../commit/74373bad18bd878d610660d95f368efefb7757ad) | 2026-08-31T11:24:05+08:00 | 2026-08-31T11:56:04+08:00 | fix: resubscribe logs after token rotation | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.29 | Fix | [6916cbb25631](../../commit/6916cbb2563121bf67fd921219124a0ec91268d9) | 2026-08-31T10:04:28+08:00 | 2026-08-31T10:28:50+08:00 | fix: clear Claude top-p for effort aliases | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.29 | Test | [2823b818b5a5](../../commit/2823b818b5a5c23afddda994918f0e74b080ffc3) | 2026-08-31T10:19:09+08:00 | 2026-08-31T10:28:50+08:00 | test: harden Claude helper regression | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.29 | Fix | [505120faf7d6](../../commit/505120faf7d6df3e3456b2037a154d0b3a1535ce) | 2026-08-31T08:43:19+08:00 | 2026-08-31T09:17:45+08:00 | fix: synchronize alias pricing cache | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.29 | Integration | [e3ab5bc85d13](../../commit/e3ab5bc85d13e370a664ea37bcdc98f3bf09d043) | 2026-08-31T08:10:18+08:00 | 2026-08-31T08:10:32+08:00 | build: integrate upstream rc29 | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.29 | Carry snapshot | [0a8422e16490](../../commit/0a8422e16490cc7854dd20c9d0ca5767498dea9e) | 2026-08-31T13:05:39+08:00 | 2026-08-31T13:13:43+08:00 | fix: guard PostgreSQL unique migrations | Backup-only; `refs/heads/fix/v1.0.0-rc.29/postgres-automigrate-compat`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.29 | Carry snapshot | [78e4960269ed](../../commit/78e4960269ed35c6ea1d90b10205fc347fd76976) | 2026-08-31T11:24:05+08:00 | 2026-08-31T11:42:00+08:00 | fix: resubscribe logs after token rotation | Backup-only; `refs/heads/fix/v1.0.0-rc.29/usage-logs-realtime-refresh`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.29 | Carry snapshot | [bc4e43801de0](../../commit/bc4e43801de037257a6b00f6b44c23b8482b5cce) | 2026-08-19T18:50:22+08:00 | 2026-08-31T11:03:14+08:00 | fix: isolate rc27 affinity tests | Backup-only; `refs/heads/fix/v1.0.0-rc.29/channel-affinity-test-isolation`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.29 | Carry snapshot | [601b53b61e15](../../commit/601b53b61e155075cc68bda5b3f4ee0c5add04ae) | 2026-08-19T18:50:04+08:00 | 2026-08-31T10:58:48+08:00 | fix: refresh rc27 usage logs in realtime | Backup-only; `refs/heads/fix/v1.0.0-rc.29/usage-logs-realtime-refresh`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.29 | Carry snapshot | [965b46fd5bb3](../../commit/965b46fd5bb3188cfb779b86b03f88fab8debb31) | 2026-08-19T18:46:46+08:00 | 2026-08-31T10:41:42+08:00 | feat: add rc27 responses compatibility | Backup-only; `refs/heads/feat/v1.0.0-rc.29/chatcompletions-responses-compat`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.29 | Carry snapshot | [91450384951a](../../commit/91450384951ac181a8d25ce2774743afb6e3f62c) | 2026-08-31T10:19:09+08:00 | 2026-08-31T10:19:09+08:00 | test: harden Claude helper regression | Backup-only; `refs/heads/feat/v1.0.0-rc.29/reasoning-model-support`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.29 | Carry snapshot | [c0d641470e34](../../commit/c0d641470e3419efba35ca767181ea49449e9f50) | 2026-08-31T10:04:28+08:00 | 2026-08-31T10:04:28+08:00 | fix: clear Claude top-p for effort aliases | Backup-only; `refs/heads/feat/v1.0.0-rc.29/reasoning-model-support`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.29 | Carry snapshot | [99184f095d65](../../commit/99184f095d652e2eafe7d23393b49ce6bf198f1c) | 2026-08-31T08:43:19+08:00 | 2026-08-31T09:35:45+08:00 | fix: synchronize alias pricing cache | Backup-only; `refs/heads/feat/v1.0.0-rc.29/reasoning-model-support`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.29 | Backup rebuild | [8c5721f815d8](../../commit/8c5721f815d891e42599a0073c782a27cae0d166) | 2026-08-30T19:14:16+08:00 | 2026-08-31T09:35:22+08:00 | fix: convert GLM responses in passthrough | Backup-only; `refs/heads/feat/v1.0.0-rc.29/reasoning-model-support`; cherry-picked from `f2a4893e73f1efc840ce50274d310687c21443e4` |
| v1.0.0-rc.29 | Backup rebuild | [0fd6e8d9fa9a](../../commit/0fd6e8d9fa9a843cebfe411a8d76a8a8af67bc23) | 2026-08-30T19:09:04+08:00 | 2026-08-31T09:35:07+08:00 | fix: preserve GLM effort across chat paths | Backup-only; `refs/heads/feat/v1.0.0-rc.29/reasoning-model-support`; cherry-picked from `d6c83c725fb7c54124152f7d91fb73c0371af049` |
| v1.0.0-rc.29 | Backup rebuild | [5409dcbc1612](../../commit/5409dcbc1612c5dcb68965a09086ab06cbd226f3) | 2026-08-30T18:46:38+08:00 | 2026-08-31T09:34:50+08:00 | test: cover GLM channel boundaries | Backup-only; `refs/heads/feat/v1.0.0-rc.29/reasoning-model-support`; cherry-picked from `e11702d6decb935619ab1211433eb0b6f9dcf195` |
| v1.0.0-rc.29 | Backup rebuild | [0c258052d2ac](../../commit/0c258052d2ac43e2aa179557542083fa703b2cb5) | 2026-08-30T18:44:01+08:00 | 2026-08-31T09:34:29+08:00 | test: cover OpenRouter GLM effort config | Backup-only; `refs/heads/feat/v1.0.0-rc.29/reasoning-model-support`; cherry-picked from `bbdcc7cdc968ef74eeff1e4e87f3a6a0f623291d` |
| v1.0.0-rc.29 | Backup rebuild | [5e19a240b553](../../commit/5e19a240b5537ac2db405b082f3a0cd33693f251) | 2026-08-30T18:42:26+08:00 | 2026-08-31T09:34:07+08:00 | feat: relay GLM effort through OpenAI chat | Backup-only; `refs/heads/feat/v1.0.0-rc.29/reasoning-model-support`; cherry-picked from `b6f58d87726b44e941016c28862861b1a739a718` |
| v1.0.0-rc.29 | Backup rebuild | [b65ebfce607d](../../commit/b65ebfce607d2fd128c7d4e312d8f74543271152) | 2026-08-30T18:39:15+08:00 | 2026-08-31T09:33:47+08:00 | feat: add GLM effort to Zhipu requests | Backup-only; `refs/heads/feat/v1.0.0-rc.29/reasoning-model-support`; cherry-picked from `c33f9553d53ec1ab2ac08354e3c53260fe501f82` |
| v1.0.0-rc.29 | Backup rebuild | [d0e62e6cbaa2](../../commit/d0e62e6cbaa296e3bbc154dd394bca725afd3477) | 2026-08-30T18:35:04+08:00 | 2026-08-31T09:33:30+08:00 | feat: map GLM aliases through base models | Backup-only; `refs/heads/feat/v1.0.0-rc.29/reasoning-model-support`; cherry-picked from `16c88e7f38c152bb5cd53c466d67c6ccd920b1d6` |
| v1.0.0-rc.29 | Backup rebuild | [80f411a0ba0c](../../commit/80f411a0ba0c6f800969fa20774196dd085255b3) | 2026-08-30T18:32:45+08:00 | 2026-08-31T09:33:12+08:00 | feat: route generic GLM effort aliases | Backup-only; `refs/heads/feat/v1.0.0-rc.29/reasoning-model-support`; cherry-picked from `d63eade70d45da5120f023a6398cb1c4069abd95` |
| v1.0.0-rc.29 | Backup rebuild | [58388cc5749c](../../commit/58388cc5749c069bf5dd98f3bcd4409c6f655944) | 2026-08-30T18:25:03+08:00 | 2026-08-31T09:32:47+08:00 | feat: parse generic GLM effort aliases | Backup-only; `refs/heads/feat/v1.0.0-rc.29/reasoning-model-support`; cherry-picked from `523feccf6ea713f5f71879584d16312a674e72d7` |
| v1.0.0-rc.29 | Carry snapshot | [c242c586a6e9](../../commit/c242c586a6e9b3a5ea1eba435f382c85e78b0cf6) | 2026-08-30T19:02:50+08:00 | 2026-08-31T09:32:11+08:00 | feat: carry rc27 reasoning model support | Backup-only; `refs/heads/feat/v1.0.0-rc.29/reasoning-model-support`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.27 | Integration | [15ccc2555635](../../commit/15ccc25556359fa097d039426966d618c5e7d2b8) | 2026-08-30T19:21:52+08:00 | 2026-08-30T19:21:52+08:00 | feat: promote GLM effort aliases to rc27 | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Fix | [f2a4893e73f1](../../commit/f2a4893e73f1efc840ce50274d310687c21443e4) | 2026-08-30T19:14:16+08:00 | 2026-08-30T19:14:16+08:00 | fix: convert GLM responses in passthrough | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Fix | [d6c83c725fb7](../../commit/d6c83c725fb7c54124152f7d91fb73c0371af049) | 2026-08-30T19:09:04+08:00 | 2026-08-30T19:09:04+08:00 | fix: preserve GLM effort across chat paths | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Test | [e11702d6decb](../../commit/e11702d6decb935619ab1211433eb0b6f9dcf195) | 2026-08-30T18:46:38+08:00 | 2026-08-30T18:46:38+08:00 | test: cover GLM channel boundaries | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Test | [bbdcc7cdc968](../../commit/bbdcc7cdc968ef74eeff1e4e87f3a6a0f623291d) | 2026-08-30T18:44:01+08:00 | 2026-08-30T18:44:01+08:00 | test: cover OpenRouter GLM effort config | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Feature | [b6f58d87726b](../../commit/b6f58d87726b44e941016c28862861b1a739a718) | 2026-08-30T18:42:26+08:00 | 2026-08-30T18:42:26+08:00 | feat: relay GLM effort through OpenAI chat | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Feature | [c33f9553d53e](../../commit/c33f9553d53ec1ab2ac08354e3c53260fe501f82) | 2026-08-30T18:39:15+08:00 | 2026-08-30T18:39:15+08:00 | feat: add GLM effort to Zhipu requests | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Feature | [16c88e7f38c1](../../commit/16c88e7f38c152bb5cd53c466d67c6ccd920b1d6) | 2026-08-30T18:35:04+08:00 | 2026-08-30T18:35:04+08:00 | feat: map GLM aliases through base models | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Feature | [d63eade70d45](../../commit/d63eade70d45da5120f023a6398cb1c4069abd95) | 2026-08-30T18:32:45+08:00 | 2026-08-30T18:32:45+08:00 | feat: route generic GLM effort aliases | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Feature | [523feccf6ea7](../../commit/523feccf6ea713f5f71879584d16312a674e72d7) | 2026-08-30T18:25:03+08:00 | 2026-08-30T18:25:03+08:00 | feat: parse generic GLM effort aliases | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Test | [ceeb7a0c7aec](../../commit/ceeb7a0c7aecb69610177d2e2c5b4f7f24894a0d) | 2026-08-30T17:59:21+08:00 | 2026-08-30T17:59:21+08:00 | test: cover rc27 channel type filters | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Integration | [12b7ac06e9af](../../commit/12b7ac06e9af895c9aea47a963297cf5831fcfac) | 2026-08-30T17:57:41+08:00 | 2026-08-30T17:57:41+08:00 | build: integrate upstream rc27 | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Docs | [7add487c63e2](../../commit/7add487c63e295287b4a0419ae5d983eac6040c9) | 2026-08-30T17:49:32+08:00 | 2026-08-30T17:49:32+08:00 | docs: fix rc27 tag verification | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Docs | [74e223c6cdff](../../commit/74e223c6cdff076ea55e611074f127e80e69ed63) | 2026-08-30T17:46:40+08:00 | 2026-08-30T17:46:40+08:00 | docs: plan rc27 GLM implementation | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Docs | [3084ef434183](../../commit/3084ef43418370ebc3ab9c266ad11105df2a207c) | 2026-08-30T16:36:41+08:00 | 2026-08-30T16:36:41+08:00 | docs: design rc27 GLM effort support | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.27 | Backup rebuild | [e6fe18d23e01](../../commit/e6fe18d23e016023f1542adee4aaba7d5d726dc5) | 2026-08-30T19:14:16+08:00 | 2026-08-30T19:19:13+08:00 | fix: convert GLM responses in passthrough | Backup-only; `refs/heads/feat/v1.0.0-rc.27/reasoning-model-support`; cherry-picked from `f2a4893e73f1efc840ce50274d310687c21443e4` |
| v1.0.0-rc.27 | Backup rebuild | [1b27408105a2](../../commit/1b27408105a2bac5d13dd3f7ab005711ce6805c9) | 2026-08-30T19:09:04+08:00 | 2026-08-30T19:10:25+08:00 | fix: preserve GLM effort across chat paths | Backup-only; `refs/heads/feat/v1.0.0-rc.27/reasoning-model-support`; cherry-picked from `d6c83c725fb7c54124152f7d91fb73c0371af049` |
| v1.0.0-rc.27 | Backup rebuild | [db3aab9ef7fb](../../commit/db3aab9ef7fb9fdf88469a4f81c60e7b242a2814) | 2026-08-30T18:44:01+08:00 | 2026-08-30T19:09:50+08:00 | test: cover OpenRouter GLM effort config | Backup-only; `refs/heads/feat/v1.0.0-rc.27/reasoning-model-support`; cherry-picked from `bbdcc7cdc968ef74eeff1e4e87f3a6a0f623291d` |
| v1.0.0-rc.27 | Backup rebuild | [aaf7038a62fd](../../commit/aaf7038a62fde0d5022f52751933120c09a777ba) | 2026-08-30T18:46:38+08:00 | 2026-08-30T19:09:50+08:00 | test: cover GLM channel boundaries | Backup-only; `refs/heads/feat/v1.0.0-rc.27/reasoning-model-support`; cherry-picked from `e11702d6decb935619ab1211433eb0b6f9dcf195` |
| v1.0.0-rc.27 | Backup rebuild | [407608a7f42f](../../commit/407608a7f42f173dbb199487189cc38dda62c8d2) | 2026-08-30T18:25:03+08:00 | 2026-08-30T19:09:49+08:00 | feat: parse generic GLM effort aliases | Backup-only; `refs/heads/feat/v1.0.0-rc.27/reasoning-model-support`; cherry-picked from `523feccf6ea713f5f71879584d16312a674e72d7` |
| v1.0.0-rc.27 | Backup rebuild | [4eed98f53a3c](../../commit/4eed98f53a3c7758b20c516218ce00011c7fdf6d) | 2026-08-30T18:32:45+08:00 | 2026-08-30T19:09:49+08:00 | feat: route generic GLM effort aliases | Backup-only; `refs/heads/feat/v1.0.0-rc.27/reasoning-model-support`; cherry-picked from `d63eade70d45da5120f023a6398cb1c4069abd95` |
| v1.0.0-rc.27 | Backup rebuild | [37334db1b293](../../commit/37334db1b29311354c878e0e4db4672c3dd4ea8e) | 2026-08-30T18:35:04+08:00 | 2026-08-30T19:09:49+08:00 | feat: map GLM aliases through base models | Backup-only; `refs/heads/feat/v1.0.0-rc.27/reasoning-model-support`; cherry-picked from `16c88e7f38c152bb5cd53c466d67c6ccd920b1d6` |
| v1.0.0-rc.27 | Backup rebuild | [a18ffb5c5a93](../../commit/a18ffb5c5a93c8c9307d09fdb1250534c67a7b97) | 2026-08-30T18:39:15+08:00 | 2026-08-30T19:09:49+08:00 | feat: add GLM effort to Zhipu requests | Backup-only; `refs/heads/feat/v1.0.0-rc.27/reasoning-model-support`; cherry-picked from `c33f9553d53ec1ab2ac08354e3c53260fe501f82` |
| v1.0.0-rc.27 | Backup rebuild | [dd707778b451](../../commit/dd707778b451daa1fce79ef7e6e2921d1e2adc88) | 2026-08-30T18:42:26+08:00 | 2026-08-30T19:09:49+08:00 | feat: relay GLM effort through OpenAI chat | Backup-only; `refs/heads/feat/v1.0.0-rc.27/reasoning-model-support`; cherry-picked from `b6f58d87726b44e941016c28862861b1a739a718` |
| v1.0.0-rc.27 | Carry snapshot | [0cb154df4dc7](../../commit/0cb154df4dc7a59bd4e62b4d06b06191658fda95) | 2026-08-30T19:02:50+08:00 | 2026-08-30T19:02:50+08:00 | feat: carry rc27 reasoning model support | Backup-only; `refs/heads/feat/v1.0.0-rc.27/reasoning-model-support`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.27 | Carry snapshot | [770cefb65001](../../commit/770cefb65001fb22720a774a29dccb5e513c3126) | 2026-08-19T18:50:04+08:00 | 2026-08-30T18:18:14+08:00 | fix: refresh rc27 usage logs in realtime | Backup-only; `refs/heads/fix/v1.0.0-rc.27/usage-logs-realtime-refresh`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.27 | Carry snapshot | [01a3db206f80](../../commit/01a3db206f807c6e31571ea266c7fd5be780f605) | 2026-08-19T18:46:46+08:00 | 2026-08-30T18:15:41+08:00 | feat: add rc27 responses compatibility | Backup-only; `refs/heads/feat/v1.0.0-rc.27/chatcompletions-responses-compat`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.27 | Carry snapshot | [d101e7b005a0](../../commit/d101e7b005a0dde42c02937cb7e34bcc7127f3af) | 2026-08-19T18:50:22+08:00 | 2026-08-30T18:15:27+08:00 | fix: isolate rc27 affinity tests | Backup-only; `refs/heads/fix/v1.0.0-rc.27/channel-affinity-test-isolation`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.26 | Docs | [c0b9df91c8b0](../../commit/c0b9df91c8b0e9e4cc84d71740d7ffaacdb6d2b7) | 2026-08-26T23:06:31+08:00 | 2026-08-26T23:06:31+08:00 | docs: record rc26 baseline exception | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.26 | Integration | [b4e1486c4f1c](../../commit/b4e1486c4f1c91d3ef485b77f4027bf90d02d709) | 2026-08-26T22:47:25+08:00 | 2026-08-26T22:47:25+08:00 | chore: merge upstream v1.0.0-rc.26 | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.26 | Docs | [fd81ba893e64](../../commit/fd81ba893e649c3cb9cb946dec023e60a2fa7b5f) | 2026-08-26T22:45:18+08:00 | 2026-08-26T22:45:18+08:00 | docs: plan rc26 upgrade | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.26 | Docs | [e595d819211d](../../commit/e595d819211dba119c8f9e9543a287e0c1122798) | 2026-08-26T22:36:00+08:00 | 2026-08-26T22:36:00+08:00 | docs: design rc26 upgrade | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.26 | Carry snapshot | [a478fe451308](../../commit/a478fe4513088d15582b7a372c319e700bc0a62b) | 2026-08-19T18:50:22+08:00 | 2026-08-26T23:18:40+08:00 | fix: channel affinity test isolation for rc26 | Backup-only; `refs/heads/fix/v1.0.0-rc.26/channel-affinity-test-isolation`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.26 | Carry snapshot | [7f68d21c5e84](../../commit/7f68d21c5e84ee846c4412d8088873fd4ab69186) | 2026-08-19T18:50:04+08:00 | 2026-08-26T23:16:35+08:00 | fix: usage logs realtime refresh for rc26 | Backup-only; `refs/heads/fix/v1.0.0-rc.26/usage-logs-realtime-refresh`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.26 | Carry snapshot | [d2c2a1c4d60c](../../commit/d2c2a1c4d60c10c62bfc87829ecf21c9e9dd2a1c) | 2026-08-19T18:46:46+08:00 | 2026-08-26T23:14:51+08:00 | feat: chat completions responses compat for rc26 | Backup-only; `refs/heads/feat/v1.0.0-rc.26/chatcompletions-responses-compat`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.26 | Carry snapshot | [189797b31564](../../commit/189797b3156475cb723976099a714bf9440983de) | 2026-08-19T18:45:00+08:00 | 2026-08-26T23:10:45+08:00 | feat: reasoning model support for v1.0.0-rc.26 | Backup-only; `refs/heads/feat/v1.0.0-rc.26/reasoning-model-support`; carry snapshot / canonical backup provenance |
| v1.0.0-rc.25 | Governance | [7352232c5d3f](../../commit/7352232c5d3fa185f26a153640090066b37b25ff) | 2026-08-19T18:41:07+08:00 | 2026-08-19T18:41:07+08:00 | docs: group backups by theme | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.25 | Integration | [fbaea03172fb](../../commit/fbaea03172fb4204041a28f77b39209096f4830d) | 2026-08-19T18:30:22+08:00 | 2026-08-19T18:30:22+08:00 | release: promote GLM-5.3 effort suffixes | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.25 | Feature | [b9236f0f61fc](../../commit/b9236f0f61fc92f3ee84dc1cd4581bb6053c67f5) | 2026-08-19T18:30:00+08:00 | 2026-08-19T18:30:00+08:00 | feat: add GLM-5.3 effort suffixes | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.25 | Test | [ca5026c1606e](../../commit/ca5026c1606e8d1d2646981e17f093638a1d6908) | 2026-08-19T18:21:40+08:00 | 2026-08-19T18:21:40+08:00 | test: cover GLM-5.3 effort suffixes | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.25 | Docs | [7ea510fa3416](../../commit/7ea510fa3416f01142e63de6034e4a2770d77ad5) | 2026-08-19T18:19:19+08:00 | 2026-08-19T18:19:19+08:00 | docs: design GLM-5.3 effort suffixes | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.25 | Test | [c31ec9e81309](../../commit/c31ec9e813096147e1da029934daa98a04291a13) | 2026-08-19T18:16:54+08:00 | 2026-08-19T18:16:54+08:00 | test(web): migrate usage log tests to Vitest | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.25 | Test | [0c44f42f3456](../../commit/0c44f42f34562a1c6fc438ce5b12d14ad330ec9d) | 2026-08-19T18:09:29+08:00 | 2026-08-19T18:09:29+08:00 | test: drop retired compact pricing precedence | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.25 | Governance | [6b418c2cf23c](../../commit/6b418c2cf23c8afed69fe25ec5887637419e7f4e) | 2026-08-19T18:05:52+08:00 | 2026-08-19T18:05:52+08:00 | docs: forbid upstream pull requests | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.25 | Integration | [c936432d3659](../../commit/c936432d36591f63eb112201ce86c5d052dd6292) | 2026-08-19T18:05:09+08:00 | 2026-08-19T18:05:09+08:00 | chore: merge upstream v1.0.0-rc.25 | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Integration | [f4daf51fc6ff](../../commit/f4daf51fc6ffc658446b3dc2a168538811ab9a3c) | 2026-08-14T10:11:42+08:00 | 2026-08-14T10:11:42+08:00 | chore: promote Kimi K3 none suffix | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Feature | [79e94e27b8ca](../../commit/79e94e27b8ca193977908b983f6f5152c7c016c1) | 2026-08-14T09:58:08+08:00 | 2026-08-14T10:07:59+08:00 | feat: add Kimi K3 none suffix | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Integration | [e1c0767de16c](../../commit/e1c0767de16c99c7fefad6306c9c701368b1d6a9) | 2026-08-14T05:14:22+08:00 | 2026-08-14T05:14:22+08:00 | chore: promote GLM-5.2 effort suffixes | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Fix | [0c8d0cc109d0](../../commit/0c8d0cc109d02049c1451d42899e46a9676566eb) | 2026-08-14T03:45:50+08:00 | 2026-08-14T03:45:50+08:00 | fix: preserve GLM alias retry priority | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Fix | [6c7054468f3d](../../commit/6c7054468f3dbf1b5d16b0dba88546aaadd3b4c6) | 2026-08-14T03:38:00+08:00 | 2026-08-14T03:38:00+08:00 | fix: limit GLM aliases to Zhipu V4 | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Fix | [4dea093de3c7](../../commit/4dea093de3c739979dece3d97fdd7a3a278e4e70) | 2026-08-14T02:53:27+08:00 | 2026-08-14T02:53:27+08:00 | fix: guard GLM-5.2 relay metadata | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Feature | [05cbbf6cbb0c](../../commit/05cbbf6cbb0c94a3bc5b7d40819230e138fa0c1f) | 2026-08-14T02:43:37+08:00 | 2026-08-14T02:43:37+08:00 | feat: add GLM-5.2 effort aliases | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Feature | [ea771fc2db91](../../commit/ea771fc2db910833e33561cd51c4cca42ba34bd6) | 2026-08-14T02:33:18+08:00 | 2026-08-14T02:33:18+08:00 | feat: parse GLM-5.2 effort suffixes | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Docs | [8bb827cf5b32](../../commit/8bb827cf5b3221187c610e36560ead9fec6a87b8) | 2026-08-14T02:18:39+08:00 | 2026-08-14T02:21:48+08:00 | docs: design GLM-5.2 effort suffixes | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Integration | [bcf2ed95dc81](../../commit/bcf2ed95dc81f8424095dcf32c071cd3c82684b2) | 2026-08-14T01:04:16+08:00 | 2026-08-14T01:04:16+08:00 | chore: promote Grok effort suffixes | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Feature | [b53436d02375](../../commit/b53436d02375f1bce2477fb821404a1e18604781) | 2026-08-14T01:02:20+08:00 | 2026-08-14T01:02:20+08:00 | feat: add Grok effort suffixes | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Integration | [160937517778](../../commit/160937517778c2a1a86e89402025208995e29eda) | 2026-08-13T13:38:23+08:00 | 2026-08-13T13:38:23+08:00 | chore: promote rc.24 candidate | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Test | [2e58c0dfd2d6](../../commit/2e58c0dfd2d6233f9342e7ba69792c9fc8505ea8) | 2026-08-13T13:34:57+08:00 | 2026-08-13T13:34:57+08:00 | test: isolate affinity cache counters | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Feature | [b54f87ecd006](../../commit/b54f87ecd006ee0aa945e905515f5ad719f0a97f) | 2026-08-13T13:27:15+08:00 | 2026-08-13T13:27:15+08:00 | feat: add DeepSeek low effort suffix | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.24 | Integration | [9c42a8083486](../../commit/9c42a808348676e561ca4b53ed017a8918278ed6) | 2026-08-13T13:21:18+08:00 | 2026-08-13T13:21:18+08:00 | chore: merge upstream rc.24 | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Integration | [2013b9a29194](../../commit/2013b9a2919468af1e824acf32405d400dc6bec2) | 2026-07-25T03:19:38+08:00 | 2026-07-25T03:19:38+08:00 | feat: release Claude Opus 5 pricing | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Feature | [035ee56654e3](../../commit/035ee56654e31c4b11eeb8c9dcea1efbf9f24a9f) | 2026-07-25T03:12:33+08:00 | 2026-07-25T03:14:39+08:00 | feat: add Claude Opus 5 pricing | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Integration | [b69297ac8cef](../../commit/b69297ac8cef8af880ed9044d0127f64fe432f34) | 2026-07-23T05:12:05+08:00 | 2026-07-23T05:12:05+08:00 | feat: release K3 effort suffix aliases | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Feature | [89e518e20646](../../commit/89e518e20646eed40c298d7f3ff84b65210bd7fe) | 2026-07-23T05:09:37+08:00 | 2026-07-23T05:09:37+08:00 | feat: add K3 effort suffix aliases | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Integration | [46833cedc697](../../commit/46833cedc697839ebba99ddc1ab1a38e793ca0c8) | 2026-07-20T03:12:28+08:00 | 2026-07-20T03:12:28+08:00 | chore(release): integrate zeabur deployment map docs | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Governance | [8fac33ce047b](../../commit/8fac33ce047b647a67486366ba8eb025df43bfa8) | 2026-07-20T03:12:18+08:00 | 2026-07-20T03:12:18+08:00 | docs: record zeabur dev/release deployment map | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Integration | [f857d4c477af](../../commit/f857d4c477af95ffb8eb6fd41a1bd985666624a0) | 2026-07-20T02:28:55+08:00 | 2026-07-20T02:28:55+08:00 | chore(release): promote log stream refresh | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Governance | [bafea1f0c477](../../commit/bafea1f0c47799e8591fe18b62154a0eea2910e0) | 2026-07-20T02:14:53+08:00 | 2026-07-20T02:14:53+08:00 | docs: record local testing procedure | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Fix | [5be3f1e7d695](../../commit/5be3f1e7d695dee9bc7b240d94edcb5b83b1745e) | 2026-07-20T02:14:13+08:00 | 2026-07-20T02:14:13+08:00 | fix: stream usage log updates | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Governance | [935bf63036ed](../../commit/935bf63036ed7bd69198757dbc65e4ce6f214c94) | 2026-07-20T01:25:42+08:00 | 2026-07-20T01:25:42+08:00 | docs: keep only active version branches | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Integration | [99af294b7bd1](../../commit/99af294b7bd158ec69f7bb9cd61c8be0f563a353) | 2026-07-20T00:56:31+08:00 | 2026-07-20T00:56:31+08:00 | chore(release): integrate usage logs fix | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Fix | [3285b198e9a9](../../commit/3285b198e9a914919d231d6cee1c5644f182c89e) | 2026-07-20T00:54:05+08:00 | 2026-07-20T00:54:05+08:00 | fix: refresh usage logs after tests | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Feature | [54174ef5d4d1](../../commit/54174ef5d4d1b0e347538fa6581f82110114679e) | 2026-07-19T11:42:18+08:00 | 2026-07-19T11:42:18+08:00 | feat: add Kimi reasoning modes | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Feature | [e479446253f9](../../commit/e479446253f9a8b55dcbe1d335c66e8b1558720e) | 2026-07-19T11:20:48+08:00 | 2026-07-19T11:20:48+08:00 | feat: make K3 default to no reasoning | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Fix | [c329e717efa5](../../commit/c329e717efa520223c986cce3204fb9ed581530d) | 2026-07-19T03:07:47+08:00 | 2026-07-19T03:07:47+08:00 | fix: normalize Kimi fixed parameters | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Feature | [6d07f6f079cf](../../commit/6d07f6f079cf19a642672f4239ba4fe72f0235bb) | 2026-07-19T02:05:49+08:00 | 2026-07-19T02:05:49+08:00 | feat: add Moonshot thinking suffixes | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Feature | [682204ac367c](../../commit/682204ac367c228aa7daee778b5e484bc9a1cccb) | 2026-07-18T22:53:00+08:00 | 2026-07-18T22:53:00+08:00 | feat: support Kimi K3 max reasoning effort | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Governance | [3e55e6428489](../../commit/3e55e6428489f0528681dec8f919d015d1cf26ef) | 2026-07-18T21:37:16+08:00 | 2026-07-18T21:37:16+08:00 | docs: document agent skills and issue tracking Co-authored-by: Codex &lt;noreply@openai.com&gt; | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Governance | [46dc0c793c5c](../../commit/46dc0c793c5cf7e4c872a04293adc1846995d719) | 2026-07-18T17:39:11+08:00 | 2026-07-18T17:39:11+08:00 | docs: define versioned release workflow | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Governance | [2968c851a74d](../../commit/2968c851a74d5f6ee04150f3886842e6ecff4125) | 2026-07-13T04:19:12+08:00 | 2026-07-13T04:22:38+08:00 | docs: preserve all rc feature branches | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.21 | Integration | [27d6c0220d38](../../commit/27d6c0220d381f3c95aa047feed93506105ebc20) | 2026-07-13T03:50:25+08:00 | 2026-07-13T03:53:50+08:00 | Merge tag 'v1.0.0-rc.21' into release/v1.0.0-rc.21 | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.20 | Feature | [d1a0dfd547aa](../../commit/d1a0dfd547aa69f9dadff6360905c22847e06e2b) | 2026-07-11T04:13:58+08:00 | 2026-07-11T04:13:58+08:00 | feat: add wildcard model pricing | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.20 | Feature | [7c6b82bcbc7e](../../commit/7c6b82bcbc7ea8b9f1b05d0498d9394301d35385) | 2026-07-10T21:42:09+08:00 | 2026-07-10T21:42:09+08:00 | feat: add GPT-5.6 reasoning suffixes | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.20 | Governance | [60489207ec3e](../../commit/60489207ec3e775c2e407c422a4c0e7b783cd5ff) | 2026-07-07T22:25:31+08:00 | 2026-07-07T22:25:31+08:00 | docs: add release carry-forward policy Co-authored-by: Codex &lt;noreply@openai.com&gt; | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.20 | Integration | [5705d3a40d78](../../commit/5705d3a40d78b9999556e10a02a7733d9496cc68) | 2026-07-07T22:10:43+08:00 | 2026-07-07T22:10:43+08:00 | Merge tag 'v1.0.0-rc.20' | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.18 | Integration | [17a0d49c4c56](../../commit/17a0d49c4c56665007858ddf536fff65d8a05309) | 2026-07-07T12:50:32+08:00 | 2026-07-07T12:50:32+08:00 | Merge tag 'v1.0.0-rc.18' | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.15 | Feature | [58e14b1294a8](../../commit/58e14b1294a8e48b5c588fd8d5fd771d815c0cc6) | 2026-07-01T13:07:48+08:00 | 2026-07-01T13:07:48+08:00 | feat: add Claude Sonnet 5 support | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.15 | Fix | [612da8085a89](../../commit/612da8085a89bb4faeedeaf2218dba1827eafcc8) | 2026-06-21T05:53:27+08:00 | 2026-07-01T12:58:25+08:00 | fix: fill classic zh-TW translations | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.15 | Feature | [16c884b0d66c](../../commit/16c884b0d66c05ee1f347bb4ba6927efeb8f9400) | 2026-06-21T05:51:37+08:00 | 2026-07-01T12:58:25+08:00 | feat: add Claude adaptive Fable models | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.15 | Fix | [7390b78a4f64](../../commit/7390b78a4f64ce7827a4ad1646284c89bae01eb5) | 2026-06-21T05:48:36+08:00 | 2026-07-01T12:58:25+08:00 | fix: align responses reasoning pointer | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.15 | Fix | [93a7cd118fa6](../../commit/93a7cd118fa65f22a9f17ed63810c57874a37d78) | 2026-04-28T01:19:03+08:00 | 2026-07-01T12:58:25+08:00 | fix: preserve responses conversion details | Released; `refs/heads/release/v1.0.0-rc.30` snapshot ancestry at `75cca222f607b703cef48bcd3e478b101e28f062` |
| v1.0.0-rc.25 | Feature | [5fd1641ae442](../../commit/5fd1641ae442a24ee7d5c70e7a70186ae6417af1) | 2026-08-20T11:47:24+08:00 | 2026-08-20T11:47:24+08:00 | feat: serve GLM effort beyond Zhipu V4 | Unreleased; target `v1.0.0-rc.25`; `refs/heads/feat/rc25/glm-effort-openai-openrouter` |
| v1.0.0-rc.25 | Test | [7ceecb3b7174](../../commit/7ceecb3b7174c458f3a0ade4a297a90cec91ad86) | 2026-08-20T11:47:23+08:00 | 2026-08-20T11:47:23+08:00 | test: cover GLM effort beyond Zhipu V4 | Unreleased; target `v1.0.0-rc.25`; `refs/heads/feat/rc25/glm-effort-openai-openrouter` |
| v1.0.0-rc.25 | Docs | [552bc0dc411e](../../commit/552bc0dc411ef9880f5ddc97801fd2d2f48cbe11) | 2026-08-20T11:47:10+08:00 | 2026-08-20T11:47:10+08:00 | docs: design GLM effort beyond Zhipu V4 | Unreleased; target `v1.0.0-rc.25`; `refs/heads/feat/rc25/glm-effort-openai-openrouter` |

## 可重現參考資料

本參考區塊與技術帳本同於 `2026-09-02T23:26:07.0513037+08:00` 更新。157-OID snapshot 的 signed release 上界固定為 `50f0f0d13f09b08601bc15ce2e40bf989e351e3e`；最終 `docs: refresh zeta changelog` 提交與其後提交不在本帳本內。

集合計數為 Released 98、Backup-only 56、Unreleased 3，共 157 個唯一 OID。完整性結果為 duplicates=0、missing=0、extra=0、unclassified=0，兩個 malformed archive 與帳本 overlap=0。

rc.26 epoch 從 `e595d819211dba119c8f9e9543a287e0c1122798` 開始，並包含 `fd81ba893e649c3cb9cb946dec023e60a2fa7b5f`。rc.27 epoch 從 `3084ef43418370ebc3ab9c266ad11105df2a207c` 開始，並在 tag merge 前包含 `74e223c6cdff076ea55e611074f127e80e69ed63` 與 `7add487c63e295287b4a0419ae5d983eac6040c9`。

### 本地來源 ref tips

| Ref | Full OID |
| --- | --- |
| `refs/heads/dev/v1.0.0-rc.26` | `c0b9df91c8b0e9e4cc84d71740d7ffaacdb6d2b7` |
| `refs/heads/dev/v1.0.0-rc.27` | `15ccc25556359fa097d039426966d618c5e7d2b8` |
| `refs/heads/dev/v1.0.0-rc.29` | `55c0dea9d8573d9b0f55fd2891d906938e00a850` |
| `refs/heads/dev/v1.0.0-rc.30` | `f9f1a465e7572569a7392d79024697fdb5e7a1e7` |
| `refs/heads/feat/rc25/glm-effort-openai-openrouter` | `5fd1641ae442a24ee7d5c70e7a70186ae6417af1` |
| `refs/heads/feat/rc27/glm-generic-effort-suffixes` | `f2a4893e73f1efc840ce50274d310687c21443e4` |
| `refs/heads/feat/v1.0.0-rc.26/chatcompletions-responses-compat` | `d2c2a1c4d60c10c62bfc87829ecf21c9e9dd2a1c` |
| `refs/heads/feat/v1.0.0-rc.26/reasoning-model-support` | `189797b3156475cb723976099a714bf9440983de` |
| `refs/heads/feat/v1.0.0-rc.27/chatcompletions-responses-compat` | `01a3db206f807c6e31571ea266c7fd5be780f605` |
| `refs/heads/feat/v1.0.0-rc.27/reasoning-model-support` | `e6fe18d23e016023f1542adee4aaba7d5d726dc5` |
| `refs/heads/feat/v1.0.0-rc.29/chatcompletions-responses-compat` | `965b46fd5bb3188cfb779b86b03f88fab8debb31` |
| `refs/heads/feat/v1.0.0-rc.29/reasoning-model-support` | `91450384951ac181a8d25ce2774743afb6e3f62c` |
| `refs/heads/feat/v1.0.0-rc.30/chatcompletions-responses-compat` | `f2f151c1df31e0e3f28e1eee53e58ecc3de11795` |
| `refs/heads/feat/v1.0.0-rc.30/reasoning-model-support` | `c19ae9c73f49af76542e767b3c82ba1c790011fc` |
| `refs/heads/fix/rc29/postgres-automigrate-compat` | `8e026e7c6abb6a8129bf6b39834ee5c05563aa6e` |
| `refs/heads/fix/rc30/production-review-gaps` | `6f2abbdf9810a5cc51b9c2cccdb4368bc7ca4870` |
| `refs/heads/fix/rc30/nullable-priority-selection` | `f9f1a465e7572569a7392d79024697fdb5e7a1e7` |
| `refs/heads/fix/rc30/rollback-bridge-snapshot` | `35869c53edb2b3da775f071e2f4c1c86178e07d1` |
| `refs/heads/fix/v1.0.0-rc.26/channel-affinity-test-isolation` | `a478fe4513088d15582b7a372c319e700bc0a62b` |
| `refs/heads/fix/v1.0.0-rc.26/usage-logs-realtime-refresh` | `7f68d21c5e84ee846c4412d8088873fd4ab69186` |
| `refs/heads/fix/v1.0.0-rc.27/channel-affinity-test-isolation` | `d101e7b005a0dde42c02937cb7e34bcc7127f3af` |
| `refs/heads/fix/v1.0.0-rc.27/usage-logs-realtime-refresh` | `770cefb65001fb22720a774a29dccb5e513c3126` |
| `refs/heads/fix/v1.0.0-rc.29/channel-affinity-test-isolation` | `bc4e43801de037257a6b00f6b44c23b8482b5cce` |
| `refs/heads/fix/v1.0.0-rc.29/postgres-automigrate-compat` | `0a8422e16490cc7854dd20c9d0ca5767498dea9e` |
| `refs/heads/fix/v1.0.0-rc.29/usage-logs-realtime-refresh` | `78e4960269ed35c6ea1d90b10205fc347fd76976` |
| `refs/heads/fix/v1.0.0-rc.30/channel-affinity-test-isolation` | `78ea940348058f04037b90cb87ad8b90aa466c72` |
| `refs/heads/fix/v1.0.0-rc.30/postgres-automigrate-compat` | `8615f9a843efd8f5bed2e604f96e6b46507fbc55` |
| `refs/heads/fix/v1.0.0-rc.30/usage-logs-realtime-refresh` | `6b6a176e74433e45c30b3dff874815d5e0fc9d4d` |
| `refs/heads/release/v1.0.0-rc.26` | `c0b9df91c8b0e9e4cc84d71740d7ffaacdb6d2b7` |
| `refs/heads/release/v1.0.0-rc.27` | `15ccc25556359fa097d039426966d618c5e7d2b8` |
| `refs/heads/release/v1.0.0-rc.29` | `dccf03595850addfd0901523e5ace279ecb9da83` |
| `refs/heads/release/v1.0.0-rc.30` | `50f0f0d13f09b08601bc15ce2e40bf989e351e3e` |

### Origin remote-tracking ref tips

| Ref | Full OID |
| --- | --- |
| `refs/remotes/origin/dev/v1.0.0-rc.26` | `c0b9df91c8b0e9e4cc84d71740d7ffaacdb6d2b7` |
| `refs/remotes/origin/dev/v1.0.0-rc.27` | `15ccc25556359fa097d039426966d618c5e7d2b8` |
| `refs/remotes/origin/dev/v1.0.0-rc.29` | `55c0dea9d8573d9b0f55fd2891d906938e00a850` |
| `refs/remotes/origin/dev/v1.0.0-rc.30` | `da443762c38078ba4552b381a6c9a2b22ff9189c` |
| `refs/remotes/origin/feat/rc25/glm-effort-openai-openrouter` | `5fd1641ae442a24ee7d5c70e7a70186ae6417af1` |
| `refs/remotes/origin/feat/v1.0.0-rc.26/chatcompletions-responses-compat` | `d2c2a1c4d60c10c62bfc87829ecf21c9e9dd2a1c` |
| `refs/remotes/origin/feat/v1.0.0-rc.26/reasoning-model-support` | `189797b3156475cb723976099a714bf9440983de` |
| `refs/remotes/origin/feat/v1.0.0-rc.29/chatcompletions-responses-compat` | `965b46fd5bb3188cfb779b86b03f88fab8debb31` |
| `refs/remotes/origin/feat/v1.0.0-rc.29/reasoning-model-support` | `91450384951ac181a8d25ce2774743afb6e3f62c` |
| `refs/remotes/origin/feat/v1.0.0-rc.30/chatcompletions-responses-compat` | `f2f151c1df31e0e3f28e1eee53e58ecc3de11795` |
| `refs/remotes/origin/feat/v1.0.0-rc.30/reasoning-model-support` | `d283b48067f9410a0e4bc09b8555167c009fc1cd` |
| `refs/remotes/origin/fix/v1.0.0-rc.26/channel-affinity-test-isolation` | `a478fe4513088d15582b7a372c319e700bc0a62b` |
| `refs/remotes/origin/fix/v1.0.0-rc.26/usage-logs-realtime-refresh` | `7f68d21c5e84ee846c4412d8088873fd4ab69186` |
| `refs/remotes/origin/fix/v1.0.0-rc.29/channel-affinity-test-isolation` | `bc4e43801de037257a6b00f6b44c23b8482b5cce` |
| `refs/remotes/origin/fix/v1.0.0-rc.29/postgres-automigrate-compat` | `0a8422e16490cc7854dd20c9d0ca5767498dea9e` |
| `refs/remotes/origin/fix/v1.0.0-rc.29/usage-logs-realtime-refresh` | `78e4960269ed35c6ea1d90b10205fc347fd76976` |
| `refs/remotes/origin/fix/v1.0.0-rc.30/channel-affinity-test-isolation` | `78ea940348058f04037b90cb87ad8b90aa466c72` |
| `refs/remotes/origin/fix/v1.0.0-rc.30/postgres-automigrate-compat` | `8615f9a843efd8f5bed2e604f96e6b46507fbc55` |
| `refs/remotes/origin/fix/v1.0.0-rc.30/usage-logs-realtime-refresh` | `6b6a176e74433e45c30b3dff874815d5e0fc9d4d` |
| `refs/remotes/origin/release/v1.0.0-rc.26` | `c0b9df91c8b0e9e4cc84d71740d7ffaacdb6d2b7` |
| `refs/remotes/origin/release/v1.0.0-rc.29` | `dccf03595850addfd0901523e5ace279ecb9da83` |
| `refs/remotes/origin/release/v1.0.0-rc.30` | `35869c53edb2b3da775f071e2f4c1c86178e07d1` |

### Main、upstream tracking 與目標 tag

| Ref | Full OID |
| --- | --- |
| `refs/heads/main` | `f116414284162ad15d8925f7bca494c109b83e93` |
| `refs/remotes/origin/main` | `27ff6a8767e728f879d52770c273d4f73214a430` |
| `refs/remotes/upstream/HEAD` → `refs/remotes/upstream/main` | `2b6f1dfefbe217fed31fc0726717cc7de6958e8e` |
| `refs/remotes/upstream/main` | `2b6f1dfefbe217fed31fc0726717cc7de6958e8e` |
| `refs/tags/v1.0.0-rc.30` | `27ff6a8767e728f879d52770c273d4f73214a430` |

### Live remote readback

| Remote ref | Full OID |
| --- | --- |
| `upstream:HEAD` | `0ed497f066a68613375124303ef54f220267b334` |
| `upstream:refs/heads/main` | `0ed497f066a68613375124303ef54f220267b334` |
| `origin:refs/heads/main` | `27ff6a8767e728f879d52770c273d4f73214a430` |

Live readback 顯示 origin rc.30 development 位於先前候選 `da443762c38078ba4552b381a6c9a2b22ff9189c`，release 仍為已發布的 Bridge SHA `35869c53edb2b3da775f071e2f4c1c86178e07d1`；五個 rc.30 backup refs 均存在，其中 PostgreSQL backup 已前進至 Contract tip `8615f9a843efd8f5bed2e604f96e6b46507fbc55`。本次 nullable-priority 修正尚未推送 development、reasoning backup 或 release。

### 已驗證 epoch tags

686 個本地 tag refs 全部可直接解析為 commit，或經由 annotated tag peel 解析為 commit；無效 tag refs 為 0。完整 `refs/tags/**` 命名空間均納入排除來源。與版本紀元相關的 tips 如下：

| Tag ref | Object type | Ref object OID | Peeled commit OID |
| --- | --- | --- | --- |
| `refs/tags/v1.0.0-rc.15` | tag | `9fb60192abec803f94f99a7ee86480bd9efdefc9` | `69b0f0b56f528efa292a2893feb0c55c37399f4b` |
| `refs/tags/v1.0.0-rc.18` | tag | `40aacc4cd1037115d88b6ca5f86e3c5bc0eefa65` | `c9943d37ad93477dd937fc4901cc3c4e0fd8aaab` |
| `refs/tags/v1.0.0-rc.20` | tag | `a7f3067bf34a2fa125f843acdfcf45d0b0bfd682` | `6ce7305cd36f16506fb6a2c3c524a5a318539ba7` |
| `refs/tags/v1.0.0-rc.21` | tag | `8a784d7c8f8da9041b30c6c6ea4c3b3df0d683de` | `bde9b2f44887d34ec54799ae191d50f97914359e` |
| `refs/tags/v1.0.0-rc.24` | tag | `21ee1f565b47663b2d9f791c0ddf593b096ebfe2` | `5c3abffe8572aa8a49f15c3916707d2019d66af4` |
| `refs/tags/v1.0.0-rc.25` | commit | `f116414284162ad15d8925f7bca494c109b83e93` | `f116414284162ad15d8925f7bca494c109b83e93` |
| `refs/tags/v1.0.0-rc.26` | commit | `8f6961c675932f406260ff0c218bc2aa0603e9b2` | `8f6961c675932f406260ff0c218bc2aa0603e9b2` |
| `refs/tags/v1.0.0-rc.27` | commit | `eb48396d5fe97d27772d0cd5e3ca8aa5caa4f3e9` | `eb48396d5fe97d27772d0cd5e3ca8aa5caa4f3e9` |
| `refs/tags/v1.0.0-rc.29` | commit | `2b6f1dfefbe217fed31fc0726717cc7de6958e8e` | `2b6f1dfefbe217fed31fc0726717cc7de6958e8e` |
| `refs/tags/v1.0.0-rc.30` | commit | `27ff6a8767e728f879d52770c273d4f73214a430` | `27ff6a8767e728f879d52770c273d4f73214a430` |

額外納入的 upstream 排除 refs：無。

### 157 個完整 OID

下列資料採 `Version|State|Full OID` 格式，順序與技術帳本完全相同。

```text
v1.0.0-rc.30|Backup-only|c19ae9c73f49af76542e767b3c82ba1c790011fc
v1.0.0-rc.30|Released|50f0f0d13f09b08601bc15ce2e40bf989e351e3e
v1.0.0-rc.30|Released|f9f1a465e7572569a7392d79024697fdb5e7a1e7
v1.0.0-rc.30|Backup-only|8615f9a843efd8f5bed2e604f96e6b46507fbc55
v1.0.0-rc.30|Released|85da814a0d8f0d7a10e6d479017aabcf3c829e88
v1.0.0-rc.30|Released|6f2abbdf9810a5cc51b9c2cccdb4368bc7ca4870
v1.0.0-rc.30|Released|35869c53edb2b3da775f071e2f4c1c86178e07d1
v1.0.0-rc.30|Backup-only|e19302477e75e411e9e08563b77e1ed9db177707
v1.0.0-rc.30|Released|61c001f8407980c750c40c238421134fc78bf1e1
v1.0.0-rc.30|Released|df818ccd0f11428ef5a7de9fd33179c56ff929af
v1.0.0-rc.30|Released|3c988a4f0d98e63bc20bb4e500ed25bd5d1be497
v1.0.0-rc.30|Released|91f41f7b680e509d1c389f88a8fef03beb1928eb
v1.0.0-rc.30|Released|a94284fe5c6b10112c43bfb7832caa82945c8dff
v1.0.0-rc.30|Released|97c9413e88ce48033d2638529c61810902adc842
v1.0.0-rc.30|Released|e18d488a6c1132d9267276f690fd0bf6659f7d91
v1.0.0-rc.30|Released|75cca222f607b703cef48bcd3e478b101e28f062
v1.0.0-rc.30|Released|57ebbda62da4afa2b5b41c113f3c26cc4c5c4f21
v1.0.0-rc.30|Released|43b18f09db5e42b7e6a01677b6498e41f567c165
v1.0.0-rc.30|Released|e8e2d3469cfaa8df759957731a6ff3c2306393e2
v1.0.0-rc.30|Backup-only|b37f6be3a309c89b60067e64bb80c47340800096
v1.0.0-rc.30|Backup-only|78ea940348058f04037b90cb87ad8b90aa466c72
v1.0.0-rc.30|Backup-only|6b6a176e74433e45c30b3dff874815d5e0fc9d4d
v1.0.0-rc.30|Backup-only|59ba761701a0a07b4046c813e6d19f8702a38e98
v1.0.0-rc.30|Backup-only|f2f151c1df31e0e3f28e1eee53e58ecc3de11795
v1.0.0-rc.30|Backup-only|d283b48067f9410a0e4bc09b8555167c009fc1cd
v1.0.0-rc.30|Backup-only|e809f06cbe471089888e421487908dc209e7f1bc
v1.0.0-rc.30|Backup-only|a1279e9980d34cab544b44e528bf0bbbcbf863ce
v1.0.0-rc.30|Backup-only|c6c8abff3fc5faa1d24617a49c5e3dcb2f860480
v1.0.0-rc.30|Backup-only|02cd75ee860f3ddd83cd9e0643db0fc4f607d027
v1.0.0-rc.30|Backup-only|2e5eb4593d1fbc672c014856f24328b2789a419a
v1.0.0-rc.30|Backup-only|8a1275feebc47e4880e03cf9d1b897260008ae89
v1.0.0-rc.30|Backup-only|e5a2429609d33422d34c0b8b74cf731ff7120d64
v1.0.0-rc.30|Backup-only|56633722ad0dd71a83fbac004ecc635b07faa57a
v1.0.0-rc.30|Backup-only|227348acbad64fa3abbd4673c354272bed6791a8
v1.0.0-rc.30|Backup-only|72c0d721a6186fa16ac474b04c33e3b106078c5c
v1.0.0-rc.30|Backup-only|c27f8fb79406dd6b6ab23a326fcbe6d672a70e84
v1.0.0-rc.30|Backup-only|9fb194b2fb28732d9e537bdd0ec8482172255496
v1.0.0-rc.29|Released|dccf03595850addfd0901523e5ace279ecb9da83
v1.0.0-rc.29|Released|55c0dea9d8573d9b0f55fd2891d906938e00a850
v1.0.0-rc.29|Released|8e026e7c6abb6a8129bf6b39834ee5c05563aa6e
v1.0.0-rc.29|Released|74373bad18bd878d610660d95f368efefb7757ad
v1.0.0-rc.29|Released|6916cbb2563121bf67fd921219124a0ec91268d9
v1.0.0-rc.29|Released|2823b818b5a5c23afddda994918f0e74b080ffc3
v1.0.0-rc.29|Released|505120faf7d6df3e3456b2037a154d0b3a1535ce
v1.0.0-rc.29|Released|e3ab5bc85d13e370a664ea37bcdc98f3bf09d043
v1.0.0-rc.29|Backup-only|0a8422e16490cc7854dd20c9d0ca5767498dea9e
v1.0.0-rc.29|Backup-only|78e4960269ed35c6ea1d90b10205fc347fd76976
v1.0.0-rc.29|Backup-only|bc4e43801de037257a6b00f6b44c23b8482b5cce
v1.0.0-rc.29|Backup-only|601b53b61e155075cc68bda5b3f4ee0c5add04ae
v1.0.0-rc.29|Backup-only|965b46fd5bb3188cfb779b86b03f88fab8debb31
v1.0.0-rc.29|Backup-only|91450384951ac181a8d25ce2774743afb6e3f62c
v1.0.0-rc.29|Backup-only|c0d641470e3419efba35ca767181ea49449e9f50
v1.0.0-rc.29|Backup-only|99184f095d652e2eafe7d23393b49ce6bf198f1c
v1.0.0-rc.29|Backup-only|8c5721f815d891e42599a0073c782a27cae0d166
v1.0.0-rc.29|Backup-only|0fd6e8d9fa9a843cebfe411a8d76a8a8af67bc23
v1.0.0-rc.29|Backup-only|5409dcbc1612c5dcb68965a09086ab06cbd226f3
v1.0.0-rc.29|Backup-only|0c258052d2ac43e2aa179557542083fa703b2cb5
v1.0.0-rc.29|Backup-only|5e19a240b5537ac2db405b082f3a0cd33693f251
v1.0.0-rc.29|Backup-only|b65ebfce607d2fd128c7d4e312d8f74543271152
v1.0.0-rc.29|Backup-only|d0e62e6cbaa296e3bbc154dd394bca725afd3477
v1.0.0-rc.29|Backup-only|80f411a0ba0c6f800969fa20774196dd085255b3
v1.0.0-rc.29|Backup-only|58388cc5749c069bf5dd98f3bcd4409c6f655944
v1.0.0-rc.29|Backup-only|c242c586a6e9b3a5ea1eba435f382c85e78b0cf6
v1.0.0-rc.27|Released|15ccc25556359fa097d039426966d618c5e7d2b8
v1.0.0-rc.27|Released|f2a4893e73f1efc840ce50274d310687c21443e4
v1.0.0-rc.27|Released|d6c83c725fb7c54124152f7d91fb73c0371af049
v1.0.0-rc.27|Released|e11702d6decb935619ab1211433eb0b6f9dcf195
v1.0.0-rc.27|Released|bbdcc7cdc968ef74eeff1e4e87f3a6a0f623291d
v1.0.0-rc.27|Released|b6f58d87726b44e941016c28862861b1a739a718
v1.0.0-rc.27|Released|c33f9553d53ec1ab2ac08354e3c53260fe501f82
v1.0.0-rc.27|Released|16c88e7f38c152bb5cd53c466d67c6ccd920b1d6
v1.0.0-rc.27|Released|d63eade70d45da5120f023a6398cb1c4069abd95
v1.0.0-rc.27|Released|523feccf6ea713f5f71879584d16312a674e72d7
v1.0.0-rc.27|Released|ceeb7a0c7aecb69610177d2e2c5b4f7f24894a0d
v1.0.0-rc.27|Released|12b7ac06e9af895c9aea47a963297cf5831fcfac
v1.0.0-rc.27|Released|7add487c63e295287b4a0419ae5d983eac6040c9
v1.0.0-rc.27|Released|74e223c6cdff076ea55e611074f127e80e69ed63
v1.0.0-rc.27|Released|3084ef43418370ebc3ab9c266ad11105df2a207c
v1.0.0-rc.27|Backup-only|e6fe18d23e016023f1542adee4aaba7d5d726dc5
v1.0.0-rc.27|Backup-only|1b27408105a2bac5d13dd3f7ab005711ce6805c9
v1.0.0-rc.27|Backup-only|db3aab9ef7fb9fdf88469a4f81c60e7b242a2814
v1.0.0-rc.27|Backup-only|aaf7038a62fde0d5022f52751933120c09a777ba
v1.0.0-rc.27|Backup-only|407608a7f42f173dbb199487189cc38dda62c8d2
v1.0.0-rc.27|Backup-only|4eed98f53a3c7758b20c516218ce00011c7fdf6d
v1.0.0-rc.27|Backup-only|37334db1b29311354c878e0e4db4672c3dd4ea8e
v1.0.0-rc.27|Backup-only|a18ffb5c5a93c8c9307d09fdb1250534c67a7b97
v1.0.0-rc.27|Backup-only|dd707778b451daa1fce79ef7e6e2921d1e2adc88
v1.0.0-rc.27|Backup-only|0cb154df4dc7a59bd4e62b4d06b06191658fda95
v1.0.0-rc.27|Backup-only|770cefb65001fb22720a774a29dccb5e513c3126
v1.0.0-rc.27|Backup-only|01a3db206f807c6e31571ea266c7fd5be780f605
v1.0.0-rc.27|Backup-only|d101e7b005a0dde42c02937cb7e34bcc7127f3af
v1.0.0-rc.26|Released|c0b9df91c8b0e9e4cc84d71740d7ffaacdb6d2b7
v1.0.0-rc.26|Released|b4e1486c4f1c91d3ef485b77f4027bf90d02d709
v1.0.0-rc.26|Released|fd81ba893e649c3cb9cb946dec023e60a2fa7b5f
v1.0.0-rc.26|Released|e595d819211dba119c8f9e9543a287e0c1122798
v1.0.0-rc.26|Backup-only|a478fe4513088d15582b7a372c319e700bc0a62b
v1.0.0-rc.26|Backup-only|7f68d21c5e84ee846c4412d8088873fd4ab69186
v1.0.0-rc.26|Backup-only|d2c2a1c4d60c10c62bfc87829ecf21c9e9dd2a1c
v1.0.0-rc.26|Backup-only|189797b3156475cb723976099a714bf9440983de
v1.0.0-rc.25|Released|7352232c5d3fa185f26a153640090066b37b25ff
v1.0.0-rc.25|Released|fbaea03172fb4204041a28f77b39209096f4830d
v1.0.0-rc.25|Released|b9236f0f61fc92f3ee84dc1cd4581bb6053c67f5
v1.0.0-rc.25|Released|ca5026c1606e8d1d2646981e17f093638a1d6908
v1.0.0-rc.25|Released|7ea510fa3416f01142e63de6034e4a2770d77ad5
v1.0.0-rc.25|Released|c31ec9e813096147e1da029934daa98a04291a13
v1.0.0-rc.25|Released|0c44f42f34562a1c6fc438ce5b12d14ad330ec9d
v1.0.0-rc.25|Released|6b418c2cf23c8afed69fe25ec5887637419e7f4e
v1.0.0-rc.25|Released|c936432d36591f63eb112201ce86c5d052dd6292
v1.0.0-rc.24|Released|f4daf51fc6ffc658446b3dc2a168538811ab9a3c
v1.0.0-rc.24|Released|79e94e27b8ca193977908b983f6f5152c7c016c1
v1.0.0-rc.24|Released|e1c0767de16c99c7fefad6306c9c701368b1d6a9
v1.0.0-rc.24|Released|0c8d0cc109d02049c1451d42899e46a9676566eb
v1.0.0-rc.24|Released|6c7054468f3dbf1b5d16b0dba88546aaadd3b4c6
v1.0.0-rc.24|Released|4dea093de3c739979dece3d97fdd7a3a278e4e70
v1.0.0-rc.24|Released|05cbbf6cbb0c94a3bc5b7d40819230e138fa0c1f
v1.0.0-rc.24|Released|ea771fc2db910833e33561cd51c4cca42ba34bd6
v1.0.0-rc.24|Released|8bb827cf5b3221187c610e36560ead9fec6a87b8
v1.0.0-rc.24|Released|bcf2ed95dc81f8424095dcf32c071cd3c82684b2
v1.0.0-rc.24|Released|b53436d02375f1bce2477fb821404a1e18604781
v1.0.0-rc.24|Released|160937517778c2a1a86e89402025208995e29eda
v1.0.0-rc.24|Released|2e58c0dfd2d6233f9342e7ba69792c9fc8505ea8
v1.0.0-rc.24|Released|b54f87ecd006ee0aa945e905515f5ad719f0a97f
v1.0.0-rc.24|Released|9c42a808348676e561ca4b53ed017a8918278ed6
v1.0.0-rc.21|Released|2013b9a2919468af1e824acf32405d400dc6bec2
v1.0.0-rc.21|Released|035ee56654e31c4b11eeb8c9dcea1efbf9f24a9f
v1.0.0-rc.21|Released|b69297ac8cef8af880ed9044d0127f64fe432f34
v1.0.0-rc.21|Released|89e518e20646eed40c298d7f3ff84b65210bd7fe
v1.0.0-rc.21|Released|46833cedc697839ebba99ddc1ab1a38e793ca0c8
v1.0.0-rc.21|Released|8fac33ce047b647a67486366ba8eb025df43bfa8
v1.0.0-rc.21|Released|f857d4c477af95ffb8eb6fd41a1bd985666624a0
v1.0.0-rc.21|Released|bafea1f0c47799e8591fe18b62154a0eea2910e0
v1.0.0-rc.21|Released|5be3f1e7d695dee9bc7b240d94edcb5b83b1745e
v1.0.0-rc.21|Released|935bf63036ed7bd69198757dbc65e4ce6f214c94
v1.0.0-rc.21|Released|99af294b7bd158ec69f7bb9cd61c8be0f563a353
v1.0.0-rc.21|Released|3285b198e9a914919d231d6cee1c5644f182c89e
v1.0.0-rc.21|Released|54174ef5d4d1b0e347538fa6581f82110114679e
v1.0.0-rc.21|Released|e479446253f9a8b55dcbe1d335c66e8b1558720e
v1.0.0-rc.21|Released|c329e717efa520223c986cce3204fb9ed581530d
v1.0.0-rc.21|Released|6d07f6f079cf19a642672f4239ba4fe72f0235bb
v1.0.0-rc.21|Released|682204ac367c228aa7daee778b5e484bc9a1cccb
v1.0.0-rc.21|Released|3e55e6428489f0528681dec8f919d015d1cf26ef
v1.0.0-rc.21|Released|46dc0c793c5cf7e4c872a04293adc1846995d719
v1.0.0-rc.21|Released|2968c851a74d5f6ee04150f3886842e6ecff4125
v1.0.0-rc.21|Released|27d6c0220d381f3c95aa047feed93506105ebc20
v1.0.0-rc.20|Released|d1a0dfd547aa69f9dadff6360905c22847e06e2b
v1.0.0-rc.20|Released|7c6b82bcbc7ea8b9f1b05d0498d9394301d35385
v1.0.0-rc.20|Released|60489207ec3e775c2e407c422a4c0e7b783cd5ff
v1.0.0-rc.20|Released|5705d3a40d78b9999556e10a02a7733d9496cc68
v1.0.0-rc.18|Released|17a0d49c4c56665007858ddf536fff65d8a05309
v1.0.0-rc.15|Released|58e14b1294a8e48b5c588fd8d5fd771d815c0cc6
v1.0.0-rc.15|Released|612da8085a89bb4faeedeaf2218dba1827eafcc8
v1.0.0-rc.15|Released|16c884b0d66c05ee1f347bb4ba6927efeb8f9400
v1.0.0-rc.15|Released|7390b78a4f64ce7827a4ad1646284c89bae01eb5
v1.0.0-rc.15|Released|93a7cd118fa65f22a9f17ed63810c57874a37d78
v1.0.0-rc.25|Unreleased|5fd1641ae442a24ee7d5c70e7a70186ae6417af1
v1.0.0-rc.25|Unreleased|7ceecb3b7174c458f3a0ade4a297a90cec91ad86
v1.0.0-rc.25|Unreleased|552bc0dc411ef9880f5ddc97801fd2d2f48cbe11
```

### 排除與清理清單

下列 refs 保存 Task 4 的兩次 malformed 嘗試，不得進入正式技術帳本。它們只作為等待擁有者核准精確 branch inventory 後的清理候選。

| Archive ref | Tip | Excluded OID count |
| --- | --- | ---: |
| `refs/heads/feat/rc30/reasoning-model-support-bad-message-archive` | `3a9cb7bf3cbe3e62dde22a55f327e00ccb5f5c76` | 13 |
| `refs/heads/feat/rc30/reasoning-model-support-duplicate-trailer-archive` | `59ac0efbd7cc44925397009fd2910f765a865fc7` | 13 |

### 錯誤訊息格式 archive OIDs

- `fa513d77a02ac56ef23c0068f7b1edda8afd4242`
- `8a41a0d30c33565b36f661aa5cdad500cfd065b8`
- `1d07e6210d0a2201cbf0b89b1727b80d941d2d69`
- `348e2f2ed92f4e9cd0019fdacf9d20e78c0ee571`
- `c605da67989702f785c4377f22df14de5cca7bea`
- `99bee4fdeac2d4a7f3c4610eee8e2163647e318d`
- `04ebc654dbff17cc57949577c44d665a3dd52540`
- `921606dfdb0f1c644a02f0d3cfaf8e449255ef48`
- `8ca972138c62879a01ff2aca4dae1bffe5892335`
- `b8f69da07b53a4358ddcb201f5c0be0cc2a9fd4a`
- `ac6883094a73648142afaa8db62ec3744df9f9ec`
- `e24d57de6d6c7f0cee20cdf679d3dccbb4ed6657`
- `3a9cb7bf3cbe3e62dde22a55f327e00ccb5f5c76`

### 重複 trailer archive OIDs

- `36b96f657b08d4a5f09bd2961beb7f279c566758`
- `c06d904ad7d6d30973e65cc6590699c020aadc2b`
- `f4bb426458f1bbff32b6ba5cdf2b2b8b5ab59bb1`
- `55cac6cd833f3873ced72bf855a764a6cf5dbd6b`
- `8d2608984c2e2b3df04d28f6cad52bda841773a6`
- `2496e6567a881839d86f8477cda6d8590bb3ed6b`
- `47d5b4e7274dc0ef88f0ae2353724e1eb19e121a`
- `b98411be226ca33dd0b04ca931730445e0798c69`
- `e0268c13954a6fab493a960713d2579c0a83e47a`
- `8ed0d9e5b85206777387e6dd10077efbce6491fc`
- `580c3def20a923fcdacb00cc4306861995c1f82b`
- `22437b034b4fb34edbc1eac33ac23f5f27b26017`
- `59ac0efbd7cc44925397009fd2910f765a865fc7`

Archive 與正式帳本的交集為 0 個 OID，兩組 archive 之間的重複數亦為 0。
