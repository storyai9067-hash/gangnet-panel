package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ===== مدل‌های داده =====
type User struct {
	ID       string    `json:"id"`
	UUID     string    `json:"uuid"`
	Traffic  int64     `json:"traffic"`
	Expiry   time.Time `json:"expiry"`
	IsActive bool      `json:"isActive"`
	Email    string    `json:"email"`
	Device   int       `json:"device"`
	Note     string    `json:"note"`
}

type Admin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Settings struct {
	Domain    string `json:"domain"`
	PanelName string `json:"panelName"`
	Theme     string `json:"theme"`
}

type Stats struct {
	TotalUsers   int `json:"totalUsers"`
	ActiveUsers  int `json:"activeUsers"`
	TotalTraffic int64 `json:"totalTraffic"`
}

// ===== ذخیره‌سازی در حافظه =====
var users []User
var settings = Settings{
	Domain:    "gngnet.ir",
	PanelName: "گنگ نت پنل",
	Theme:     "dark",
}
var admin = Admin{Username: "admin", Password: "GngNet@2025"}
var stats = Stats{}
var mu sync.Mutex

func main() {
	// ===== مسیرهای اصلی =====
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/", authMiddleware(indexHandler))
	http.HandleFunc("/admin", authMiddleware(adminHandler))
	http.HandleFunc("/settings", authMiddleware(settingsHandler))
	http.HandleFunc("/api/users", authMiddleware(apiUsersHandler))
	http.HandleFunc("/api/stats", authMiddleware(apiStatsHandler))
	http.HandleFunc("/update-domain", authMiddleware(updateDomainHandler))
	http.HandleFunc("/create-user", authMiddleware(createUserHandler))
	http.HandleFunc("/sub/", authMiddleware(subHandler))
	http.HandleFunc("/logout", logoutHandler)

	fmt.Printf("🚀 %s is running on :8080\n", settings.PanelName)
	fmt.Println("👑 Login: admin / GngNet@2025")
	http.ListenAndServe(":8080", nil)
}

// ===== صفحه ورود =====
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")
		if username == admin.Username && password == admin.Password {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "authenticated", Path: "/", MaxAge: 86400})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}
	tmpl := `<!DOCTYPE html>
	<html>
	<head>
		<title>ورود به گنگ نت پنل</title>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<style>
			* { margin: 0; padding: 0; box-sizing: border-box; font-family: 'Segoe UI', Tahoma, sans-serif; }
			body { background: linear-gradient(135deg, #0f0c29, #302b63, #24243e); min-height: 100vh; display: flex; justify-content: center; align-items: center; }
			.login-box { background: rgba(255,255,255,0.05); backdrop-filter: blur(10px); padding: 50px; border-radius: 20px; border: 1px solid rgba(255,255,255,0.1); width: 400px; box-shadow: 0 25px 50px rgba(0,0,0,0.5); }
			h1 { color: #fff; text-align: center; font-size: 28px; margin-bottom: 10px; letter-spacing: 2px; }
			.subtitle { color: #888; text-align: center; margin-bottom: 30px; font-size: 14px; }
			.input-group { margin-bottom: 20px; }
			.input-group label { color: #aaa; font-size: 14px; display: block; margin-bottom: 5px; }
			.input-group input { width: 100%; padding: 12px 15px; background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.1); border-radius: 10px; color: #fff; font-size: 16px; transition: 0.3s; }
			.input-group input:focus { outline: none; border-color: #6c63ff; box-shadow: 0 0 20px rgba(108,99,255,0.2); }
			button { width: 100%; padding: 14px; background: linear-gradient(135deg, #6c63ff, #5a52d5); border: none; border-radius: 10px; color: #fff; font-size: 18px; font-weight: bold; cursor: pointer; transition: 0.3s; margin-top: 10px; }
			button:hover { transform: scale(1.02); box-shadow: 0 5px 25px rgba(108,99,255,0.4); }
			.error { color: #ff4757; text-align: center; margin-top: 15px; font-size: 14px; }
			.footer { text-align: center; margin-top: 20px; color: #555; font-size: 12px; }
			.version { color: #333; text-align: center; font-size: 12px; margin-top: 10px; }
		</style>
	</head>
	<body>
	<div class="login-box">
		<h1>🔐 گنگ نت پنل</h1>
		<div class="subtitle">GangNet Panel v2.0</div>
		<form method="POST">
			<div class="input-group">
				<label>👤 نام کاربری</label>
				<input name="username" placeholder="admin" required>
			</div>
			<div class="input-group">
				<label>🔑 رمز عبور</label>
				<input name="password" type="password" placeholder="••••••••" required>
			</div>
			<button type="submit">🚀 ورود به پنل</button>
		</form>
		{{if .}}<div class="error">❌ نام کاربری یا رمز عبور اشتباه است!</div>{{end}}
		<div class="footer">© 2025 گنگ نت - تمامی حقوق محفوظ است</div>
		<div class="version">v2.0.0</div>
	</div>
	</body>
	</html>`
	t := template.Must(template.New("login").Parse(tmpl))
	t.Execute(w, r.URL.Query().Get("error") == "1")
}

