# ผลทดสอบ API ด้วย cURL

ทดสอบวันที่ 2026-11-19 เวลา 13:10 น. (เวลาไทย) โดยเปิด API ที่ `http://127.0.0.1:8080` จากโปรแกรมที่ build ด้วย `go build main.go` และใช้ MongoDB 8.0 ชั่วคราว

ตัวอย่างด้านล่างมาจากการทดสอบจริง จัดรูปแบบ JSON ให้อ่านง่ายและแทนค่า JWT ด้วย `REDACTED`

## GET /livez — ตรวจว่า API ทำงาน

คำสั่งจาก [`get-livez.txt`](get-livez.txt)

```text
HTTP/1.1 200 OK

OK
```

## GET /readyz — ตรวจความพร้อมของ MongoDB

คำสั่งจาก [`get-readyz.txt`](get-readyz.txt)

```text
HTTP/1.1 200 OK

OK
```

## GET /users — ไม่ส่ง Authorization

ใช้ [`get-users.txt`](get-users.txt) โดยนำ header `Authorization` ออก

```text
HTTP/1.1 401 Unauthorized

{
  "error": "Unauthorized"
}
```

## POST /users — ส่ง body เป็น {}

ใช้ [`post-users.txt`](post-users.txt) โดยเปลี่ยน JSON body เป็น `{}`

```text
HTTP/1.1 400 Bad Request

{
  "error": "name, valid email, and password are required"
}
```

## POST /users — สมัครบัญชีทดสอบ

คำสั่งจาก [`post-users.txt`](post-users.txt)

```text
HTTP/1.1 201 Created

{
  "id": "6afe92e8d986d35b06dacfe9",
  "name": "Manual Test",
  "email": "manual-test@example.com",
  "created_at": "2026-11-19T06:10:48Z"
}
```

## POST /users — สมัครซ้ำด้วยอีเมลเดิม

คำสั่งจาก [`post-users.txt`](post-users.txt)

```text
HTTP/1.1 409 Conflict

{
  "error": "email already taken"
}
```

## POST /auth/login — ใช้รหัสผ่านผิด

ใช้ [`post-auth-login.txt`](post-auth-login.txt) โดยเปลี่ยน `password` เป็น `wrong-password`

```text
HTTP/1.1 401 Unauthorized

{
  "error": "invalid email or password"
}
```

## POST /auth/login — ล็อกอินสำเร็จ

คำสั่งจาก [`post-auth-login.txt`](post-auth-login.txt)

```text
HTTP/1.1 200 OK

{
  "access_token": "REDACTED"
}
```

## GET /users — ดูรายการผู้ใช้

คำสั่งจาก [`get-users.txt`](get-users.txt)

```text
HTTP/1.1 200 OK

[
  {
    "id": "6afe92e8d986d35b06dacfe9",
    "name": "Manual Test",
    "email": "manual-test@example.com",
    "created_at": "2026-11-19T06:10:48Z"
  }
]
```

## GET /users/:id — อ่านข้อมูลผู้ใช้

คำสั่งจาก [`get-user.txt`](get-user.txt)

```text
HTTP/1.1 200 OK

{
  "id": "6afe92e8d986d35b06dacfe9",
  "name": "Manual Test",
  "email": "manual-test@example.com",
  "created_at": "2026-11-19T06:10:48Z"
}
```

## PATCH /users/:id — เปลี่ยนชื่อผู้ใช้

คำสั่งจาก [`patch-user.txt`](patch-user.txt)

```text
HTTP/1.1 200 OK

{
  "id": "6afe92e8d986d35b06dacfe9",
  "name": "Manual Test Updated",
  "email": "manual-test@example.com",
  "created_at": "2026-11-19T06:10:48Z"
}
```

## GET /users/:id — อ่านชื่อหลังแก้ไข

คำสั่งจาก [`get-user.txt`](get-user.txt)

```text
HTTP/1.1 200 OK

{
  "id": "6afe92e8d986d35b06dacfe9",
  "name": "Manual Test Updated",
  "email": "manual-test@example.com",
  "created_at": "2026-11-19T06:10:48Z"
}
```

## DELETE /users/:id — ลบบัญชีทดสอบ

คำสั่งจาก [`delete-user.txt`](delete-user.txt)

```text
HTTP/1.1 204 No Content
```

## GET /users/:id — อ่านหลังลบ

คำสั่งจาก [`get-user.txt`](get-user.txt)

```text
HTTP/1.1 404 Not Found

{
  "error": "user not found"
}
```

**สรุป** ผ่าน 14/14 กรณี ครบทั้ง 8 endpoints
