# User Management API

REST API สำหรับจัดการผู้ใช้ด้วย Go, Fiber และ MongoDB รองรับการสมัคร ล็อกอินด้วย JWT และการอ่าน แก้ไข และลบข้อมูลผู้ใช้

## ตั้งค่าและรันบนเครื่อง

ต้องใช้ Go 1.26 ขึ้นไป, Docker พร้อม Compose, OpenSSL และ cURL ให้รันคำสั่งจาก root ของ repo

1. สร้าง `.env` จากไฟล์ตัวอย่างเมื่อเริ่มตั้งค่าครั้งแรก แล้วสร้าง signing secret

   ```sh
   cp .env.example .env
   openssl rand -hex 32
   ```

   นำค่าที่ได้ไปใส่ใน `JWT_SECRET` ใน `.env` แอปต้องการ secret อย่างน้อย 32 ไบต์ และ Git จะไม่ติดตามไฟล์ `.env` ดูค่าอื่น ๆ ได้ใน [`.env.example`](.env.example)

2. เปิด MongoDB แล้วรัน API

   ```sh
   docker compose up -d --wait mongo
   go run .
   ```

   API ฟังที่ `127.0.0.1:8080` และเชื่อมต่อ MongoDB ที่ `127.0.0.1:27017` ตามค่าตัวอย่าง แอปสร้าง unique index บน `email` ตอนเริ่มทำงาน

3. เปิด terminal อีกหน้าต่างเพื่อเช็กสถานะ ทั้งสองคำสั่งควรได้ `200 OK`

   ```sh
   curl -i http://127.0.0.1:8080/livez
   curl -i http://127.0.0.1:8080/readyz
   ```

กด `Ctrl+C` ในหน้าต่างที่รัน API เพื่อหยุดแอป จากนั้นใช้ `docker compose stop mongo` เพื่อหยุด MongoDB โดยเก็บข้อมูลใน volume ไว้

## สมัครและใช้งาน JWT

1. สมัครด้วย `POST /users` ตัวอย่างนี้ใช้บัญชีทดสอบ หากเคยใช้ email นี้แล้ว ให้เปลี่ยน email ในทั้งสองคำสั่ง

   ```sh
   curl -i http://127.0.0.1:8080/users \
     -H 'Content-Type: application/json' \
     -d '{"name":"Manual Test","email":"manual-test@example.com","password":"ManualTest123!"}'
   ```

   เมื่อได้ `201 Created` ให้นำ `id` จาก response ไปใช้กับ endpoint `/users/:id`

2. ล็อกอินด้วย email และ password เดียวกัน API จะออก JWT ที่เซ็นด้วย `HS256` แล้วส่งกลับใน `access_token`

   ```sh
   curl -i http://127.0.0.1:8080/auth/login \
     -H 'Content-Type: application/json' \
     -d '{"email":"manual-test@example.com","password":"ManualTest123!"}'
   ```

3. แทน `REPLACE_ACCESS_TOKEN` ด้วยค่า `access_token` จาก response แล้วส่งผ่าน header `Authorization`

   ```sh
   curl -i http://127.0.0.1:8080/users \
     -H 'Authorization: Bearer REPLACE_ACCESS_TOKEN'
   ```

   Token มีอายุ 1 ชั่วโมงตามค่าเริ่มต้นของ `JWT_ACCESS_TOKEN_LIFETIME` ถ้า token หมดอายุ ให้ล็อกอินใหม่เพื่อรับ token อันใหม่ คำขอที่ขาด token หรือใช้ token ที่ตรวจสอบไม่ผ่านจะได้ `401 Unauthorized`

## Endpoints

| Method | Path | ต้องใช้ JWT | สถานะเมื่อสำเร็จ |
| --- | --- | --- | --- |
| POST | `/users` | ไม่ | `201 Created` |
| POST | `/auth/login` | ไม่ | `200 OK` |
| GET | `/users` | ใช่ | `200 OK` |
| GET | `/users/:id` | ใช่ | `200 OK` |
| PATCH | `/users/:id` | ใช่ | `200 OK` |
| DELETE | `/users/:id` | ใช่ | `204 No Content` |
| GET | `/livez` | ไม่ | `200 OK` |
| GET | `/readyz` | ไม่ | `200 OK` |

`PATCH /users/:id` รับ `name`, `email` หรือทั้งสองฟิลด์ เมื่อใช้ email ที่มีอยู่แล้วกับการสมัครหรือแก้ไข API จะตอบ `409 Conflict` และเมื่ออ่าน แก้ไข หรือลบด้วย ID ที่ไม่มีอยู่จะได้ `404 Not Found`

ดู [คำสั่ง cURL ครบทุก endpoint](scripts/curl/README.md) และ [ตัวอย่าง status และ response จากการรันกับ MongoDB](scripts/curl/test-results.md)

## ทดสอบ

```sh
go test ./...
go test -race ./...
```

ชุดเทสต์ใช้ Go `testing` และ mock การเข้าถึง MongoDB จึงรันได้โดยไม่ต้องเปิดฐานข้อมูล ส่วนการลอง API ตามคู่มือ cURL ต้องเปิดทั้งแอปและ MongoDB

## ข้อตกลงและการออกแบบ

- **สิทธิ์ของผู้ใช้** เปิดให้สมัครและล็อกอินได้โดยตรง ผู้ถือ JWT ที่ผ่านการตรวจสอบสามารถดูรายการ อ่าน แก้ไข และลบบัญชีตาม ID ได้ทุกบัญชีตามขอบเขต API นี้ Token ที่ออกให้แล้วยังใช้ได้จนหมดอายุแม้บัญชีต้นทางจะถูกลบ
- **ข้อมูลผู้ใช้** สร้าง ID จาก MongoDB ObjectID และเก็บในรูป hex string เก็บ `created_at` เป็นเวลา UTC รูปแบบ RFC 3339 ตัดช่องว่างหัวท้าย email และตรวจ email ซ้ำตามตัวพิมพ์ด้วย unique index รหัสผ่านเก็บเป็น bcrypt hash และไม่รวมใน JSON response
- **การแยกส่วน** HTTP handler เรียก service ส่วน service ใช้ repository interface และ MongoDB เป็นตัวเก็บข้อมูล ทำให้ทดสอบ business logic ด้วย mock ได้
- **การทำงานเบื้องหลัง** Middleware บันทึก HTTP method, path และเวลาที่ใช้ ส่วน goroutine นับผู้ใช้จาก MongoDB และบันทึก `total_users` ทุก 10 วินาทีตามค่าเริ่มต้น เมื่อรับสัญญาณหยุดแอปจะยกเลิก worker และปิดการเชื่อมต่อ MongoDB
