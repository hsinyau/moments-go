#!/bin/bash

# 测试环境变量配置脚本

echo "🔍 检查环境变量配置..."

# 检查必需的环境变量
required_vars=(
    "TELEGRAM_BOT_TOKEN"
    "TELEGRAM_USER_ID"
    "GITHUB_SECRET"
    "GITHUB_USERNAME"
)

missing_vars=()

for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        missing_vars+=("$var")
    fi
done

if [ ${#missing_vars[@]} -ne 0 ]; then
    echo "❌ 缺少必需的环境变量:"
    for var in "${missing_vars[@]}"; do
        echo "   - $var"
    done
    exit 1
fi

# 检查可选的环境变量（显示默认值）
echo "✅ 必需的环境变量已设置"
echo ""
echo "📋 当前配置:"
echo "   TELEGRAM_BOT_TOKEN: ${TELEGRAM_BOT_TOKEN:0:10}..."
echo "   TELEGRAM_USER_ID: $TELEGRAM_USER_ID"
echo "   GITHUB_SECRET: ${GITHUB_SECRET:0:10}..."
echo "   GITHUB_USERNAME: ${GITHUB_USERNAME:-未设置}"
echo "   GITHUB_REPO: ${GITHUB_REPO:-moments (默认)}"
echo "   GITHUB_FILE_REPO: ${GITHUB_FILE_REPO:-moments-files (默认)}"
echo "   GITHUB_USER_AGENT: ${GITHUB_USER_AGENT:-moments-bot/1.0 (默认)}"

echo ""
echo "🎉 配置检查完成！" 
