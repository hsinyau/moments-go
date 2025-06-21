#!/bin/bash

echo "🔍 测试网络连接到 Telegram API..."

# 测试基本连接
echo "1. 测试基本 HTTPS 连接..."
curl -I --connect-timeout 10 https://api.telegram.org

if [ $? -eq 0 ]; then
    echo "✅ 基本连接正常"
else
    echo "❌ 基本连接失败"
fi

# 测试 DNS 解析
echo -e "\n2. 测试 DNS 解析..."
nslookup api.telegram.org

# 测试延迟
echo -e "\n3. 测试网络延迟..."
ping -c 3 api.telegram.org

# 测试 TLS 连接
echo -e "\n4. 测试 TLS 连接..."
openssl s_client -connect api.telegram.org:443 -servername api.telegram.org < /dev/null 2>/dev/null | grep "Verify return code"

echo -e "\n5. 检查系统时间..."
date

echo -e "\n6. 检查网络配置..."
echo "当前网络接口:"
ip route show default

echo -e "\n如果以上测试都正常，TLS 错误可能是临时的网络问题。"
echo "建议："
echo "1. 等待几分钟后重试"
echo "2. 检查网络代理设置"
echo "3. 尝试重启网络连接" 
