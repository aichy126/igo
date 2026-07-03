package db

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func sqliteViper(t *testing.T) *viper.Viper {
	t.Helper()
	v := viper.New()
	v.Set("sqlite.test", map[string]any{
		"data_source": filepath.Join(t.TempDir(), "test.db"),
	})
	return v
}

// TestNewSqlite 验证 sqlite 正常初始化
func TestNewSqlite(t *testing.T) {
	m, err := New(sqliteViper(t))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer m.Close()

	dm := m.Get("test")
	if dm == nil || dm.WriteDB == nil {
		t.Fatal("Get(test) 应返回可用的 DatabaseManager")
	}
	if err := dm.Ping(); err != nil {
		t.Errorf("Ping error: %v", err)
	}
	if names := m.Names(); len(names) != 1 || names[0] != "test" {
		t.Errorf("Names() = %v", names)
	}
	for name, err := range m.PingAll() {
		if err != nil {
			t.Errorf("PingAll[%s] error: %v", name, err)
		}
	}
}

// TestNewEmptyConfig 验证完全没配置数据库时返回空 manager 且不报错
func TestNewEmptyConfig(t *testing.T) {
	m, err := New(viper.New())
	if err != nil {
		t.Fatalf("空配置 New() 不应报错: %v", err)
	}
	if m.Get("whatever") != nil {
		t.Error("空 manager Get 应返回 nil")
	}
}

// TestNewBadConfigFailFast 验证配置了但连不上的数据库返回错误(不再静默或 panic)
func TestNewBadConfigFailFast(t *testing.T) {
	v := viper.New()
	v.Set("mysql.bad", map[string]any{
		"data_source": "root:wrong@tcp(127.0.0.1:1)/nope?timeout=200ms",
	})
	_, err := New(v)
	if err == nil {
		t.Fatal("连不上的 mysql 配置应返回错误")
	}
	if !strings.Contains(err.Error(), "bad") {
		t.Errorf("错误信息应包含配置名: %v", err)
	}
}

// TestNewDBTablePanicMessage 验证配置名写错时 panic 且信息明确
func TestNewDBTablePanicMessage(t *testing.T) {
	m, err := New(sqliteViper(t))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer m.Close()
	d := &DB{DBResourceManager: m}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("不存在的 dbname 应 panic")
		}
		if !strings.Contains(r.(string), "not-exist") {
			t.Errorf("panic 信息应包含配置名: %v", r)
		}
	}()
	d.NewDBTable("not-exist", "table")
}

// TestRepoCRUD 验证 Repo 基本读写和 SetTableName
func TestRepoCRUD(t *testing.T) {
	m, err := New(sqliteViper(t))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer m.Close()
	d := &DB{DBResourceManager: m}

	type User struct {
		Id   int64  `xorm:"pk autoincr"`
		Name string `xorm:"varchar(64)"`
	}

	repo := d.NewDBTable("test", "user")
	if err := repo.Engine.Sync2(new(User)); err != nil {
		t.Fatalf("Sync2 error: %v", err)
	}

	if _, err := repo.InsertOne(&User{Name: "alice"}); err != nil {
		t.Fatalf("InsertOne error: %v", err)
	}

	got := new(User)
	has, err := repo.Where("name = ?", "alice").Get(got)
	if err != nil || !has {
		t.Fatalf("查询失败: has=%v err=%v", has, err)
	}

	// SetTableName 修复验证:改名后 session 应查询新表
	repo2 := d.NewDBTable("test", "wrong_table")
	repo2.SetTableName("user")
	has, err = repo2.Where("name = ?", "alice").Get(new(User))
	if err != nil || !has {
		t.Errorf("SetTableName 后查询失败: has=%v err=%v", has, err)
	}
}
