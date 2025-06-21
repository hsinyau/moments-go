#!/bin/bash

echo "🌐 网络连接诊断工具"
echo "=================="

# 检查网络连接
echo "1. 检查网络连接..."
if ping -c 1 8.8.8.8 > /dev/null 2>&1; then
    echo "✅ 网络连接正常"
else
    echo "❌ 网络连接失败"
    echo "   请检查网络设置或联系网络管理员"
    exit 1
fi

# 检查 DNS 解析
echo -e "\n2. 检查 DNS 解析..."
if nslookup api.telegram.org > /dev/null 2>&1; then
    echo "✅ DNS 解析正常"
else
    echo "❌ DNS 解析失败"
    echo "   尝试使用公共 DNS："
    echo "   echo 'nameserver 8.8.8.8' | sudo tee /etc/resolv.conf"
    echo "   echo 'nameserver 1.1.1.1' | sudo tee -a /etc/resolv.conf"
fi

# 检查 HTTPS 连接
echo -e "\n3. 检查 HTTPS 连接..."
if curl -I --connect-timeout 10 https://api.telegram.org > /dev/null 2>&1; then
    echo "✅ HTTPS 连接正常"
else
    echo "❌ HTTPS 连接失败"
    echo "   可能是防火墙或代理问题"
fi

# 检查系统时间
echo -e "\n4. 检查系统时间..."
current_time=$(date +%s)
if [ $current_time -gt 0 ]; then
    echo "✅ 系统时间正常: $(date)"
else
    echo "❌ 系统时间异常"
    echo "   请同步系统时间：sudo ntpdate -s time.nist.gov"
fi

# 检查代理设置
echo -e "\n5. 检查代理设置..."
if [ -n "$HTTP_PROXY" ] || [ -n "$HTTPS_PROXY" ]; then
    echo "⚠️  检测到代理设置："
    echo "   HTTP_PROXY: $HTTP_PROXY"
    echo "   HTTPS_PROXY: $HTTPS_PROXY"
    echo "   如果连接有问题，请检查代理配置"
else
    echo "✅ 未检测到代理设置"
fi

# 检查防火墙
echo -e "\n6. 检查防火墙状态..."
if command -v ufw > /dev/null 2>&1; then
    ufw_status=$(sudo ufw status 2>/dev/null | grep "Status")
    echo "UFW 状态: $ufw_status"
fi

if command -v firewall-cmd > /dev/null 2>&1; then
    firewalld_status=$(sudo firewall-cmd --state 2>/dev/null)
    echo "Firewalld 状态: $firewalld_status"
fi

echo -e "\n7. 建议的解决方案："
echo "   - 如果网络连接正常但仍有问题，请等待几分钟后重试"
echo "   - 检查是否有 VPN 或代理影响连接"
echo "   - 尝试重启网络服务：sudo systemctl restart NetworkManager"
echo "   - 如果问题持续，请联系网络管理员"

echo -e "\n✅ 诊断完成！" 
