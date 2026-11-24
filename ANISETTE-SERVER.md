# NVD-Anisette-Server

Máy chủ Anisette chạy trên PC của riêng anh Dũng, để app **NVDAppleIDDownload**
trên iPhone lấy header xác minh thiết bị của Apple khi đăng nhập App Store.
Fork từ [majd/ipatool](https://github.com/majd/ipatool) v2.8.1 (MIT) — chỉ thêm
một HTTP wrapper nhỏ (`cmd/anisette-server/`) bọc quanh engine Anisette local
của ipatool, không sửa engine.

Chi tiết vì sao cần server này (và vì sao không port engine sang app iOS):
xem `docs/ANISETTE-RESEARCH.md` trong repo NVDAppleIDDownload.

## Vì sao dùng engine ipatool 2.8.0+ mà không dùng Dadoum/Provision

Từ ~06/11/2026 Apple yêu cầu đăng nhập App Store phải kèm bằng chứng xác minh
thiết bị (Anisette/ADI) qua cổng provisioning `otp/1.5` mới. Server Anisette
kiểu cũ (Dadoum/Provision, dùng ADI thế hệ Sierra) **không ăn được** cổng này
— chính code ipatool ghi chú: *"Apple Store otp/1.5 provisioning requires
this generation; the Sierra implementation cannot consume its settingInfo
payload"*. ipatool 2.8.0 (commit `001d889`) là bản sửa đã chứng minh được trên
PC: tải IPSW macOS 27.0.1, chạy 4 máy ảo Unicorn (commerce/adid/fairplayd/
FairPlay kernel), provisioning 2 bước tới `gsa.apple.com` rồi `RequestOTP`.
Server này bọc đúng engine đó qua HTTP.

> **Chưa kiểm chứng đầu-cuối:** chưa có bằng chứng server này qua được cổng
> `otp/1.5` khi app thật dùng header của nó để đăng nhập — cần test bằng
> **tài khoản Apple phụ** trên iPhone thật (xem `HUONG-DAN.md` bước 6).
> Tuyệt đối không dùng tài khoản/mật khẩu chính để test lần đầu.

## Cách hoạt động

1. Lần đầu, engine tải file IPSW macOS 27.0.1 (~11 GB) từ
   `updates.cdn-apple.com`, giải nén lấy 7 binary Apple (đã ghim SHA256),
   chạy giả lập để provisioning máy với `gsa.apple.com`, lưu danh tính máy
   vào `adi.json` trong thư mục dữ liệu. **Các binary của Apple không được
   đóng gói trong exe và không được phân phối lại** — chúng được tải từ CDN
   của chính Apple lúc chạy, giống hệt ipatool gốc.
2. Mỗi lần app gọi `GET /anisette`, server chạy lại provisioning + `RequestOTP`
   và trả về header mới tinh (không tái dùng OTP cũ).

## Endpoint

| Endpoint | Mô tả |
|---|---|
| `GET /` | Giống `/anisette` |
| `GET /anisette` | Header Anisette mới, dạng JSON |
| `GET /health` | `{"status":"ok",...}` — không provisioning, dùng để kiểm tra server sống hay chưa |

Ví dụ `GET /anisette` (giá trị dưới đây là ví dụ minh hoạ, không phải header thật):

```json
{
  "X-Apple-I-Client-Time": "2026-11-24T09:00:00Z",
  "X-Apple-I-MD": "<OTP base64>",
  "X-Apple-I-MD-M": "<machine ID base64>",
  "X-Apple-I-MD-RINFO": "<routing info, số>",
  "X-Apple-I-MD-LU": "<local user ID hex>",
  "X-Apple-I-SRL-NO": "0",
  "X-Apple-I-TimeZone": "Local",
  "X-Apple-Locale": "en_US",
  "X-Mme-Device-Id": "<UUID dạng xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx>",
  "X-MMe-Client-Info": "<MacBookPro13,3> <Mac OS X;10.12.6;16G29> <com.apple.AuthKit/1 (com.apple.akd/1.0)>"
}
```

Ví dụ `GET /health`:

```json
{"status": "ok", "version": "dev", "provisioned": false, "provisioning": true}
```

## Chạy

```powershell
# Mặc định chỉ nghe trên máy mình:
.\NVD-Anisette-Server.exe

# Cho iPhone trong cùng Wi-Fi truy cập (chỉ làm trên mạng nhà tin cậy):
.\NVD-Anisette-Server.exe --listen 0.0.0.0:6969

# Provision ngay khi khởi động thay vì chờ request đầu tiên:
.\NVD-Anisette-Server.exe --provision-on-start
```

Hướng dẫn từng bước bằng tiếng Việt: xem [`HUONG-DAN.md`](HUONG-DAN.md).

## Bảo mật — đọc trước khi chạy

- **Server này không bao giờ nhận, lưu hay ghi log Apple ID/mật khẩu.**
  Nó chỉ trả header chứng minh "máy này là một thiết bị". Việc đăng nhập
  vẫn diễn ra trong app, bằng tài khoản của chính anh.
- Mặc định chỉ listen `127.0.0.1:6969`. Mở ra LAN (`0.0.0.0`) thì bất kỳ ai
  trong mạng đó cũng lấy được header của máy anh — chỉ làm trên mạng nhà.
- File `adi.json` (trong `%APPDATA%\NVD-Anisette-Server\` trên Windows) là
  danh tính thiết bị của máy anh. **Không chia sẻ, không gửi cho ai.**
  Ai có file này có thể giả danh máy anh trước Apple.
- Log của server chỉ ghi method + đường dẫn + mã trạng thái + thời gian,
  không ghi header hay body.

## Build

Build chính qua GitHub Actions (`.github/workflows/build-anisette-server.yml`,
Go 1.26.5): mỗi lần push lên `main` sẽ build `NVD-Anisette-Server.exe`
(Windows amd64) + binary Linux, lưu trong tab Actions → Artifacts.
Release (tag `v*`) đính kèm exe vào release.

Build tay (cần Go 1.26+):

```sh
go vet ./cmd/anisette-server/
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o NVD-Anisette-Server.exe ./cmd/anisette-server
```

Docker (không bắt buộc, chưa build tự động trên CI):

```sh
docker build -t nvd-anisette-server .
docker run -d --name anisette -p 6969:6969 -v anisette-data:/data nvd-anisette-server
```

## Giấy phép

MIT — như ipatool gốc (majd/ipatool). Giữ nguyên LICENSE của upstream.
Phần wrapper `cmd/anisette-server/` và tài liệu tiếng Việt do nvd29u1 viết
thêm, cũng theo MIT.
