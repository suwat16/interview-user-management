# ทดสอบ API ด้วย cURL

แต่ละไฟล์ `.txt` มีคำสั่ง cURL หนึ่งคำสั่ง เปิด API และ MongoDB ก่อนส่ง request แล้วแก้ URL ในคำสั่งหาก API ใช้พอร์ตอื่นจาก `8080`

ดู HTTP status และ response body ที่บันทึกจากการรันจริงได้ใน [ผลทดสอบ API ด้วย cURL](test-results.md)

- **Terminal** คัดลอกคำสั่งไปวางแล้วรันได้เลย เพิ่ม `-i` หลัง `curl` หากต้องการดู HTTP status และ headers
- **Postman** เปิด **Import** แล้ววางคำสั่งจากไฟล์ [คู่มือ Postman](https://learning.postman.com/docs/getting-started/importing-and-exporting/importing-data#import-and-convert-curl-commands)
- **Bruno** เลือก **New Request → From cURL** แล้ววางคำสั่งจากไฟล์ [คู่มือ Bruno](https://blog.usebruno.com/curl-to-gui-import-curl-commands)

| Method | Endpoint | ไฟล์ | HTTP ที่คาดหวัง |
| --- | --- | --- | --- |
| GET | `/livez` | [get-livez.txt](get-livez.txt) | 200 |
| GET | `/readyz` | [get-readyz.txt](get-readyz.txt) | 200 |
| POST | `/users` | [post-users.txt](post-users.txt) | 201 |
| POST | `/auth/login` | [post-auth-login.txt](post-auth-login.txt) | 200 |
| GET | `/users` | [get-users.txt](get-users.txt) | 200 |
| GET | `/users/:id` | [get-user.txt](get-user.txt) | 200 |
| PATCH | `/users/:id` | [patch-user.txt](patch-user.txt) | 200 |
| DELETE | `/users/:id` | [delete-user.txt](delete-user.txt) | 204 |

1. สมัครบัญชีทดสอบผ่าน `post-users.txt` ด้วยอีเมลที่ยังไม่เคยใช้ แล้วนำ `id` จากผลสมัครมาแทน `REPLACE_USER_ID`
2. ล็อกอินผ่าน `post-auth-login.txt` ด้วยอีเมลและรหัสผ่านเดียวกัน แล้วนำ `access_token` มาแทน `REPLACE_ACCESS_TOKEN` ในคำขอที่มี header `Authorization`
3. ลองอ่านและแก้ไขบัญชีทดสอบตามต้องการ เมื่อลบแล้วให้เรียก GET ด้วย ID เดิมเพื่อเช็กว่าได้ `404`