// ===== میان‌افزار احراز هویت =====
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil || cookie.Value != "authenticated" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// ===== صفحه اصلی (داشبورد) =====
func indexHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	stats.TotalUsers = len(users)
	active := 0
	var totalTraffic int64
	for _, u := range users {
		if u.IsActive {
			active++
		}
		totalTraffic += u.Traffic
	}
	stats.ActiveUsers = active
	stats.TotalTraffic = totalTraffic
	mu.Unlock()

	tmpl := `<!DOCTYPE html>
	<html>
	<head>
		<title>گنگ نت پنل - داشبورد</title>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
		<style>
			* { margin: 0; padding: 0; box-sizing: border-box; font-family: 'Segoe UI', Tahoma, sans-serif; }
			body { background: #0f0c29; color: #fff; }
			.navbar { background: rgba(255,255,255,0.05); padding: 15px 30px; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid rgba(255,255,255,0.05); }
			.navbar h1 { color: #6c63ff; font-size: 24px; }
			.navbar h1 span { color: #fff; }
			.nav-links a { color: #aaa; text-decoration: none; margin-left: 25px; padding: 8px 15px; border-radius: 8px; transition: 0.3s; }
			.nav-links a:hover { background: rgba(108,99,255,0.2); color: #fff; }
			.nav-links a.active { background: rgba(108,99,255,0.2); color: #6c63ff; }
			.nav-links a.logout { color: #ff4757; }
			.container { max-width: 1400px; margin: 30px auto; padding: 0 30px; }
			.stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 30px; }
			.stat-card { background: rgba(255,255,255,0.05); padding: 20px; border-radius: 15px; border: 1px solid rgba(255,255,255,0.05); text-align: center; }
			.stat-card .number { font-size: 32px; font-weight: bold; color: #6c63ff; }
			.stat-card .label { color: #888; font-size: 14px; margin-top: 5px; }
			.stat-card .icon { font-size: 24px; color: #6c63ff; margin-bottom: 10px; }
			.section { background: rgba(255,255,255,0.05); padding: 25px; border-radius: 15px; border: 1px solid rgba(255,255,255,0.05); margin-bottom: 30px; }
			.section h2 { color: #6c63ff; margin-bottom: 20px; font-size: 20px; }
			.section h2 i { margin-right: 10px; }
			.form-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; }
			.form-row input { padding: 12px 15px; background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.1); border-radius: 10px; color: #fff; font-size: 14px; }
			.form-row input:focus { outline: none; border-color: #6c63ff; }
			.btn-primary { padding: 12px 25px; background: linear-gradient(135deg, #6c63ff, #5a52d5); border: none; border-radius: 10px; color: #fff; font-weight: bold; cursor: pointer; transition: 0.3s; }
			.btn-primary:hover { transform: scale(1.02); box-shadow: 0 5px 25px rgba(108,99,255,0.4); }
			.btn-danger { padding: 8px 15px; background: #ff4757; border: none; border-radius: 8px; color: #fff; cursor: pointer; font-size: 12px; }
			.user-table { width: 100%; border-collapse: collapse; margin-top: 15px; }
			.user-table th { text-align: left; padding: 12px 15px; border-bottom: 1px solid rgba(255,255,255,0.05); color: #888; font-weight: normal; font-size: 13px; text-transform: uppercase; letter-spacing: 1px; }
			.user-table td { padding: 12px 15px; border-bottom: 1px solid rgba(255,255,255,0.03); color: #ddd; }
			.user-table tr:hover { background: rgba(255,255,255,0.02); }
			.badge { padding: 4px 12px; border-radius: 20px; font-size: 12px; }
			.badge.active { background: rgba(46,213,115,0.2); color: #2ed573; }
			.badge.inactive { background: rgba(255,71,87,0.2); color: #ff4757; }
			.link { color: #6c63ff; text-decoration: none; }
			.link:hover { text-decoration: underline; }
			@media (max-width: 768px) { .nav-links a { margin-left: 10px; font-size: 13px; padding: 6px 10px; } .container { padding: 0 15px; } }
		</style>
	</head>
	<body>
	<div class="navbar">
		<h1><i class="fas fa-crown" style="color:#6c63ff;"></i> گنگ <span>نت پنل</span></h1>
		<div class="nav-links">
			<a href="/" class="active"><i class="fas fa-home"></i> داشبورد</a>
			<a href="/settings"><i class="fas fa-cog"></i> تنظیمات</a>
			<a href="/admin"><i class="fas fa-user-shield"></i> مدیریت</a>
			<a href="/logout" class="logout"><i class="fas fa-sign-out-alt"></i> خروج</a>
		</div>
	</div>
	<div class="container">
		<div class="stats-grid">
			<div class="stat-card"><div class="icon"><i class="fas fa-users"></i></div><div class="number">{{.Stats.TotalUsers}}</div><div class="label">کل کاربران</div></div>
			<div class="stat-card"><div class="icon"><i class="fas fa-user-check"></i></div><div class="number">{{.Stats.ActiveUsers}}</div><div class="label">کاربران فعال</div></div>
			<div class="stat-card"><div class="icon"><i class="fas fa-database"></i></div><div class="number">{{.Stats.TotalTraffic}}</div><div class="label">کل ترافیک (بایت)</div></div>
			<div class="stat-card"><div class="icon"><i class="fas fa-server"></i></div><div class="number">گنگ نت</div><div class="label">وضعیت سرور</div></div>
		</div>

		<div class="section">
			<h2><i class="fas fa-user-plus"></i> ساخت کاربر جدید</h2>
			<form action="/create-user" method="POST" class="form-row">
				<input name="email" placeholder="📧 ایمیل کاربر" required>
				<input name="traffic" placeholder="📊 حجم (گیگابایت)" type="number" required>
				<input name="expiry" placeholder="📅 انقضا (2025-12-31)" required>
				<button type="submit" class="btn-primary"><i class="fas fa-plus"></i> ساخت</button>
			</form>
		</div>

		<div class="section">
			<h2><i class="fas fa-list"></i> لیست کاربران</h2>
			<table class="user-table">
				<thead><tr><th>UUID</th><th>حجم</th><th>انقضا</th><th>وضعیت</th><th>لینک ساب</th></tr></thead>
				<tbody>
					{{range .Users}}
					<tr>
						<td style="font-family: monospace; font-size: 13px;">{{.UUID}}</td>
						<td>{{.Traffic}} بایت</td>
						<td>{{.Expiry}}</td>
						<td>{{if .IsActive}}<span class="badge active">فعال</span>{{else}}<span class="badge inactive">غیرفعال</span>{{end}}</td>
						<td><a href="/sub/{{.UUID}}" class="link"><i class="fas fa-link"></i> دریافت</a></td>
					</tr>
					{{else}}
					<tr><td colspan="5" style="text-align:center; color:#555; padding:30px;">هیچ کاربری ساخته نشده است</td></tr>
					{{end}}
				</tbody>
			</table>
		</div>
	</div>
	</body>
	</html>`
	t := template.Must(template.New("index").Parse(tmpl))
	data := struct {
		Users []User
		Stats Stats
	}{users, stats}
	mu.Lock()
	t.Execute(w, data)
	mu.Unlock()
}

