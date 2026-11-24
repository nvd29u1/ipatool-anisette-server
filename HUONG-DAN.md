# Hướng dẫn chạy NVD-Anisette-Server trên PC Windows

> Server này chỉ phục vụ một mình anh Dũng. Nó **không nhận, không lưu, không
> ghi log Apple ID hay mật khẩu** — chỉ trả header chứng minh "máy này là một
> thiết bị". Việc đăng nhập vẫn làm trong app NVDAppleIDDownload trên iPhone.

## Cần chuẩn bị

- PC Windows 64-bit, còn trống **ít nhất 20 GB ổ đĩa** (lần đầu tải ~11 GB
  file IPSW của Apple + file giải nén, xong sẽ gọn lại).
- Mạng ổn định. Lần đầu tải ~11 GB nên cần thời gian (tuỳ mạng, có thể
  30 phút đến vài tiếng) — cứ để máy chạy, đừng tắt giữa chừng.
- iPhone và PC dùng **chung một Wi-Fi nhà** (khi muốn iPhone gọi sang PC).

## Bước 1 — Tải file exe

1. Mở trang release của repo:
   `https://github.com/nvd29u1/ipatool-anisette-server/releases`
   (bản mới nhất có file `NVD-Anisette-Server.exe`).
2. Nếu chưa có release: vào tab **Actions** → chạy workflow
   **"Build Anisette Server"** mới nhất → mục **Artifacts** → tải
   `NVD-Anisette-Server-windows-amd64` về, giải nén lấy file exe bên trong.
3. Tạo thư mục riêng, ví dụ `D:\NVD-Anisette-Server\`, chép file
   `NVD-Anisette-Server.exe` vào đó.

## Bước 2 — Chạy lần đầu (provisioning)

1. Mở thư mục vừa tạo, **bấm đúp** `NVD-Anisette-Server.exe`.
   (Hoặc bấm chuột phải → Run as administrator nếu Windows chặn ghi file.)
2. Nếu Windows Firewall hiện bảng hỏi có cho phép truy cập mạng không → bấm
   **Allow access** (cho mạng Private).
3. Cửa sổ đen hiện lên, báo đang nghe ở `http://127.0.0.1:6969`.
   **Đừng đóng cửa sổ này** — đóng là server tắt.
4. Chờ server sẵn sàng: mở trình duyệt trên PC, vào
   `http://127.0.0.1:6969/health`
   - Thấy `{"status":"ok",...}` là server đã chạy.
   - Lúc này `"provisioned": false` là bình thường — máy chưa provisioning.

## Bước 3 — Provisioning máy (lần đầu, lâu nhất)

1. Trên trình duyệt PC, mở `http://127.0.0.1:6969/anisette`
   (hoặc chạy server với cờ `--provision-on-start` ngay từ bước 2).
2. Trang sẽ quay rất lâu — đây là lúc server tải file IPSW ~11 GB từ Apple
   rồi provisioning máy. **Cứ để yên, đừng F5 liên tục, đừng tắt cửa sổ.**
3. Khi xong, trình duyệt hiện một đoạn JSON có các dòng
   `X-Apple-I-MD`, `X-Apple-I-MD-M`, `X-Apple-I-MD-RINFO`... là thành công.
4. Kiểm tra lại `http://127.0.0.1:6969/health` — giờ phải là
   `"provisioned": true`.
5. Từ lần sau, mỗi lần gọi `/anisette` chỉ mất vài chục giây (không tải
   lại 11 GB nữa, vì file Apple đã lưu trong cache của máy).

> File danh tính máy nằm ở `%APPDATA%\NVD-Anisette-Server\adi.json`.
> **Không gửi file này cho ai, không up lên mạng.** Ai có nó có thể giả
> danh máy anh trước Apple.

## Bước 4 — Cho iPhone gọi sang PC

1. Trên PC, bấm `Win + R`, gõ `cmd`, Enter, rồi gõ `ipconfig` và Enter.
   Tìm dòng **IPv4 Address** của Wi-Fi, ví dụ `192.168.1.23`. Ghi số này lại.
