#!/bin/bash
# ============================================================
# count_lines.sh — 统计 smart-recruit 项目代码行数
#
# 用法:
#   ./scripts/count_lines.sh           统计所有代码
#   ./scripts/count_lines.sh --detail  按文件展示明细
#   ./scripts/count_lines.sh --help    显示帮助
# ============================================================

set -euo pipefail

# 自动定位项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

DETAIL=false

for arg in "$@"; do
  case "$arg" in
    --detail|-d) DETAIL=true ;;
    --help|-h)
      echo "用法: $0 [--detail]"
      echo ""
      echo "  无参数        按语言分类汇总统计"
      echo "  --detail, -d  同时展示每个文件的明细行数"
      exit 0
      ;;
  esac
done

# ============================================================
# 排除路径：使用 */xxx/* 匹配任意深度的目录
# ============================================================

# 基础排除（所有语言都排除）
BASE_EXCLUDES=(
  '*.git/*'
  '*/node_modules/*'
  '*/.pnpm-store/*'
  '*/dist/*'
  '*/docs/*'
  '*/.claude/*'
  '*/.codegraph/*'
  '*/.ai-guides/*'
  '*/tools/*'
  '*/.reasonix/*'
  '*/.dev/*'
)

# 特定排除
EXCLUDE_PB='*/recruitment/pb/*'    # protobuf 自动生成代码
EXCLUDE_LOCK_YAML='*/pnpm-lock.yaml'  # 依赖锁文件
EXCLUDE_LOCK_YML='*.yml'              # 默认排除所有 .yml，下面单独加回 CI
EXCLUDE_IMAGES=('*.png' '*.webp' '*.jpg' '*.svg' '*.ico')
EXCLUDE_SECRETS=('*/docker/.env' '*/deploy/k8s/secret.yaml')
EXCLUDE_ENV=('.env' '.env.*')

# ============================================================
# 辅助函数
# ============================================================

# 将排除规则数组转换为 find 的 -not -path 参数
build_exclude_args() {
  local args=()
  for pattern in "$@"; do
    [[ -n "$pattern" ]] && args+=(-not -path "$pattern")
  done
  echo "${args[@]}"
}

# 统计符合 pattern 的文件列表
find_files() {
  local pattern="$1"
  shift
  # 通过函数参数传递排除规则数组
  local exclude_paths=()
  for arg in "$@"; do
    exclude_paths+=(-not -path "$arg")
  done
  find . -type f -name "$pattern" "${exclude_paths[@]}" 2>/dev/null | sort
}

# 计算文件的总行数
count_total() {
  local files="$1"
  if [ -z "$files" ]; then echo 0; return; fi
  echo "$files" | xargs cat 2>/dev/null | wc -l | tr -d ' '
}

# 计算文件的非空行数
count_effective() {
  local files="$1"
  if [ -z "$files" ]; then echo 0; return; fi
  echo "$files" | xargs cat 2>/dev/null | grep -cv '^\s*$' || echo 0
}

# 统计一组文件
count_group() {
  local desc="$1"
  local pattern="$2"
  shift 2

  local files
  files=$(find_files "$pattern" "$@")

  local file_count
  file_count=$(echo "$files" | grep -c '^' 2>/dev/null || echo 0)

  if [ "$file_count" -eq 0 ] || [ -z "$files" ]; then
    return
  fi

  local total effective
  total=$(count_total "$files")
  effective=$(count_effective "$files")

  printf "  %-12s  %4s 个文件  %6s 总行  %6s 有效行\n" \
    "$desc" "$file_count" "$total" "$effective"

  # 明细模式
  if $DETAIL; then
    local IFS=$'\n'
    for f in $files; do
      local ft
      ft=$(wc -l < "$f" 2>/dev/null | tr -d ' ')
      printf "    %-60s %5s 行\n" "$f" "$ft"
    done
  fi
}

# ============================================================
# 输出
# ============================================================

echo ""
echo "  smart-recruit 项目代码统计"
echo "  =========================================="
echo ""

# ============================================================
# 后端
# ============================================================
echo "  [后端]"

count_group "Go" "*.go" \
  "${BASE_EXCLUDES[@]}" "$EXCLUDE_PB"

echo ""

# ============================================================
# 前端
# ============================================================
echo "  [前端]"

count_group "Vue 3" "*.vue" \
  "${BASE_EXCLUDES[@]}"

count_group "TypeScript" "*.ts" \
  "${BASE_EXCLUDES[@]}"

count_group "CSS" "*.css" \
  "${BASE_EXCLUDES[@]}"

count_group "HTML" "*.html" \
  "${BASE_EXCLUDES[@]}"

echo ""

# ============================================================
# 数据库
# ============================================================
echo "  [数据库]"

count_group "SQL" "*.sql" \
  "${BASE_EXCLUDES[@]}"

echo ""

# ============================================================
# 接口定义
# ============================================================
echo "  [接口定义]"

# Proto 文件以 smart-recruit-proto 为唯一契约源
# 两者都统计，因为属于不同服务
count_group "Protobuf" "*.proto" \
  "${BASE_EXCLUDES[@]}"

echo ""

# ============================================================
# 部署 & 运维
# ============================================================
echo "  [部署 & 运维]"

count_group "Dockerfile" "*.Dockerfile" \
  "${BASE_EXCLUDES[@]}"

count_group "YAML" "*.yaml" \
  "${BASE_EXCLUDES[@]}" "$EXCLUDE_LOCK_YAML" \
  "${EXCLUDE_SECRETS[@]}"

# .yml: 只包含 CI 配置（.github/），排除 docs 下的文档 yml
count_group "YML" "*.yml" \
  "${BASE_EXCLUDES[@]}" \
  '*/docs/*'

count_group "Nginx" "*.conf" \
  "${BASE_EXCLUDES[@]}"

count_group "Shell" "*.sh" \
  "${BASE_EXCLUDES[@]}"

echo ""

# ============================================================
# 汇总
# ============================================================
echo "  =========================================="
printf "  有效行 = 总行 - 空白行\n"
echo ""