// ===== بقیه هندلرها (همون‌های قبلی با تغییرات جزئی) =====
func adminHandler(w http.ResponseWriter, r *http.Request) {
	// همون کد قبلی
	if r.Method == http.MethodPost {
		r.ParseForm()
		newUsername := r.FormValue("username")
		newPassword := r.FormValue("password")
		if newUsername != "" && newPassword != "" {
			mu.Lock()
			admin.Username = newUsername
			admin.Password = newPassword
			mu.Unlock()
		}
		http.Redirect(w, r, "/admin?success=1", http.StatusSeeOther)
		return
	}
	tmpl := `<!DOCTYPE html>
	<html>
	<head><title>مدیریت حساب - گنگ نت</title>
	<style>
		body { font-family: 'Segoe UI', Tahoma, sans-serif; background: #0f0c29; color: #fff; padding: 20px; }
		.container { max-width: 600px; margin: 0 auto; background: rgba(255,255,255,0.05); padding: 30px; border-radius: 15px; border: 1px solid rgba(255,255,255,0.05); }
		h1 { color: #6c63ff; }
		form { margin-top: 20px; }
		input { width: 100%; padding: 12px; margin: 10px 0; background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.1); border-radius: 10px; color: #fff; }
		button { padding: 12px 25px; background: linear-gradient(135deg, #6c63ff, #5a52d5); border: none; border-radius: 10px; color: #fff; font-weight: bold; cursor: pointer; }
		.nav a { color: #aaa; text-decoration: none; margin-right: 20px; }
		.success { color: #2ed573; }
	</style>
	</head>
	<body>
	<div class="container">
		<h1>👤 مدیریت حساب</h1>
		<div class="nav"><a href="/">📋 داشبورد</a><a href="/settings">⚙️ تنظیمات</a><a href="/logout">🚪 خروج</a></div>
		{{if .}}<div class="success">✅ اطلاعات با موفقیت تغییر کرد!</div>{{end}}
		<form method="POST">
			<input name="username" placeholder="نام کاربری جدید" value="{{.Username}}">
			<input name="password" type="password" placeholder="رمز عبور جدید">
			<button type="submit">به‌روزرسانی</button>
		</form>
		<div style="margin-top:20px; color:#555;">نام کاربری فعلی: <strong style="color:#fff;">{{.Username}}</strong></div>
	</div>
	</body>
	</html>`
	t := template.Must(template.New("admin").Parse(tmpl))
	mu.Lock()
	data := struct {
		Username string
		Success  bool
	}{admin.Username, r.URL.Query().Get("success") == "1"}
	t.Execute(w, data)
	mu.Unlock()
}

