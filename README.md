# CamPron Enterprise (Vue3 + Gin) v3

目标：做一个更贴近真实企业开发的「前后端分离」小系统。

当前能力：

1. 输入单词（例如 activity），选择 US/UK/BOTH
2. 后端抓取 Cambridge 词条页面，解析：
   - 发音 mp3 地址
   - 对应口音的 IPA 音标（显示格式与 Cambridge 页面一致，例如 **/ækˈtɪv.ə.ti/**）
3. 保存到本地：在配置目录下自动创建以单词命名的文件夹，同时存放 mp3 + ipa
4. 前端展示结果，并提供可直接播放/下载的 mp3 URL
5. 前端在输入框 **按回车即可下载**（也可点击按钮）

---

## 目录结构（企业常见分层）

- `backend/`
  - `cmd/server/`：入口
  - `configs/`：配置文件
  - `internal/`
    - `httpserver/`：Gin 路由与服务器
    - `middleware/`：中间件（RequestID、日志、CORS）
    - `handler/`：接口层（参数校验 + 调用 service）
    - `service/cambridge/`：抓取/解析/下载核心逻辑
    - `util/`：通用工具
- `frontend/`
  - `src/api/`：Axios 客户端与接口封装
  - `src/App.vue`：页面（回车触发下载）

---

## 保存目录与文件结构

在 `storage.download_dir` 指定一个目录，例如：

- Windows：`D:/EnglishAudio/cambridge`
- macOS/Linux：`/Users/xxx/EnglishAudio/cambridge`

下载 `activity`（both）后会生成：

```
<download_dir>/activity/activity_uk.mp3
<download_dir>/activity/activity_uk.ipa.txt
<download_dir>/activity/activity_us.mp3
<download_dir>/activity/activity_us.ipa.txt
```

---

## 后端运行

### 1) 修改配置

编辑 `backend/configs/config.yaml`：

```yaml
server:
  addr: ":8080"
  base_url: "http://localhost:8080"
  cors_allow_origin: "*"
storage:
  download_dir: "../downloads"
```

**把 download_dir 改成你想保存的位置**（推荐绝对路径），改完重启后端。

> 如果报端口占用：把 `addr` 改成 `:8081`，并同步改 `base_url`，前端 `.env` 也要改。

### 2) 启动

```bash
cd backend
go mod tidy
go run ./cmd/server
```

### 3) 验证

- 健康检查：`GET http://localhost:8080/api/v1/health`
- 静态文件：`GET http://localhost:8080/files/<word>/<filename>`

---

## 前端运行（Vue3 + Vite）

### 1) 配置后端地址

编辑 `frontend/.env.development`：

```env
VITE_API_BASE=http://localhost:8080
```

### 2) 启动

```bash
cd frontend
npm i
npm run dev
```

打开提示的地址（一般 `http://localhost:5173`）。

使用方式：

- 在输入框输入单词
- **按回车** 或点击按钮即可下载

### 整体启动

```bash
powershell -ExecutionPolicy Bypass -File .\script\dev.ps1
```

---

## API 说明

### POST /api/v1/pronunciations/download

请求：

```json
{ "word": "activity", "accent": "both" }
```

响应（字段节选）：

```json
{
  "ok": true,
  "page_url": "https://dictionary.cambridge.org/dictionary/english/activity",
  "saved": [
    {
      "accent": "uk",
      "folder": "D:/EnglishAudio/cambridge/activity",
      "mp3_url": "http://localhost:8080/files/activity/activity_uk.mp3",
      "ipa": "/ækˈtɪv.ə.ti/"
    }
  ]
}
```

---

## 备注（关于 IPA 解析）

IPA 解析属于“页面结构依赖”的抓取行为：

- 当前实现优先根据 `uk/us` 的 region 标记，在其附近窗口内取第一个 IPA span（通常就是页面红框展示的那个）
- 若 Cambridge 改版可能导致 ipa 为空，但 mp3 仍可能正常下载（前端会显示 `-`）

如要进一步增强鲁棒性，可以改成：

- 基于 HTML 解析器（goquery）而不是正则
- 按更精确的 DOM 结构定位 UK/US 模块
