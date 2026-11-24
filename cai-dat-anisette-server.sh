#!/usr/bin/env bash
# Cài đặt NVD-Anisette-Server trên VPS Linux (Ubuntu 22.04 / 24.04).
# Nguồn: repo nvd29u1/ipatool-anisette-server, release v1.0.1 trở lên.
# Chạy bằng quyền root (hoặc sudo). Script tải binary từ GitHub Releases
# theo đúng kiến trúc máy (x86_64 hoặc aarch64), đặt systemd service,
# mở cổng 6969 qua UFW.
# LƯU Ý BẢO MẬT: server này KHÔNG có xác thực — bất kỳ ai gọi được tới
# cổng 6969 đều lấy được header Anisette của máy này. Chỉ mở cổng cho
# đúng địa chỉ IP của người dùng (xem HUONG-DAN-VPS.md), không mở 0.0.0.0/0.
set -euo pipefail

RELEASE_TAG="v1.0.1"
INSTALL_DIR="/opt/nvd-anisette-server"
DATA_DIR="/var/lib/nvd-anisette-server"
SERVICE_USER="nvd-anisette"

# --- Chọn binary theo kiến trúc máy ---
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)
    BIN_NAME="anisette-server-linux-amd64"
    BIN_SHA256="6f568117e030a7be739947e8a4e4eb35419bab90678e211937ac2af317aa497e"
    BIN_SIZE=16969993
    ;;
  aarch64|arm64)
    # Oracle Ampere A1 (Always Free). Binary ARM64 build từ cùng source,
    # kiểm chứng: ELF aarch64, engine Unicorn tải libunicorn.so.2 bản
    # manylinux aarch64 lúc chạy (đã có sẵn trong code ipatool).
    # Chưa kiểm chứng chạy thật trên máy ARM — nếu provisioning báo lỗi,
    # ghi lại log, đừng xóa adi.json vội.
    BIN_NAME="anisette-server-linux-arm64"
    BIN_SHA256="39aad0288b36bb7d43bd55d2cb9dfcb9ad6a7795d5ae626be1d2fee5a15a4da6"
    BIN_SIZE=16253193
    ;;
  *)
    echo "LỖI: kiến trúc $ARCH chưa có binary phát hành (chỉ có x86_64 và aarch64)."
    exit 1
    ;;
esac
BIN_URL="https://github.com/nvd29u1/ipatool-anisette-server/releases/download/${RELEASE_TAG}/${BIN_NAME}"

# --- Kiểm tra ổ đĩa trống (cần tối thiểu 20 GB, khuyến nghị 40 GB) ---
FREE_KB="$(df --output=avail -k / | tail -1)"
if [ "$FREE_KB" -lt 20971520 ]; then
  echo "CẢNH BÁO: ổ đĩa còn trống dưới 20 GB. Lần đầu server tải IPSW ~11 GB từ Apple."
  echo "Nên tăng ổ đĩa lên tối thiểu 40 GB trước khi tiếp tục."
  read -r -p "Vẫn tiếp tục? (y/N) " ANS
  [ "$ANS" = "y" ] || [ "$ANS" = "Y" ] || exit 1
fi

export DEBIAN_FRONTEND=noninteractive
apt-get update -y
apt-get install -y curl ca-certificates ufw

# --- Người dùng hệ thống riêng, không chạy bằng root ---
id -u "$SERVICE_USER" >/dev/null 2>&1 || useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER"
mkdir -p "$INSTALL_DIR" "$DATA_DIR"
chown "$SERVICE_USER:$SERVICE_USER" "$DATA_DIR"
chmod 700 "$DATA_DIR"

# --- Tải binary Linux, kiểm tra kích thước + SHA256 ---
cd /tmp
curl -fL -o "$BIN_NAME" "$BIN_URL"
ACTUAL_SIZE="$(stat -c%s "$BIN_NAME")"
[ "$ACTUAL_SIZE" = "$BIN_SIZE" ] || { echo "LỖI: kích thước file tải về ($ACTUAL_SIZE) khác kỳ vọng ($BIN_SIZE)."; exit 1; }
echo "$BIN_SHA256  $BIN_NAME" | sha256sum -c -
install -m 755 "$BIN_NAME" "$INSTALL_DIR/anisette-server"
rm -f "$BIN_NAME"

# --- systemd service: chỉ nghe 127.0.0.1 mặc định ---
# Muốn iPhone/app gọi từ xa thì đổi thành --listen 0.0.0.0:6969
# VÀ khóa firewall chỉ cho đúng IP (xem HUONG-DAN-VPS.md).
cat > /etc/systemd/system/nvd-anisette-server.service <<EOF
[Unit]
Description=NVD Anisette Server (ipatool Anisette engine)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
ExecStart=$INSTALL_DIR/anisette-server --listen 127.0.0.1:6969 --data-dir $DATA_DIR --generate-timeout 60m
Restart=on-failure
RestartSec=10
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=$DATA_DIR

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable nvd-anisette-server.service

# --- UFW: mặc định CHẶN cổng 6969 từ internet ---
# Mở sau, chỉ cho đúng IP, bằng lệnh:
#   ufw allow from <IP-cua-ban> to any port 6969 proto tcp
ufw --force enable || true
ufw deny 6969/tcp || true

echo "Cài đặt xong. Khởi động server:"
echo "  systemctl start nvd-anisette-server.service"
echo "Kiểm tra:"
echo "  curl http://127.0.0.1:6969/health"
echo "Provisioning lần đầu (tải ~11 GB, có thể mất 30 phút đến vài giờ):"
echo "  curl http://127.0.0.1:6969/anisette"
echo "File danh tính máy: $DATA_DIR/adi.json — KHÔNG chia sẻ cho ai, sao lưu riêng."
echo "Trên Oracle Always Free: sao lưu adi.json ra ngoài máy ngay sau khi"
echo "provisioning xong — máy free có thể bị thu hồi nếu để nhàn quá 7 ngày."