2. Tắt server đang chạy (bấm `Ctrl + C` trong cửa sổ đen), rồi chạy lại bằng
   lệnh sau trong thư mục chứa exe (gõ vào thanh địa chỉ của File Explorer
   `cmd` rồi Enter để mở cửa sổ lệnh tại đúng thư mục đó):

   ```cmd
   NVD-Anisette-Server.exe --listen 0.0.0.0:6969
   ```

   > Chỉ làm bước này trên **Wi-Fi nhà**. Nghe `0.0.0.0` nghĩa là mọi máy
   > trong mạng nhà đều gọi được server — đừng bật ở quán cà phê hay mạng
   > công cộng.
3. Trên iPhone (chung Wi-Fi), mở Safari, vào
   `http://192.168.1.23:6969/health` (thay bằng IP ở bước 1).
   Thấy `{"status":"ok",...}` là iPhone đã gọi được sang PC.

## Bước 5 — Điền URL vào app NVDAppleIDDownload

1. Trong app, vào **Cài đặt → Máy chủ Anisette** (mục này sẽ có sau khi app
   được cập nhật ở giai đoạn 2 — hiện tại app chưa có chỗ điền).
2. Nhập URL: `http://192.168.1.23:6969` (IP PC ở bước 4, không có `/anisette`
   ở cuối — app tự gọi thêm).
3. Lưu lại. Từ giờ, mỗi lần đăng nhập app sẽ tự lấy header từ PC trước.

## Bước 6 — Kiểm chứng (quan trọng, làm bằng tài khoản phụ)

1. **Dùng tài khoản Apple phụ** (không phải tài khoản chính) để đăng nhập
   trong app lần đầu tiên qua server này.
2. Nếu đăng nhập được và tải được IPA → server qua được cổng `otp/1.5`
   mới của Apple. Lúc đó mới cân nhắc dùng tài khoản chính.
3. Nếu vẫn lỗi 204 / redirect / 404 như cũ → chụp màn hình lỗi trong app
   + log trong cửa sổ server gửi cho trợ lý, **đừng thử đi thử lại bằng
   tài khoản chính**.

## Tắt server / chạy lại

- Tắt: bấm `Ctrl + C` trong cửa sổ đen, hoặc đóng cửa sổ.
- Chạy lại: bấm đúp exe như bước 2 (không cần provisioning lại — `adi.json`
  đã lưu, lần sau gọi `/anisette` là có header ngay).
- Muốn server tự chạy khi mở máy: tạo shortcut của exe bỏ vào thư mục
  Startup (`Win + R` → `shell:startup`). Không bắt buộc.

## Nếu gặp lỗi

| Hiện tượng | Cách xử lý |
|---|---|
| Mở `/health` báo không kết nối được | Kiểm tra cửa sổ đen còn mở không; kiểm tra có gõ đúng `127.0.0.1:6969` không |
| `/anisette` quay mãi không xong (lần đầu) | Đang tải ~11 GB, cứ chờ. Kiểm tra ổ đĩa còn trống, mạng còn chạy |
| `/anisette` báo lỗi `anisette generation failed` | Chụp lại dòng lỗi trong cửa sổ server gửi cho trợ lý; đừng xoá `adi.json` vội |
| iPhone mở `http://<IP-PC>:6969/health` không được | Kiểm tra iPhone và PC chung Wi-Fi; kiểm tra Windows Firewall đã Allow; kiểm tra server đang chạy với `--listen 0.0.0.0:6969` (không phải mặc định `127.0.0.1`) |
| IP PC đổi sau khi khởi động lại modem | Chạy lại `ipconfig` lấy IP mới, sửa URL trong app |

## Rủi ro cần biết

- Server dùng engine **mới** của ipatool 2.8.0+ nên đây là hướng tốt nhất
  đã biết để qua cổng `otp/1.5` — nhưng **chưa ai kiểm chứng** header từ
  server này đăng nhập được thật qua app iOS. Khả năng vẫn lỗi là có thật.
- Nếu server này không qua được, phương án dự phòng là server kiểu cũ
  Dadoum/Provision (ADI thế hệ Sierra) — nhưng bản thân nó đã được ghi
  nhận là **không ăn được** cổng `otp/1.5`, nên chỉ thử khi đã hết cách.
- Đường chắc chắn nhất lúc này vẫn là **ipatool 2.8.1 trên PC**: đăng nhập
  trên PC, tải IPA về, chuyển sang iPhone cài qua SideStore/NVDsign.