func settingsHandler(w http.ResponseWriter, r *http.Request) {
	// همون کد قبلی با استایل جدید
	tmpl := `<!DOCTYPE html>
	<html>
	<head><title>تنظیمات - گنگ نت</title>
	<style>
		body { font-family: 'Segoe UI', Tahoma, sans-serif; background: #0f0c29; color: #fff; padding: 20px; }
		.container { max-width: 800px; margin: 0 auto; background: rgba(255,255,255,0.05); padding: 30px; border-radius: 15px; border: 1px solid rgba(255,255,255,0.05); }
		h1 { color: #6c63ff; }
		form { margin-top: 20px; }
		input { width: 100%; padding: 12px; margin: 10px 0; background: rgba(255,255,255,0.05); border: 1px solid rgba(255,255,255,0.1); border-radius: 10px; color: #fff; }
		button { padding: 12px 25px; background: linear-gradient(135deg, #6c63ff, #5a52d5); border: none; border-radius: 10px; color: #fff; font-weight: bold; cursor: pointer; }
		.nav a { color: #aaa; text-decoration: none; margin-right: 20px; }
		.info { background: rgba(255,255,255,0.03); padding: 15px; border-radius: 10px; margin: 10px 0; color: #888; }
	</style>
	</head>
	<body>
	<div class="container">
		<h1>⚙️ تنظیمات</h1>
		<div class="nav"><a href="/">📋 داشبورد</a><a href="/admin">👤 مدیریت</a><a href="/logout">🚪 خروج</a></div>
		<h2 style="color:#888; font-size:16px; margin-top:20px;">🔗 تغییر دامنه لینک ساب</h2>
		<form action="/update-domain" method="POST">
			<input name="domain" placeholder="دامنه جدید (مثلاً gngnet.ir)" value="{{.Domain}}" required>
			<button type="submit">به‌روزرسانی</button>
		</form>
		<div class="info">دامنه فعلی: <strong style="color:#fff;">{{.Domain}}</strong></div>
	</div>
	</body>
	</html>`
	t := template.Must(template.New("settings").Parse(tmpl))
	mu.Lock()
	t.Execute(w, settings)
	mu.Unlock()
}

// ===== API برای دریافت لیست کاربران (JSON) =====
func apiUsersHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// ===== API برای آمار =====
func apiStatsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	stats.TotalUsers = len(users)
	active := 0
	for _, u := range users {
		if u.IsActive {
			active++
		}
	}
	stats.ActiveUsers = active
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// ===== بقیه هندلرها (همون‌های قبلی) =====
func updateDomainHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	newDomain := r.FormValue("domain")
	if newDomain != "" {
		mu.Lock()
		settings.Domain = newDomain
		mu.Unlock()
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	email := r.FormValue("email")
	trafficGB := r.FormValue("traffic")
	expiryStr := r.FormValue("expiry")

	traffic, _ := strconv.ParseInt(trafficGB, 10, 64)
	trafficBytes := traffic * 1024 * 1024 * 1024
	expiry, _ := time.Parse("2006-01-02", expiryStr)

	mu.Lock()
	defer mu.Unlock()
	newUser := User{
		ID:       fmt.Sprintf("user-%d", len(users)+1),
		UUID:     uuid.New().String(),
		Traffic:  trafficBytes,
		Expiry:   expiry,
		IsActive: true,
		Email:    email,
	}
	users = append(users, newUser)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func subHandler(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Path[len("/sub/"):]
	mu.Lock()
	defer mu.Unlock()
	for _, user := range users {
		if user.UUID == uuid && user.IsActive {
			config := fmt.Sprintf("vless://%s@%s:443?encryption=none&security=tls&sni=%s&fp=chrome&type=tcp#GangNet", user.UUID, settings.Domain, settings.Domain)
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, config)
			return
		}
	}
	http.Error(w, "User not found or inactive", http.StatusNotFound)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
