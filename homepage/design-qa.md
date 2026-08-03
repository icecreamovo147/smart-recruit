# Smart Recruit 主页 Design QA

## 重构说明（2026-08-03）

主页完成视觉世界替换，采用用户选定并批准的 **Deployable Hiring System / Miura fold** 方向：把平台表达为从金色部署包中展开的招聘系统，依次显露人才旅程、可信 AI 治理与开放架构。视觉语言使用哑光纸面、深机构绿、折痕蓝与克制金箔，以工程化折叠结构替代通用 SaaS 卡片堆叠。

- 方向决策：Impeccable concept-seed（seed key `db6c7cf2`），用户选择 `challenger-miura`，并在三张高保真构图中批准方案 A。
- 产品事实：`PRODUCT.md`；双受众、双语、双主题、现有文案与功能契约均保留。
- 设计系统：`DESIGN.md` + `.impeccable/design.json`，在最终评审通过后从已实现页面重新提取。
- 生成资产：`src/assets/miura-deploy-hero.webp` 与 `src/assets/gold-foil.webp`；生成说明记录于 `.impeccable/assets-manifest.md`。

## 验收视口

- `design-qa/new-desktop-light-zh.png`：1440×1024 桌面亮色中文首屏
- `design-qa/new-desktop-dark-en.png`：1440×1024 桌面暗色英文首屏
- `design-qa/new-mobile-dark-en.png`：390×844 移动端暗色英文首屏
- `design-qa/new-mobile-dark-en-menu.png`：390×844 移动端展开导航

## 浏览器验证（全部通过）

- 1440px 与 390px 均无横向溢出（`scrollWidth === innerWidth`）。
- 1440×1024 首屏中，Capabilities 标题顶部为 945px，能明确预告下一段系统能力。
- 移动端主图清晰，人才旅程 / 可信 AI / 开放架构三层图例均保留。
- 移动菜单保留当前语言对应的品牌名，并包含 Product、AI、Architecture、Open Source、Docs、语言与 GitHub。
- 移动菜单支持 Esc 关闭；主题与语言切换、存储 key、页内锚点与 skip-link 保持不变。
- 控制台只有 Vite 开发连接日志，无 warning / error。

## 工程验证

- `pnpm test`：3/3 通过。
- `pnpm typecheck`：通过。
- `pnpm build`：通过；Chakra Petch、JetBrains Mono、Miura 主图与金箔纹理均进入产物。
- `dist/index.html` 保留方向契约 seed key `db6c7cf2`。
- Impeccable 机械检测器按流程仅运行一次；只报告旧 `DESIGN.md` 与新实现之间的字体、字号和颜色漂移，没有结构性反模式。最终 documenter 已在评审通过后重建文档。

## 独立评审

- 首轮：`disposition: fix`，指出显示字体、金箔材质、移动端主图/图例、首屏下一段预告、移动菜单功能清单与重复 reveal 动效六类问题。
- 修正后复判：六项全部 `resolved`，无修正回归。
- 最终结论：**`disposition: pass`**。

## 保留的功能契约

品牌名 / logo / GitHub 链接 / Docs 链接 / MIT 表述 / 中英双语 / 明暗双主题 / 页内锚点 / `data-testid` / `recruit.jkghjk123.site` / skip-link / `prefers-reduced-motion`。营销文案保持原意；中文界面品牌显示由 Smart Recruit 统一为“智联招聘”，英文界面仍使用 Smart Recruit。页脚不再重复显示品牌区块。

final result: passed · finish-review disposition: pass
