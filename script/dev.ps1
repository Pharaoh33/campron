# 如果脚本中出现错误，则立即停止执行
$ErrorActionPreference = "Stop"

# 获取项目根目录
# $PSScriptRoot 表示当前脚本所在目录（script）
# Split-Path -Parent 获取上级目录
$root = Split-Path -Parent $PSScriptRoot

# 拼接 backend 目录路径
$backend = Join-Path $root "backend"

# 拼接 frontend 目录路径
$frontend = Join-Path $root "frontend"


# =========================
# 启动后端服务
# =========================

Write-Host "[dev] start backend..."

# 进入 backend 目录
Set-Location $backend

# 自动整理并下载 go 依赖
go mod tidy | Out-Null


# 设置后端日志文件路径
$backendOutLog = Join-Path $root ".backend.out.log"
$backendErrLog = Join-Path $root ".backend.err.log"


# 启动后端服务
# 相当于执行：
# go run ./cmd/server
#
# -PassThru       返回进程对象
# -Redirect...    把日志写入文件
$backendProc = Start-Process `
    -FilePath "go" `
    -ArgumentList "run","./cmd/server" `
    -PassThru `
    -RedirectStandardOutput $backendOutLog `
    -RedirectStandardError $backendErrLog



# =========================
# 启动前端服务
# =========================

Write-Host "[dev] start frontend..."

# 进入 frontend 目录
Set-Location $frontend

# 安装 npm 依赖
npm i

# 启动 Vue3 开发服务器
# 默认地址：
# http://localhost:5173
npm run dev



# =========================
# 当前端关闭时自动停止后端
# =========================

Write-Host "[dev] stop backend..."

# 停止刚刚启动的 Go 服务
Stop-Process -Id $backendProc.Id -Force