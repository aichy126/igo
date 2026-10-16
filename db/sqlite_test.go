package db

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"
	"xorm.io/xorm"
)

func TestSqliteDSNDefaults(t *testing.T) {
	got := sqliteDSN("./data/app.db")
	for _, want := range []string{"busy_timeout(5000)", "journal_mode(WAL)", "_txlock=immediate", "_time_format=sqlite"} {
		if !strings.Contains(got, want) {
			t.Errorf("sqliteDSN 缺少 %q: %s", want, got)
		}
	}
	if strings.Count(got, "?") != 1 {
		t.Errorf("应只有一个 '?': %s", got)
	}
}

// 写了的参数一律尊重，只补没写的
func TestSqliteDSNKeepsUserParams(t *testing.T) {
	got := sqliteDSN("file:app.db?_pragma=journal_mode(DELETE)&_txlock=deferred")
	if strings.Contains(got, "WAL") || strings.Contains(got, "immediate") {
		t.Errorf("不应覆盖用户写的参数: %s", got)
	}
	if !strings.Contains(got, "busy_timeout(5000)") || !strings.Contains(got, "_time_format=sqlite") {
		t.Errorf("没写的参数应补上: %s", got)
	}
}

func TestSqliteEnsureDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "b")
	v := viper.New()
	v.Set("sqlite.test", map[string]any{"data_source": filepath.Join(dir, "app.db")})
	m, err := New(v)
	if err != nil {
		t.Fatalf("目录不存在时应自动创建: %v", err)
	}
	defer m.Close()
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("目录未创建: %v", err)
	}

	for _, ds := range []string{":memory:", "file::memory:?cache=shared", "file:x?mode=memory", "app.db"} {
		if err := sqliteEnsureDir(ds); err != nil {
			t.Errorf("sqliteEnsureDir(%q) = %v", ds, err)
		}
	}
}

type tsRow struct {
	Id        int64     `xorm:"pk autoincr"`
	Name      string    `xorm:"VARCHAR(50)"`
	UpdatedAt time.Time `xorm:"DATETIME"`
}

// map 更新把 time.Time 原值交给驱动：必须写成 SQLite 能识别的格式，并能读回
func TestSqliteMapUpdateTime(t *testing.T) {
	m, err := New(sqliteViper(t))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	e := m.Get("test").WriteDB
	if err := e.Sync2(new(tsRow)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Insert(&tsRow{Name: "a", UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if _, err := e.Table(new(tsRow)).Where("name = ?", "a").Update(map[string]any{"updated_at": now}); err != nil {
		t.Fatal(err)
	}

	var raw string
	if _, err := e.SQL("SELECT CAST(updated_at AS TEXT) FROM ts_row").Get(&raw); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "m=") || strings.Contains(raw, " CST") || strings.Contains(raw, " UTC") {
		t.Errorf("时间写成了 time.String() 格式: %q", raw)
	}
	var cnt int64
	if _, err := e.SQL("SELECT count(*) FROM ts_row WHERE datetime(updated_at) IS NOT NULL").Get(&cnt); err != nil || cnt != 1 {
		t.Errorf("SQLite 日期函数应能解析写入的时间 %q (cnt=%d err=%v)", raw, cnt, err)
	}

	got := new(tsRow)
	if _, err := e.Where("name = ?", "a").Get(got); err != nil {
		t.Fatal(err)
	}
	if d := got.UpdatedAt.Sub(now); d > time.Second || d < -time.Second {
		t.Errorf("读回的时间 %v 与写入的 %v 不符", got.UpdatedAt, now)
	}
}

// 事务里先读后写：deferred 事务在 WAL 下升级写锁会直接 SQLITE_BUSY，immediate 不会
func TestSqliteReadThenWriteTxConcurrent(t *testing.T) {
	v := viper.New()
	v.Set("sqlite.test", map[string]any{
		"data_source": filepath.Join(t.TempDir(), "tx.db"),
		"max_open":    8,
	})
	m, err := New(v)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	d := &DB{m}
	e := m.Get("test").WriteDB
	if err := e.Sync2(new(tsRow)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Insert(&tsRow{Name: "c", UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 64)
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- d.Transaction("test", func(sess *xorm.Session) error {
				row := new(tsRow)
				if _, err := sess.Where("name = ?", "c").Get(row); err != nil {
					return err
				}
				_, err := sess.ID(row.Id).Cols("updated_at").Update(&tsRow{UpdatedAt: time.Now()})
				return err
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("并发「先读后写」事务失败: %v", err)
		}
	}
}
